package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/mojang"
)

func testLauncher(t *testing.T, script string) *Launcher {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fakejava")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Launcher{Java: path, Layout: NewLayout(dir)}
}

func TestCleanExitIsAQuit(t *testing.T) {
	session, err := testLauncher(t, "exit 0").Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if session.Outcome != OutcomeQuit {
		t.Errorf("expected quit, got %s", session.Outcome)
	}
	if session.ShortLived() {
		t.Error("a clean quit is never short lived, however fast it was")
	}
}

func TestNonZeroExitIsACrash(t *testing.T) {
	session, err := testLauncher(t, "exit 1").Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if session.Outcome != OutcomeCrash {
		t.Errorf("expected crash, got %s", session.Outcome)
	}
	if session.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", session.ExitCode)
	}
	if !session.ShortLived() {
		t.Error("an immediate crash should count as short lived")
	}
}

func TestSignalIsDistinguishedFromACrash(t *testing.T) {
	session, err := testLauncher(t, "kill -9 $$").Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if session.Outcome != OutcomeSignalled {
		t.Errorf("expected signalled, got %s", session.Outcome)
	}
	if session.Signal == "" {
		t.Error("signal name should be recorded so the out of memory case can be named")
	}
}

func TestMissingJavaIsAnError(t *testing.T) {
	l := &Launcher{Layout: NewLayout(t.TempDir())}
	if _, err := l.Run(context.Background(), nil); err == nil {
		t.Fatal("expected an error when no runtime was selected")
	}
}

func TestShortLivedThreshold(t *testing.T) {
	cases := []struct {
		outcome Outcome
		runtime time.Duration
		want    bool
	}{
		{OutcomeCrash, 5 * time.Second, true},
		{OutcomeCrash, 2 * time.Hour, false},
		{OutcomeSignalled, time.Second, true},
		{OutcomeQuit, time.Second, false},
	}
	for _, tc := range cases {
		s := Session{Outcome: tc.outcome, Runtime: tc.runtime}
		if s.ShortLived() != tc.want {
			t.Errorf("%s after %s: expected %v", tc.outcome, tc.runtime, tc.want)
		}
	}
}

func TestOfflineUUIDIsStableAndWellFormed(t *testing.T) {
	a := OfflineAccount("Steve")
	b := OfflineAccount("Steve")

	if a.UUID != b.UUID {
		t.Error("the same username must always produce the same uuid")
	}
	if OfflineAccount("Alex").UUID == a.UUID {
		t.Error("different usernames must produce different uuids")
	}
	if len(a.UUID) != 36 || strings.Count(a.UUID, "-") != 4 {
		t.Errorf("malformed uuid: %s", a.UUID)
	}
	if a.UUID[14] != '3' {
		t.Errorf("expected a version 3 uuid, got %s", a.UUID)
	}
	if a.Name != "Steve" || a.UserType != "legacy" {
		t.Errorf("unexpected account: %#v", a)
	}
}

func TestCommandOrdersArgumentsAndSubstitutes(t *testing.T) {
	l := &Launcher{Java: "/usr/bin/java", Layout: NewLayout("/var/lib/phantom")}
	spec := &mojang.LaunchSpec{
		MainClass:   "net.minecraft.client.main.Main",
		Classpath:   []string{"/lib/a.jar", "/lib/b.jar"},
		JVMArgs:     []string{"-Djava.library.path=${natives_directory}", "-cp", "${classpath}"},
		GameArgs:    []string{"--username", "${auth_player_name}", "--uuid", "${auth_uuid}", "--userType", "${user_type}"},
		AssetIndex:  "32",
		VersionName: "1.21.8",
	}
	m := &manifest.Manifest{JVM: manifest.JVM{HeapMB: 4096, Args: []string{"-XX:+UseG1GC"}}}

	args := l.Command(spec, m, OfflineAccount("Steve"))
	joined := strings.Join(args, " ")

	if strings.Contains(joined, "${") {
		t.Errorf("unsubstituted placeholder: %s", joined)
	}
	if !strings.Contains(joined, "-Xmx4096M") {
		t.Errorf("heap not applied: %s", joined)
	}
	if !strings.Contains(joined, "Steve") {
		t.Errorf("username not substituted: %s", joined)
	}
	if !strings.Contains(joined, "legacy") {
		t.Errorf("user type not substituted: %s", joined)
	}
	if !strings.Contains(joined, "/lib/a.jar") {
		t.Errorf("classpath not substituted: %s", joined)
	}
	uuid := OfflineAccount("Steve").UUID
	if !strings.Contains(joined, uuid) {
		t.Errorf("uuid not substituted: %s", joined)
	}

	mainAt := indexOf(args, "net.minecraft.client.main.Main")
	heapAt := indexOf(args, "-Xmx4096M")
	userAt := indexOf(args, "--username")

	if mainAt < 0 || heapAt < 0 || userAt < 0 {
		t.Fatalf("expected arguments missing: %v", args)
	}
	if !(heapAt < mainAt && mainAt < userAt) {
		t.Errorf("jvm args must precede the main class, which must precede game args: %v", args)
	}
	if indexOf(args, "-XX:+UseG1GC") > mainAt {
		t.Error("manifest jvm args must land before the main class")
	}
}

func indexOf(haystack []string, needle string) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}
