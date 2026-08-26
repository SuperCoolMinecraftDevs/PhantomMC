package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxManifestBytes = 4 << 20

// Load reads a manifest from a local path or an https url and validates it.
func Load(ctx context.Context, client *http.Client, source string) (*Manifest, error) {
	var (
		reader io.ReadCloser
		err    error
	)

	switch {
	case strings.HasPrefix(source, "https://"):
		reader, err = fetch(ctx, client, source)
	case strings.HasPrefix(source, "http://"):
		return nil, fmt.Errorf("refusing to load a manifest over plaintext http: %s", source)
	default:
		reader, err = os.Open(source)
	}
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return Decode(reader, time.Now())
}

func Decode(r io.Reader, now time.Time) (*Manifest, error) {
	var m Manifest

	decoder := json.NewDecoder(io.LimitReader(r, maxManifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if err := m.Validate(now); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	return &m, nil
}

func fetch(ctx context.Context, client *http.Client, url string) (io.ReadCloser, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch manifest: %s returned %s", url, resp.Status)
	}
	return resp.Body, nil
}
