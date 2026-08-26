package agent

import (
	"os"
	"strings"
)

// ClientIDEnv is checked when no client id was passed on the command line.
const ClientIDEnv = "PHANTOM_CLIENT_ID"

// ClientIDSource says where a value came from, so the agent can tell the user
// which of three places it picked up rather than leaving them guessing.
type ClientIDSource string

const (
	ClientIDFromFlag     ClientIDSource = "command line"
	ClientIDFromEnv      ClientIDSource = "environment"
	ClientIDFromManifest ClientIDSource = "manifest"
	ClientIDMissing      ClientIDSource = "unset"
)

// ResolveClientID picks the Azure application to sign in against. The command
// line wins so a single boot can be overridden without touching the manifest,
// then the environment, then the manifest itself, which is where a deployment
// normally carries it.
func ResolveClientID(flagValue, manifestValue string) (string, ClientIDSource) {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v, ClientIDFromFlag
	}
	if v := strings.TrimSpace(os.Getenv(ClientIDEnv)); v != "" {
		return v, ClientIDFromEnv
	}
	if v := strings.TrimSpace(manifestValue); v != "" {
		return v, ClientIDFromManifest
	}
	return "", ClientIDMissing
}

// MissingClientIDHelp is printed instead of a bare error, because "no client id"
// is useless to someone who has never registered an Azure application.
const MissingClientIDHelp = `No Microsoft client id is configured, so sign in cannot start.

Provide one in any of these ways, highest priority first:

  phantomd -client-id <application-id> ...
  PHANTOM_CLIENT_ID=<application-id> phantomd ...
  "auth": { "mode": "microsoft", "clientId": "<application-id>" }   in the manifest

An application id comes from registering an app in Microsoft Entra, and the
application additionally needs the XboxLive.signin permission before the
Minecraft API will accept it. See docs/authentication.md.`
