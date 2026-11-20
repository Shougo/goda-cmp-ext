package buffer

import (
	"context"
	"slices"
	"testing"

	"github.com/Shougo/goda-cmp.vim/cmp"
	"github.com/Shougo/goda-cmp.vim/cmp/cmptest"
)

// host is an editor with buffer 1, in the tab page, holding lines.
func host(lines ...string) *cmptest.Host {
	return &cmptest.Host{Funcs: map[string]func(...any) (any, error){
		"getbufline": func(...any) (any, error) {
			l := make([]any, len(lines))
			for i, s := range lines {
				l[i] = s
			}
			return l, nil
		},
		"bufname": func(...any) (any, error) { return "shell", nil },
		"getbufvar": func(a ...any) (any, error) {
			if a[1] == "changedtick" {
				return int64(1), nil
			}
			return "goda-term-shell", nil
		},
		"tabpagebuflist": func(...any) (any, error) { return []any{int64(1)}, nil },
		"bufnr":          func(...any) (any, error) { return int64(-1), nil },
	}}
}

func words(t *testing.T, s source, h *cmptest.Host, pattern string) []string {
	t.Helper()
	c := &cmp.Context{Bufnr: 1, Lnum: 3, Input: "% vim", Filetype: "goda-term-shell"}
	if err := c.SetKeywordPattern(pattern); err != nil {
		t.Fatal(err)
	}
	g, err := s.Gather(context.Background(), h, c)
	if err != nil {
		t.Fatal(err)
	}
	var ws []string
	for _, it := range g.Items {
		ws = append(ws, it.Word)
	}
	return ws
}

// The words are those of the keyword pattern of the buffer completed, each
// pattern its own.
func TestKeywordPattern(t *testing.T) {
	s := New(Params{}).(source)
	h := host("|  M vim/rc/goda-cmp/config.go", "% git add vim/rc", "% vim")
	if err := s.read(context.Background(), h, 1, "goda-term-shell", true); err != nil {
		t.Fatal(err)
	}
	path := "vim/rc/goda-cmp/config.go"
	for _, tc := range []struct {
		pattern string
		has     string
		hasNot  string
	}{
		{"", "config", path},
		{`[0-9a-zA-Z_./#:-]*`, path, "config"},
		{"", "config", path}, // the words of the first pattern are kept
	} {
		ws := words(t, s, h, tc.pattern)
		if !slices.Contains(ws, tc.has) || slices.Contains(ws, tc.hasNot) {
			t.Errorf("pattern %q: %q", tc.pattern, ws)
		}
		if n := len(ws); n != len(slices.Compact(slices.Sorted(slices.Values(ws)))) {
			t.Errorf("pattern %q: duplicates in %q", tc.pattern, ws)
		}
	}
}
