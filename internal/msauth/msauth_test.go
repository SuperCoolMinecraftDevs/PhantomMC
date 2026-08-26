package msauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func TestRequestDeviceCodeNeedsAClientID(t *testing.T) {
	_, err := (&Config{}).RequestDeviceCode(context.Background())
	if !errors.Is(err, ErrNoClientID) {
		t.Fatalf("expected ErrNoClientID, got %v", err)
	}
}

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("client_id") != "test-client" {
			t.Errorf("client id not sent: %v", r.Form)
		}
		if !strings.Contains(r.Form.Get("scope"), "XboxLive.signin") {
			t.Errorf("XboxLive.signin scope missing, the xbox hop will fail unhelpfully: %v", r.Form)
		}
		writeJSON(w, 200, map[string]any{
			"device_code":      "dev-123",
			"user_code":        "F8B2X9AQ",
			"verification_uri": "https://microsoft.com/link",
			"expires_in":       900,
			"interval":         1,
		})
	}))
	defer server.Close()

	cfg := &Config{ClientID: "test-client", HTTP: server.Client(), DeviceCodeURL: server.URL}
	code, err := cfg.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if code.UserCode != "F8B2X9AQ" {
		t.Errorf("unexpected user code %q", code.UserCode)
	}
	if code.PollInterval() != time.Second {
		t.Errorf("unexpected interval %s", code.PollInterval())
	}
}

func TestPollWaitsThroughPendingThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			writeJSON(w, 400, map[string]string{"error": "authorization_pending"})
			return
		}
		writeJSON(w, 200, map[string]any{"access_token": "ms-token", "expires_in": 3600})
	}))
	defer server.Close()

	var waits []time.Duration
	cfg := &Config{ClientID: "c", HTTP: server.Client(), TokenURL: server.URL,
		After: instantClock(&waits)}
	token, err := cfg.PollForToken(context.Background(), &DeviceCode{DeviceCode: "d", Interval: 0})
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "ms-token" {
		t.Errorf("unexpected token %q", token.AccessToken)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 polls, got %d", calls.Load())
	}
}

func TestPollDistinguishesTerminalErrors(t *testing.T) {
	cases := []struct {
		oauthCode string
		want      error
	}{
		{"authorization_declined", ErrDeclined},
		{"expired_token", ErrExpired},
	}

	for _, tc := range cases {
		t.Run(tc.oauthCode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, 400, map[string]string{"error": tc.oauthCode})
			}))
			defer server.Close()

			var waits []time.Duration
			cfg := &Config{ClientID: "c", HTTP: server.Client(), TokenURL: server.URL,
				After: instantClock(&waits)}
			_, err := cfg.PollForToken(context.Background(), &DeviceCode{DeviceCode: "d"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

// instantClock records what the loop asked to wait for and returns immediately,
// so backoff can be asserted without spending it.
func instantClock(waits *[]time.Duration) func(time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		*waits = append(*waits, d)
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
}

func TestPollBacksOffWhenToldTo(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			writeJSON(w, 400, map[string]string{"error": "slow_down"})
		case 2:
			writeJSON(w, 400, map[string]string{"error": "slow_down"})
		default:
			writeJSON(w, 200, map[string]any{"access_token": "ms-token"})
		}
	}))
	defer server.Close()

	var waits []time.Duration
	cfg := &Config{ClientID: "c", HTTP: server.Client(), TokenURL: server.URL,
		After: instantClock(&waits)}

	if _, err := cfg.PollForToken(context.Background(), &DeviceCode{DeviceCode: "d", Interval: 5}); err != nil {
		t.Fatal(err)
	}

	if len(waits) != 3 {
		t.Fatalf("expected 3 waits, got %v", waits)
	}
	if waits[0] != 5*time.Second {
		t.Errorf("first wait should be the advertised interval, got %s", waits[0])
	}
	if waits[1] <= waits[0] || waits[2] <= waits[1] {
		t.Errorf("each slow_down must lengthen the wait: %v", waits)
	}
}

func TestPollRespectsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 400, map[string]string{"error": "authorization_pending"})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := &Config{ClientID: "c", HTTP: server.Client(), TokenURL: server.URL}
	_, err := cfg.PollForToken(ctx, &DeviceCode{DeviceCode: "d", Interval: 1})
	if err == nil {
		t.Fatal("expected cancellation to end the poll")
	}
}

