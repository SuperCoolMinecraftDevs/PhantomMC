package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/mojang"
)

// Outcome classifies how a game process ended. The distinction drives what
// happens next, which on an appliance with no desktop is the whole question.
type Outcome int

const (
	// OutcomeQuit means the user left through the in-game menu. There is no
	// desktop to return to, so this is the normal end of a session.
	OutcomeQuit Outcome = iota
	// OutcomeCrash means the process died on its own.
	OutcomeCrash
	// OutcomeSignalled means something outside the process killed it, most
	// often the out of memory killer.
	OutcomeSignalled
)

func (o Outcome) String() string {
	switch o {
	case OutcomeQuit:
		return "quit"
	case OutcomeCrash:
		return "crash"
	case OutcomeSignalled:
		return "signalled"
	default:
		return "unknown"
	}
}

type Session struct {
	Outcome  Outcome
	ExitCode int
	Signal   string
	Runtime  time.Duration
}

// ShortLived reports whether the session died before it plausibly reached the
// main menu. A pack that crashes on startup does so every time, so restarting
// produces a loop rather than a recovery.
func (s Session) ShortLived() bool {
	return s.Outcome != OutcomeQuit && s.Runtime < 60*time.Second
}

type Launcher struct {
	Java   string
	Layout Layout
	Stdout io.Writer
	Stderr io.Writer
}

// Command assembles the java invocation. Mojang's own JVM arguments come first
// so that anything from the manifest can override them.
func (l *Launcher) Command(spec *mojang.LaunchSpec, m *manifest.Manifest, account Account) []string {
	vars := mojang.Vars{
		Classpath:       strings.Join(spec.Classpath, string(filepath.ListSeparator)),
		VersionName:     spec.VersionName,
		VersionType:     spec.VersionType,
		PlayerName:      account.Name,
		UUID:            account.UUID,
		AccessToken:     account.AccessToken,
		UserType:        account.UserType,
		GameDir:         l.Layout.Game,
		AssetsDir:       l.Layout.Assets,
		AssetIndex:      spec.AssetIndex,
		NativesDir:      l.Layout.Libraries,
		LauncherName:    "phantommc",
		LauncherVersion: "0",
	}
	args := make([]string, 0, len(spec.JVMArgs)+len(spec.GameArgs)+len(m.JVM.Args)+2)

	// Substitution runs over both argument groups rather than trusting that the
	// installer already resolved the jvm side. It is idempotent, and a
	// placeholder reaching the jvm is a launch failure with a useless message.
	for _, arg := range spec.JVMArgs {
		args = append(args, mojang.Substitute(arg, vars))
	}
	args = append(args, "-Xmx"+strconv.Itoa(m.JVM.HeapMB)+"M")
	args = append(args, m.JVM.Args...)
	args = append(args, spec.MainClass)

	for _, arg := range spec.GameArgs {
		args = append(args, mojang.Substitute(arg, vars))
	}
	return args
}

type Account struct {
	Name        string
	UUID        string
	AccessToken string
	UserType    string
}

// OfflineAccount derives a stable identity from the username the same way
// offline mode has always worked, so a player keeps the same UUID between boots
// even though nothing is stored.
func OfflineAccount(username string) Account {
	return Account{
		Name:     username,
		UUID:     offlineUUID(username),
		UserType: "legacy",
	}
}

func (l *Launcher) Run(ctx context.Context, args []string) (Session, error) {
	if l.Java == "" {
		return Session{}, fmt.Errorf("no java runtime selected")
	}
	if err := os.MkdirAll(l.Layout.Game, 0o755); err != nil {
		return Session{}, err
	}

	cmd := exec.CommandContext(ctx, l.Java, args...)
	cmd.Dir = l.Layout.Game
	cmd.Stdout = l.Stdout
	cmd.Stderr = l.Stderr

	started := time.Now()
	err := cmd.Run()
	session := Session{Runtime: time.Since(started)}

	if err == nil {
		session.Outcome = OutcomeQuit
		return session, nil
	}

	var exitErr *exec.ExitError
	if !asExitError(err, &exitErr) {
		return session, err
	}

	session.ExitCode = exitErr.ExitCode()
	if signal := signalName(exitErr); signal != "" {
		session.Outcome = OutcomeSignalled
		session.Signal = signal
	} else {
		session.Outcome = OutcomeCrash
	}
	return session, nil
}
