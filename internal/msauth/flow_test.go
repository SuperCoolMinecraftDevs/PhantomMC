package msauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeMicrosoft speaks the whole protocol the way the real endpoints do,
// including making the caller poll while the user is still approving. It exists
// so the complete flow can be exercised without a real account or a real
// application id.
func fakeMicrosoft(t *testing.T) (*Config, Endpoints, *atomic.Bool, func()) {
	t.Helper()

	var approved atomic.Bool
	mux := http.NewServeMux()

	mux.HandleFunc("/devicecode", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("client_id") == "" {
			writeJSON(w, 400, map[string]string{"error": "invalid_request"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"device_code":      "device-abc",
			"user_code":        "F8B2-X9AQ",
			"verification_uri": "https://microsoft.com/link",
			"expires_in":       900,
			"interval":         5,
		})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if !approved.Load() {
			writeJSON(w, 400, map[string]string{"error": "authorization_pending"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"access_token":  "ms-access-token",
			"refresh_token": "ms-refresh-token",
			"expires_in":    3600,
		})
	})

	mux.HandleFunc("/xbl", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		props := body["Properties"].(map[string]any)
		if props["RpsTicket"] != "d=ms-access-token" {
			t.Errorf("xbox live got the wrong ticket: %v", props["RpsTicket"])
		}
		writeJSON(w, 200, map[string]any{
			"Token":         "xbl-token",
			"DisplayClaims": map[string]any{"xui": []map[string]string{{"uhs": "1234567890"}}},
		})
	})

	mux.HandleFunc("/xsts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"Token":         "xsts-token",
			"DisplayClaims": map[string]any{"xui": []map[string]string{{"uhs": "1234567890"}}},
		})
	})

	mux.HandleFunc("/mclogin", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"access_token": "minecraft-access-token"})
	})

	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{
			"id": "853c80ef3c3749fdaa49938b674adae6", "name": "jeb_",
		})
	})

	server := httptest.NewServer(mux)
	cfg := &Config{
		ClientID:      "test-application-id",
		HTTP:          server.Client(),
		DeviceCodeURL: server.URL + "/devicecode",
		TokenURL:      server.URL + "/token",
	}
	endpoints := Endpoints{
		XBL:       server.URL + "/xbl",
		XSTS:      server.URL + "/xsts",
		MCLogin:   server.URL + "/mclogin",
		MCProfile: server.URL + "/profile",
	}
	return cfg, endpoints, &approved, server.Close
}

func TestCompleteSignInFlow(t *testing.T) {
	cfg, endpoints, approved, done := fakeMicrosoft(t)
	defer done()

	var waits []time.Duration
	cfg.After = instantClock(&waits)

	code, err := cfg.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if code.UserCode != "F8B2-X9AQ" {
		t.Fatalf("unexpected code %q", code.UserCode)
	}

	// The user is still reaching for their phone. Polling must not give up.
	go func() {
		time.Sleep(20 * time.Millisecond)
		approved.Store(true)
	}()

	token, err := cfg.PollForToken(context.Background(), code)
	if err != nil {
		t.Fatal(err)
	}
	if token.RefreshToken == "" {
		t.Error("refresh token not captured")
	}

	profile, err := cfg.Authenticate(context.Background(), endpoints, token)
	if err != nil {
		t.Fatal(err)
	}

	if profile.Name != "jeb_" {
		t.Errorf("unexpected username %q", profile.Name)
	}
	if profile.UUID != "853c80ef-3c37-49fd-aa49-938b674adae6" {
		t.Errorf("uuid not dashed: %q", profile.UUID)
	}
	if profile.AccessToken != "minecraft-access-token" {
		t.Errorf("wrong token carried through: %q", profile.AccessToken)
	}
	if len(waits) < 2 {
		t.Errorf("expected the loop to poll more than once, waits: %v", waits)
	}
}

// countingClock approves after a fixed number of polls, driven from inside the
// loop itself so the test is deterministic rather than racing a goroutine.
func countingClock(approveAfter int, approved *atomic.Bool, calls *int) func(time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		*calls++
		if *calls >= approveAfter {
			approved.Store(true)
		}
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
}

func TestSignInSurvivesASlowUser(t *testing.T) {
	cfg, endpoints, approved, done := fakeMicrosoft(t)
	defer done()

	var calls int
	cfg.After = countingClock(25, approved, &calls)

	code, err := cfg.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	token, err := cfg.PollForToken(context.Background(), code)
	if err != nil {
		t.Fatalf("polling gave up on a user who took their time: %v", err)
	}
	if calls < 25 {
		t.Errorf("expected at least 25 polls, got %d", calls)
	}
	if _, err := cfg.Authenticate(context.Background(), endpoints, token); err != nil {
		t.Fatal(err)
	}
}
