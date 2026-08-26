// Package msauth implements the sign in chain Minecraft requires: a Microsoft
// device code grant, then Xbox Live, then XSTS, then an exchange for a
// Minecraft token and a profile lookup. Four hops, each with its own failure
// mode and its own unhelpful error message.
package msauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	deviceCodeEndpoint = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	tokenEndpoint      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

	// Without XboxLive.signin the Xbox endpoint fails in a way that gives no
	// hint about what is missing.
	scope = "XboxLive.signin offline_access"
)

type Config struct {
	ClientID string
	HTTP     *http.Client

	DeviceCodeURL string
	TokenURL      string

	// After exists so tests can drive the polling loop without waiting in real
	// time. Nil means time.After.
	After func(time.Duration) <-chan time.Time
}

func (c *Config) after(d time.Duration) <-chan time.Time {
	if c.After != nil {
		return c.After(d)
	}
	return time.After(d)
}

func (c *Config) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Config) deviceCodeURL() string {
	if c.DeviceCodeURL != "" {
		return c.DeviceCodeURL
	}
	return deviceCodeEndpoint
}

func (c *Config) tokenURL() string {
	if c.TokenURL != "" {
		return c.TokenURL
	}
	return tokenEndpoint
}

// DeviceCode is what the user needs in order to approve the sign in on another
// device.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

func (d DeviceCode) Expiry(now time.Time) time.Time {
	return now.Add(time.Duration(d.ExpiresIn) * time.Second)
}

func (d DeviceCode) PollInterval() time.Duration {
	if d.Interval <= 0 {
		return 5 * time.Second
	}
	return time.Duration(d.Interval) * time.Second
}

type oauthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e oauthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// Errors the caller needs to distinguish, since they mean very different things
// to someone standing in front of the machine.
var (
	ErrDeclined   = fmt.Errorf("the sign in was declined")
	ErrExpired    = fmt.Errorf("the device code expired before it was approved")
	ErrNoClientID = fmt.Errorf("no microsoft client id is configured")
)

// RequestDeviceCode starts a sign in and returns the code to display.
func (c *Config) RequestDeviceCode(ctx context.Context) (*DeviceCode, error) {
	if c.ClientID == "" {
		return nil, ErrNoClientID
	}

	form := url.Values{"client_id": {c.ClientID}, "scope": {scope}}
	var code DeviceCode
	if err := c.postForm(ctx, c.deviceCodeURL(), form, &code); err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	if code.DeviceCode == "" || code.UserCode == "" {
		return nil, fmt.Errorf("request device code: response was missing a code")
	}
	return &code, nil
}

type MicrosoftToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// PollForToken blocks until the user approves on their phone, the code expires,
// or the context is cancelled. Microsoft asks pollers to back off when told to,
// and ignoring that gets the request throttled.
func (c *Config) PollForToken(ctx context.Context, code *DeviceCode) (*MicrosoftToken, error) {
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {c.ClientID},
		"device_code": {code.DeviceCode},
	}

	interval := code.PollInterval()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.after(interval):
		}

		var token MicrosoftToken
		err := c.postForm(ctx, c.tokenURL(), form, &token)
		if err == nil {
			return &token, nil
		}

		var oe oauthError
		if !asOAuthError(err, &oe) {
			return nil, err
		}

		switch oe.Code {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
		case "authorization_declined":
			return nil, ErrDeclined
		case "expired_token", "code_expired":
			return nil, ErrExpired
		default:
			return nil, err
		}
	}
}

func (c *Config) postForm(ctx context.Context, endpoint string, form url.Values, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body := json.NewDecoder(resp.Body)
	if resp.StatusCode >= 400 {
		var oe oauthError
		if err := body.Decode(&oe); err != nil || oe.Code == "" {
			return fmt.Errorf("%s returned %s", endpoint, resp.Status)
		}
		return oe
	}
	return body.Decode(into)
}

func asOAuthError(err error, target *oauthError) bool {
	oe, ok := err.(oauthError)
	if ok {
		*target = oe
	}
	return ok
}