// fakeChain stands in for the four hop sign in.
func fakeChain(t *testing.T, profileStatus int) (*Config, Endpoints, func()) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/xbl", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		props := body["Properties"].(map[string]any)
		if ticket := props["RpsTicket"].(string); !strings.HasPrefix(ticket, "d=") {
			t.Errorf("rps ticket must be prefixed with d=, got %q", ticket)
		}
		writeJSON(w, 200, map[string]any{
			"Token":         "xbl-token",
			"DisplayClaims": map[string]any{"xui": []map[string]string{{"uhs": "userhash"}}},
		})
	})

	mux.HandleFunc("/xsts", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["RelyingParty"] != minecraftRelyingParty {
			t.Errorf("wrong relying party: %v", body["RelyingParty"])
		}
		writeJSON(w, 200, map[string]any{
			"Token":         "xsts-token",
			"DisplayClaims": map[string]any{"xui": []map[string]string{{"uhs": "userhash"}}},
		})
	})

	mux.HandleFunc("/mclogin", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if want := "XBL3.0 x=userhash;xsts-token"; body["identityToken"] != want {
			t.Errorf("identity token malformed: %q", body["identityToken"])
		}
		writeJSON(w, 200, map[string]string{"access_token": "mc-token"})
	})

	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mc-token" {
			t.Errorf("profile call not authorized: %q", r.Header.Get("Authorization"))
		}
		if profileStatus != 200 {
			w.WriteHeader(profileStatus)
			return
		}
		writeJSON(w, 200, map[string]string{
			"id":   "069a79f444e94726a5befca90e38aaf5",
			"name": "Notch",
		})
	})

	server := httptest.NewServer(mux)
	cfg := &Config{ClientID: "c", HTTP: server.Client()}
	endpoints := Endpoints{
		XBL:       server.URL + "/xbl",
		XSTS:      server.URL + "/xsts",
		MCLogin:   server.URL + "/mclogin",
		MCProfile: server.URL + "/profile",
	}
	return cfg, endpoints, server.Close
}

func TestAuthenticateWalksTheWholeChain(t *testing.T) {
	cfg, endpoints, done := fakeChain(t, 200)
	defer done()

	profile, err := cfg.Authenticate(context.Background(), endpoints, &MicrosoftToken{AccessToken: "ms"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Notch" {
		t.Errorf("unexpected name %q", profile.Name)
	}
	if profile.UUID != "069a79f4-44e9-4726-a5be-fca90e38aaf5" {
		t.Errorf("uuid not dashed: %q", profile.UUID)
	}
	if profile.AccessToken != "mc-token" {
		t.Errorf("access token not carried: %q", profile.AccessToken)
	}
}

func TestProfile404MeansNoEntitlement(t *testing.T) {
	cfg, endpoints, done := fakeChain(t, 404)
	defer done()

	_, err := cfg.Authenticate(context.Background(), endpoints, &MicrosoftToken{AccessToken: "ms"})
	if !errors.Is(err, ErrNoEntitlement) {
		t.Fatalf("a 404 profile is the ownership check, got %v", err)
	}
}

func TestXSTSErrorsAreTranslated(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/xbl", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"Token":         "xbl-token",
			"DisplayClaims": map[string]any{"xui": []map[string]string{{"uhs": "userhash"}}},
		})
	})
	mux.HandleFunc("/xsts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, map[string]any{"XErr": 2148916233, "Message": ""})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := &Config{ClientID: "c", HTTP: server.Client()}
	_, err := cfg.Authenticate(context.Background(), Endpoints{
		XBL:  server.URL + "/xbl",
		XSTS: server.URL + "/xsts",
	}, &MicrosoftToken{AccessToken: "ms"})

	if err == nil || !strings.Contains(err.Error(), "no Xbox profile") {
		t.Fatalf("opaque XErr should become readable advice, got %v", err)
	}
}

func TestDashUUID(t *testing.T) {
	cases := map[string]string{
		"069a79f444e94726a5befca90e38aaf5":     "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		"069a79f4-44e9-4726-a5be-fca90e38aaf5": "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		"short":                                "short",
	}
	for input, want := range cases {
		if got := dashUUID(input); got != want {
			t.Errorf("dashUUID(%q) = %q, want %q", input, got, want)
		}
	}
}
