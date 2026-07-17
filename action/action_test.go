package action_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/action"
)

// noopCmd is a minimal tea.Cmd for testing.
func noopCmd() tea.Msg { return nil }

// TestDispatchRegistered routes a registered ActionEvent to its handler,
// returns the handler's (cmd, err), and proves the handler was invoked.
func TestDispatchRegistered(t *testing.T) {
	d := action.NewDispatcher()

	called := false
	d.Register("setProvider", func(ev a2ui.ActionEvent) (tea.Cmd, error) {
		called = true
		if ev.Name != "setProvider" {
			t.Fatalf("handler got Name %q, want setProvider", ev.Name)
		}
		return noopCmd, nil
	})

	ev := a2ui.ActionEvent{Name: "setProvider", SurfaceID: "s1", SourceComponentID: "btn"}
	cmd, err := d.Dispatch(ev)
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}
	if !called {
		t.Fatal("handler was not invoked")
	}
	if cmd == nil {
		t.Fatal("Dispatch returned nil cmd, want the handler's cmd")
	}
}

// TestDispatchUnknownRefused proves default-deny: an unregistered name returns
// ErrNoHandler (assertable via errors.Is), invokes no handler, and returns
// nil cmd.
func TestDispatchUnknownRefused(t *testing.T) {
	d := action.NewDispatcher()

	called := false
	d.Register("setProvider", func(ev a2ui.ActionEvent) (tea.Cmd, error) {
		called = true
		return noopCmd, nil
	})

	// Dispatch an unregistered name — a handler IS registered for
	// "setProvider", but not for "runCommand".
	ev := a2ui.ActionEvent{Name: "runCommand"}
	cmd, err := d.Dispatch(ev)

	if cmd != nil {
		t.Fatalf("Dispatch returned non-nil cmd for unregistered name")
	}
	if err == nil {
		t.Fatal("Dispatch returned nil error for unregistered name")
	}
	if !errors.Is(err, action.ErrNoHandler) {
		t.Fatalf("err = %v, want errors.Is ErrNoHandler", err)
	}
	if called {
		t.Fatal("the setProvider handler was invoked for an unregistered name")
	}
}

// TestDispatchEmptyDispatcher proves that a fresh Dispatcher denies
// everything.
func TestDispatchEmptyDispatcher(t *testing.T) {
	d := action.NewDispatcher()
	_, err := d.Dispatch(a2ui.ActionEvent{Name: "anything"})
	if !errors.Is(err, action.ErrNoHandler) {
		t.Fatalf("err = %v, want ErrNoHandler", err)
	}
}

// TestRegisterDuplicate proves duplicate registration is rejected.
func TestRegisterDuplicate(t *testing.T) {
	d := action.NewDispatcher()
	d.Register("setProvider", func(a2ui.ActionEvent) (tea.Cmd, error) { return noopCmd, nil })

	err := d.Register("setProvider", func(a2ui.ActionEvent) (tea.Cmd, error) { return noopCmd, nil })
	if err == nil {
		t.Fatal("duplicate Register returned nil error")
	}
	if !errors.Is(err, action.ErrDuplicateHandler) {
		t.Fatalf("err = %v, want errors.Is ErrDuplicateHandler", err)
	}
}

// TestRegisterEmptyName proves an empty name is rejected.
func TestRegisterEmptyName(t *testing.T) {
	d := action.NewDispatcher()
	err := d.Register("", func(a2ui.ActionEvent) (tea.Cmd, error) { return noopCmd, nil })
	if err == nil {
		t.Fatal("Register with empty name returned nil error")
	}
}

// TestRegisterNilHandler proves a nil handler is rejected.
func TestRegisterNilHandler(t *testing.T) {
	d := action.NewDispatcher()
	err := d.Register("setProvider", nil)
	if err == nil {
		t.Fatal("Register with nil handler returned nil error")
	}
}

// --- Typed accessor tests: each covers hit, missing, and wrong-type ---

