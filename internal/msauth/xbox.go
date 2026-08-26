package msauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	xblEndpoint       = "https://user.auth.xboxlive.com/user/authenticate"
	xstsEndpoint      = "https://xsts.auth.xboxlive.com/xsts/authorize"
	mcLoginEndpoint   = "https://api.minecraftservices.com/authentication/login_with_xbox"
	mcProfileEndpoint = "https://api.minecraftservices.com/minecraft/profile"

	minecraftRelyingParty = "rp://api.minecraftservices.com/"
)

type Endpoints struct {
	XBL       string
	XSTS      string
	MCLogin   string
	MCProfile string
}

func (e Endpoints) or(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// Profile is the end of the chain: who the player is, and the token the game
// needs to talk to session servers.
type Profile struct {
	Name        string
	UUID        string
	AccessToken string
}

// XSTS refuses in ways that need translating, because the raw response is an
// opaque numeric code and a blank message.
var xstsReasons = map[string]string{
	"2148916233": "this Microsoft account has no Xbox profile, create one at xbox.com and try again",
	"2148916235": "Xbox Live is not available in this account's region",
	"2148916236": "this account needs adult verification",
	"2148916237": "this account needs adult verification",
	"2148916238": "this is a child account and must be added to a family before it can sign in",
}

type xboxResponse struct {
	Token         string `json:"Token"`
	DisplayClaims struct {
		XUI []struct {
			UHS string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

func (r xboxResponse) userHash() (string, error) {
	if len(r.DisplayClaims.XUI) == 0 || r.DisplayClaims.XUI[0].UHS == "" {
		return "", fmt.Errorf("response carried no user hash")
	}
	return r.DisplayClaims.XUI[0].UHS, nil
}

// Authenticate walks Xbox Live, XSTS, the Minecraft token exchange and the
// profile lookup.
func (c *Config) Authenticate(ctx context.Context, e Endpoints, token *MicrosoftToken) (*Profile, error) {
	xbl, err := c.xboxLive(ctx, e, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("xbox live: %w", err)
	}

	xsts, userHash, err := c.xsts(ctx, e, xbl.Token)
	if err != nil {
		return nil, fmt.Errorf("xsts: %w", err)
	}

	mcToken, err := c.minecraftToken(ctx, e, userHash, xsts)
	if err != nil {
		return nil, fmt.Errorf("minecraft login: %w", err)
	}

	profile, err := c.profile(ctx, e, mcToken)
	if err != nil {
		return nil, err
	}
	profile.AccessToken = mcToken
	return profile, nil
}

func (c *Config) xboxLive(ctx context.Context, e Endpoints, msToken string) (*xboxResponse, error) {
	body := map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}

	var out xboxResponse
	if err := c.postJSON(ctx, e.or(e.XBL, xblEndpoint), body, "", &out); err != nil {
		return nil, err
	}
	if out.Token == "" {
		return nil, fmt.Errorf("no token returned")
	}
	return &out, nil
}

func (c *Config) xsts(ctx context.Context, e Endpoints, xblToken string) (string, string, error) {
	body := map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xblToken},
		},
		"RelyingParty": minecraftRelyingParty,
		"TokenType":    "JWT",
	}

	var out xboxResponse
	err := c.postJSON(ctx, e.or(e.XSTS, xstsEndpoint), body, "", &out)
	if err != nil {
		return "", "", err
	}
	if out.Token == "" {
		return "", "", fmt.Errorf("no token returned")
	}

	userHash, err := out.userHash()
	if err != nil {
		return "", "", err
	}
	return out.Token, userHash, nil
}

func (c *Config) minecraftToken(ctx context.Context, e Endpoints, userHash, xstsToken string) (string, error) {
	body := map[string]any{
		"identityToken": fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xstsToken),
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.postJSON(ctx, e.or(e.MCLogin, mcLoginEndpoint), body, "", &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("no access token returned")
	}
	return out.AccessToken, nil
}

func (c *Config) profile(ctx context.Context, e Endpoints, mcToken string) (*Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		e.or(e.MCProfile, mcProfileEndpoint), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mcToken)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// A 404 here is the ownership check. The account signed in fine, it simply
	// does not own the game.
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoEntitlement
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("profile lookup returned %s", resp.Status)
	}

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Name == "" {
		return nil, fmt.Errorf("profile lookup returned no username")
	}
	return &Profile{Name: out.Name, UUID: dashUUID(out.ID)}, nil
}

// ErrNoEntitlement means the account is valid but does not own Minecraft.
var ErrNoEntitlement = fmt.Errorf("this account does not own Minecraft Java Edition")

func (c *Config) postJSON(ctx context.Context, endpoint string, body any, bearer string, into any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return decodeXboxError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func decodeXboxError(resp *http.Response) error {
	var out struct {
		XErr    json.Number `json:"XErr"`
		Message string      `json:"Message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.XErr.String() != "" {
		if reason, known := xstsReasons[out.XErr.String()]; known {
			return fmt.Errorf("%s", reason)
		}
		return fmt.Errorf("refused with code %s", out.XErr.String())
	}
	return fmt.Errorf("returned %s", resp.Status)
}

// dashUUID converts Mojang's undashed profile id into the dashed form the game
// expects on the command line.
func dashUUID(id string) string {
	if len(id) != 32 || strings.Contains(id, "-") {
		return id
	}
	return id[0:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:32]
}
