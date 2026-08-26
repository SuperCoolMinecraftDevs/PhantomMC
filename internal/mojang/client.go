package mojang

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const VersionManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

type Client struct {
	HTTP *http.Client
	URL  string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTP: httpClient, URL: VersionManifestURL}
}

func (c *Client) Manifest(ctx context.Context) (*VersionManifest, error) {
	var m VersionManifest
	if err := c.getJSON(ctx, c.URL, &m); err != nil {
		return nil, fmt.Errorf("fetch version manifest: %w", err)
	}
	return &m, nil
}

func (c *Client) Version(ctx context.Context, summary *VersionSummary) (*Version, error) {
	var v Version
	if err := c.getJSON(ctx, summary.URL, &v); err != nil {
		return nil, fmt.Errorf("fetch version %s: %w", summary.ID, err)
	}
	return &v, nil
}

func (c *Client) Resolve(ctx context.Context, id string) (*Version, error) {
	manifest, err := c.Manifest(ctx)
	if err != nil {
		return nil, err
	}
	summary, err := manifest.Find(id)
	if err != nil {
		return nil, err
	}
	return c.Version(ctx, summary)
}

func (c *Client) getJSON(ctx context.Context, url string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

// Fetch downloads an artifact to dest and verifies it against the SHA1 Mojang
// publishes. A file already present with the right digest is left alone, which
// makes the prefetch tool and a retried boot cheap.
func (c *Client) Fetch(ctx context.Context, artifact Download, dest string) error {
	if ok, _ := verifySHA1(dest, artifact.SHA1); ok {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", artifact.URL, resp.Status)
	}

	// Write to a temporary name so an interrupted download never leaves a file
	// that looks complete on the next boot.
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".partial-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	digest := sha1.New()
	if _, err := io.Copy(io.MultiWriter(tmp, digest), resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if got := hex.EncodeToString(digest.Sum(nil)); artifact.SHA1 != "" && got != artifact.SHA1 {
		return fmt.Errorf("%s: digest mismatch, expected %s got %s", dest, artifact.SHA1, got)
	}
	return os.Rename(tmp.Name(), dest)
}

func verifySHA1(path, want string) (bool, error) {
	if want == "" {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	digest := sha1.New()
	if _, err := io.Copy(digest, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(digest.Sum(nil)) == want, nil
}