func TestStringAccessor(t *testing.T) {
	ev := a2ui.ActionEvent{
		Context: map[string]any{
			"provider": "anthropic",
			"count":    42,
		},
	}

	// Hit.
	s, ok := action.String(ev, "provider")
	if !ok || s != "anthropic" {
		t.Fatalf("String(provider) = (%q, %v), want (anthropic, true)", s, ok)
	}

	// Missing key.
	s, ok = action.String(ev, "missing")
	if ok || s != "" {
		t.Fatalf("String(missing) = (%q, %v), want (\"\", false)", s, ok)
	}

	// Wrong type.
	s, ok = action.String(ev, "count")
	if ok || s != "" {
		t.Fatalf("String(count) = (%q, %v), want (\"\", false) for int value", s, ok)
	}
}

func TestStringsAccessor(t *testing.T) {
	ev := a2ui.ActionEvent{
		Context: map[string]any{
			"tags":    []string{"go", "a2ui"},
			"single":  "not-a-list",
			"missing": nil,
		},
	}

	// Hit.
	ss, ok := action.Strings(ev, "tags")
	if !ok {
		t.Fatalf("Strings(tags) ok = false, want true")
	}
	if len(ss) != 2 || ss[0] != "go" || ss[1] != "a2ui" {
		t.Fatalf("Strings(tags) = %v, want [go a2ui]", ss)
	}

	// Missing key.
	ss, ok = action.Strings(ev, "absent")
	if ok || ss != nil {
		t.Fatalf("Strings(absent) = (%v, %v), want (nil, false)", ss, ok)
	}

	// Wrong type (string, not []string).
	ss, ok = action.Strings(ev, "single")
	if ok || ss != nil {
		t.Fatalf("Strings(single) = (%v, %v), want (nil, false) for string value", ss, ok)
	}
}

func TestBoolAccessor(t *testing.T) {
	ev := a2ui.ActionEvent{
		Context: map[string]any{
			"enabled": true,
			"label":   "yes",
		},
	}

	// Hit.
	b, ok := action.Bool(ev, "enabled")
	if !ok || b != true {
		t.Fatalf("Bool(enabled) = (%v, %v), want (true, true)", b, ok)
	}

	// Missing key.
	b, ok = action.Bool(ev, "missing")
	if ok || b != false {
		t.Fatalf("Bool(missing) = (%v, %v), want (false, false)", b, ok)
	}

	// Wrong type (string, not bool).
	b, ok = action.Bool(ev, "label")
	if ok || b != false {
		t.Fatalf("Bool(label) = (%v, %v), want (false, false) for string value", b, ok)
	}
}

// TestAccessorNilContext proves accessors don't panic on a nil Context map.
func TestAccessorNilContext(t *testing.T) {
	ev := a2ui.ActionEvent{}

	s, ok := action.String(ev, "any")
	if ok || s != "" {
		t.Fatalf("String on nil Context = (%q, %v), want (\"\", false)", s, ok)
	}

	ss, ok := action.Strings(ev, "any")
	if ok || ss != nil {
		t.Fatalf("Strings on nil Context = (%v, %v), want (nil, false)", ss, ok)
	}

	b, ok := action.Bool(ev, "any")
	if ok || b != false {
		t.Fatalf("Bool on nil Context = (%v, %v), want (false, false)", b, ok)
	}
}

// TestDispatcherConcurrentUse verifies Register and Dispatch are safe to
// call from multiple goroutines (run with -race).
func TestDispatcherConcurrentUse(t *testing.T) {
	d := action.NewDispatcher()
	if err := d.Register("seed", func(ev a2ui.ActionEvent) (tea.Cmd, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		name := fmt.Sprintf("h%d", i)
		go func() {
			defer wg.Done()
			_ = d.Register(name, func(ev a2ui.ActionEvent) (tea.Cmd, error) { return nil, nil })
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = d.Dispatch(a2ui.ActionEvent{Name: "seed"})
			}
		}()
	}
	wg.Wait()
}
