// Package action provides a default-deny dispatcher for the inbound half of
// the A2UI protocol: when the host receives an [a2ui.ActionEvent] (emitted by
// a2tea's renderer when a Button with an EventAction is activated), the
// Dispatcher routes it to a registered handler by Name — and refuses
// everything else.
//
// The security property is that an unregistered Name provably invokes no
// handler: [Dispatcher.Dispatch] returns [ErrNoHandler] (assertable via
// [errors.Is]) and executes nothing. Hosts register exactly their closed
// vocabulary (e.g. setProvider / toggleFeature / runCommand) and every other
// name is denied by construction, making "no arbitrary RPC" the path of least
// resistance rather than host discipline.
//
// The typed Context accessors ([String], [Strings], [Bool]) handle the
// map[string]any coercion with the a2ui type shapes baked in once, so hosts
// don't re-derive them. They return (zero, false) on a missing key or type
// mismatch — never panicking.
//
// The intended flow pairs with the renderer's emitted message. Activating a
// Button with a server-side event emits an [a2ui.ClientMessage] as its own
// tea.Msg (alongside the host-facing event.ButtonClicked); the host matches
// it in Update and dispatches its Action:
//
//	case a2ui.ClientMessage: // from Surface.Update's batch
//		if msg.Action == nil { return } // FunctionCall-only button
//		cmd, err := d.Dispatch(*msg.Action)
//
// [Handler] bodies are entirely host-owned. The library performs no host
// operations; it provides the routing shell and typed accessors only.
package action

import (
	"errors"
	"fmt"
	"sync"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"
)

// ErrNoHandler is returned by [Dispatcher.Dispatch] when no handler is
// registered for the ActionEvent's Name. It wraps the name so the caller can
// log it without inspecting the event separately. Assert via errors.Is.
var ErrNoHandler = errors.New("action: no handler registered")

// ErrDuplicateHandler is returned by [Dispatcher.Register] when a handler is
// already registered for the given name. This catches accidental
// double-registration early rather than silently shadowing.
var ErrDuplicateHandler = errors.New("action: handler already registered")

// Handler processes one ActionEvent. The host closes over its own state
// (config paths, command runners, etc.); the library does not thread host
// state. The returned tea.Cmd flows through the standard bubbletea Update
// loop.
type Handler func(ev a2ui.ActionEvent) (tea.Cmd, error)

// Dispatcher routes inbound ActionEvents to registered handlers by Name,
// refusing any name outside the registered vocabulary.
//
// Dispatcher is safe for concurrent use: Register and Dispatch may be called
// from different goroutines (e.g. Dispatch from inside a tea.Cmd). Handlers
// themselves run outside the Dispatcher's lock, so a handler may re-enter
// the Dispatcher without deadlocking; handler bodies are still host-owned
// and must handle their own synchronization.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewDispatcher returns an empty Dispatcher with no registered handlers.
// Every name is denied until explicitly registered.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[string]Handler)}
}

// Register adds a handler for the given action name. It returns
// [ErrDuplicateHandler] if a handler is already registered for that name.
// Names are case-sensitive (A2UI action names are a closed vocabulary, not
// display text).
func (d *Dispatcher) Register(name string, h Handler) error {
	if name == "" {
		return fmt.Errorf("action: empty handler name")
	}
	if h == nil {
		return fmt.Errorf("action: nil handler for %q", name)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateHandler, name)
	}
	d.handlers[name] = h
	return nil
}

// Dispatch routes the ActionEvent to its registered handler. If no handler is
// registered for ev.Name, it returns an [ErrNoHandler] error wrapping the name
// and invokes nothing — this is the closed-vocabulary guarantee.
func (d *Dispatcher) Dispatch(ev a2ui.ActionEvent) (tea.Cmd, error) {
	d.mu.RLock()
	h, ok := d.handlers[ev.Name]
	d.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNoHandler, ev.Name)
	}
	// The handler runs outside the lock so it may re-enter the Dispatcher.
	return h(ev)
}

// String extracts a string value from the ActionEvent's Context by key. It
// returns ("", false) if the key is missing or the value is not a string.
func String(ev a2ui.ActionEvent, key string) (string, bool) {
	v, ok := ev.Context[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// Strings extracts a []string value from the ActionEvent's Context by key.
// This is the shape ChoicePicker values take (A2UI's DynamicStringList is
// natively multi-value). It returns (nil, false) if the key is missing or the
// value is not a []string.
func Strings(ev a2ui.ActionEvent, key string) ([]string, bool) {
	v, ok := ev.Context[key]
	if !ok {
		return nil, false
	}
	s, ok := v.([]string)
	return s, ok
}

// Bool extracts a bool value from the ActionEvent's Context by key. This is
// the shape CheckBox values take. It returns (false, false) if the key is
// missing or the value is not a bool.
func Bool(ev a2ui.ActionEvent, key string) (bool, bool) {
	v, ok := ev.Context[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}
