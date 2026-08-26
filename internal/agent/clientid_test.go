package agent

import "testing"

func TestClientIDPrecedence(t *testing.T) {
	t.Setenv(ClientIDEnv, "from-env")

	if id, src := ResolveClientID("from-flag", "from-manifest"); id != "from-flag" || src != ClientIDFromFlag {
		t.Errorf("flag should win, got %q from %s", id, src)
	}
	if id, src := ResolveClientID("", "from-manifest"); id != "from-env" || src != ClientIDFromEnv {
		t.Errorf("env should beat manifest, got %q from %s", id, src)
	}
}

func TestClientIDFallsBackToManifest(t *testing.T) {
	t.Setenv(ClientIDEnv, "")

	if id, src := ResolveClientID("", "from-manifest"); id != "from-manifest" || src != ClientIDFromManifest {
		t.Errorf("manifest should be used, got %q from %s", id, src)
	}
	if id, src := ResolveClientID("", ""); id != "" || src != ClientIDMissing {
		t.Errorf("expected missing, got %q from %s", id, src)
	}
}

func TestClientIDIgnoresWhitespace(t *testing.T) {
	t.Setenv(ClientIDEnv, "")

	if id, _ := ResolveClientID("   ", "  real-id  "); id != "real-id" {
		t.Errorf("values should be trimmed, got %q", id)
	}
}
