package mojang

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func digestOf(b []byte) string {
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}

func TestFetchVerifiesDigest(t *testing.T) {
	payload := []byte("a jar, allegedly")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	dest := filepath.Join(t.TempDir(), "nested", "artifact.jar")

	err := client.Fetch(context.Background(), Download{
		URL:  server.URL,
		SHA1: digestOf(payload),
	}, dest)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("content mismatch: %q", got)
	}
}

func TestFetchRejectsWrongDigest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("something else entirely"))
	}))
	defer server.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "artifact.jar")

	err := NewClient(server.Client()).Fetch(context.Background(), Download{
		URL:  server.URL,
		SHA1: digestOf([]byte("what we asked for")),
	}, dest)

	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected a digest mismatch, got %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("a failed download must not leave the destination in place")
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("partial file left behind: %v", entries)
	}
}

func TestFetchSkipsWhenAlreadyCorrect(t *testing.T) {
	payload := []byte("already here")
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(payload)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "artifact.jar")
	if err := os.WriteFile(dest, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	err := NewClient(server.Client()).Fetch(context.Background(), Download{
		URL:  server.URL,
		SHA1: digestOf(payload),
	}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 0 {
		t.Errorf("expected no request for a file already present, got %d", hits)
	}
}

func TestFetchRedownloadsCorruptFile(t *testing.T) {
	payload := []byte("the real thing")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "artifact.jar")
	if err := os.WriteFile(dest, []byte("truncated"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := NewClient(server.Client()).Fetch(context.Background(), Download{
		URL:  server.URL,
		SHA1: digestOf(payload),
	}, dest)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(dest)
	if string(got) != string(payload) {
		t.Errorf("corrupt file was not replaced: %q", got)
	}
}

func TestResolveWalksManifestToVersion(t *testing.T) {
	mux := http.NewServeMux()
	var base string

	mux.HandleFunc("/versions/26.2.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"26.2","type":"release","mainClass":"net.minecraft.client.main.Main"}`))
	})
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"latest":{"release":"26.2"},"versions":[
			{"id":"26.2","type":"release","url":"` + base + `/versions/26.2.json"}]}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	base = server.URL

	client := NewClient(server.Client())
	client.URL = server.URL + "/manifest.json"

	v, err := client.Resolve(context.Background(), "26.2")
	if err != nil {
		t.Fatal(err)
	}
	if v.MainClass != "net.minecraft.client.main.Main" {
		t.Errorf("unexpected main class %q", v.MainClass)
	}

	if _, err := client.Resolve(context.Background(), "9.9.9"); err == nil {
		t.Error("expected an error for an unknown version")
	}
}

func TestErrorsOnBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.URL = server.URL

	if _, err := client.Manifest(context.Background()); err == nil {
		t.Error("expected an error on a 500")
	}
}
