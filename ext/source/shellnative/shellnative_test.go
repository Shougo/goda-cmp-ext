package shellnative

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shougo/goda-cmp.vim/cmp"
	"github.com/Shougo/goda-cmp.vim/cmp/cmptest"
)

func TestCmdPrefix(t *testing.T) {
	for in, want := range map[string]string{
		"!ls -":            "!",
		"silent !git st":   "silent !",
		"%!sort":           "%!",
		"terminal ++close": "terminal ++close",
		"echo 1":           "",
		"'<,'>!sort":       "'<,'>!",
	} {
		if got := cmdPrefix.FindString(in); got != want {
			t.Errorf("cmdPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnsupportedShell(t *testing.T) {
	if err := New(Params{Shell: "csh"}).(source).Init(context.Background(), nil); err == nil {
		t.Error("csh: no error")
	}
}

// With zsh installed: a real completion of a command name.
func TestZsh(t *testing.T) {
	if testing.Short() {
		t.Skip("starts zsh")
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	src := New(Params{Shell: "zsh"}).(source)
	h := &cmptest.Host{Funcs: map[string]func(...any) (any, error){
		"getcwd":  func(...any) (any, error) { return t.TempDir(), nil },
		"environ": func(...any) (any, error) { return map[string]any{"PATH": os.Getenv("PATH"), "HOME": t.TempDir()}, nil },
	}}
	ctx := context.Background()
	if err := src.Init(ctx, h); err != nil {
		t.Fatal(err)
	}
	defer src.Close(ctx)
	g, err := src.Gather(ctx, h, &cmp.Context{Mode: "t", Input: "ech"})
	if err != nil {
		t.Fatal(err)
	}
	var ws []string
	for _, it := range g.Items {
		ws = append(ws, it.Word)
	}
	if g.Start != 0 || !reflect.DeepEqual(ws[:min(1, len(ws))], []string{"echo"}) {
		t.Errorf("start %d words %q", g.Start, ws)
	}
}

// A request that gives up does not get its answer read by the next one.
func TestStaleAnswer(t *testing.T) {
	if testing.Short() {
		t.Skip("starts zsh")
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	src := New(Params{Shell: "zsh"}).(source)
	h := &cmptest.Host{Funcs: map[string]func(...any) (any, error){
		"getcwd":  func(...any) (any, error) { return t.TempDir(), nil },
		"environ": func(...any) (any, error) { return map[string]any{"PATH": os.Getenv("PATH"), "HOME": t.TempDir()}, nil },
	}}
	ctx := context.Background()
	if err := src.Init(ctx, h); err != nil {
		t.Fatal(err)
	}
	defer src.Close(ctx)
	gone, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := src.complete(gone, "ech", ""); err == nil {
		t.Fatal("a cancelled request answered")
	}
	got, err := src.complete(ctx, "printf", "")
	if err != nil {
		t.Fatal(err)
	}
	// The lines of the answer (empty ones are dropped by Gather).
	for _, l := range got {
		if l != "" && !strings.HasPrefix(l, "printf") {
			t.Errorf("the answer for printf has %q: %q", l, got)
		}
	}
	if !slices.Contains(got, "printf") {
		t.Errorf("the answer for printf: %q", got)
	}
}

// zshHost is the host of a zsh with FPATH (the default fpath and dir first).
func zshHost(t *testing.T, dir string) *cmptest.Host {
	t.Helper()
	out, err := exec.Command("zsh", "-fc", "print -r -- ${(j.:.)fpath}").Output()
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]any{"PATH": os.Getenv("PATH"), "HOME": t.TempDir(),
		"FPATH":                             dir + ":" + strings.TrimSpace(string(out)),
		"GODA_CMP_SHELLNATIVE_READ_TIMEOUT": "1"}
	return &cmptest.Host{Funcs: map[string]func(...any) (any, error){
		"getcwd":  func(...any) (any, error) { return dir, nil },
		"environ": func(...any) (any, error) { return env, nil },
	}}
}

// A completion that writes without end: the capture script gives up (its
// memory does not grow without end, as zpty -r -m did), and the shell is
// started again for the next completion.
func TestEndlessOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("starts zsh")
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "_goda_endless"),
		[]byte("#compdef goda-endless\nwhile :; do print -r -- ${(l:200::x:)}; done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := New(Params{Shell: "zsh"}).(source)
	h := zshHost(t, dir)
	ctx := context.Background()
	if err := src.Init(ctx, h); err != nil {
		t.Fatal(err)
	}
	defer src.Close(ctx)
	start := time.Now()
	if _, err := src.Gather(ctx, h, &cmp.Context{Mode: "t", Input: "goda-endless "}); err == nil {
		t.Error("endless output: no error")
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("endless output took %v", d)
	}
	// Started again.
	time.Sleep(restartInterval)
	g, err := src.Gather(ctx, h, &cmp.Context{Mode: "t", Input: "ech"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(g.Items, func(it cmp.Item) bool { return it.Word == "echo" }) {
		t.Errorf("after the restart: %v", g.Items)
	}
	// Not sooner than restartInterval after the last start.
	_ = src.st.cmd.Process.Kill()
	<-src.st.dead
	if _, err := src.Gather(ctx, h, &cmp.Context{Mode: "t", Input: "ech"}); err == nil {
		t.Error("started again at once")
	}
}

// Closed: not started again.
func TestNoRestartAfterClose(t *testing.T) {
	src := New(Params{Shell: "zsh"}).(source)
	_ = src.Close(context.Background())
	if err := src.ensure(context.Background(), nil); err != nil {
		t.Errorf("ensure after Close: %v", err)
	}
	if src.st.cmd != nil {
		t.Error("started after Close")
	}
}
