package action_test

import (
	"errors"
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/action"
)

// ExampleDispatch shows the intended flow: the host receives an
// a2ui.ActionEvent (from the renderer's ClientMessage), routes it through a
// Dispatcher with a closed vocabulary, and reads typed values from Context.
//
// An unregistered name is denied by construction — the handler is never
// invoked and ErrNoHandler is returned.
func Example_dispatch() {
	d := action.NewDispatcher()

	// Register the host's closed vocabulary.
	d.Register("setProvider", func(ev a2ui.ActionEvent) (tea.Cmd, error) {
		provider, ok := action.String(ev, "provider")
		if !ok {
			return nil, fmt.Errorf("setProvider: missing provider in context")
		}
		fmt.Printf("switching to %s\n", provider)
		return nil, nil
	})

	// Simulate the ActionEvent the renderer would emit on a Button activation.
	ev := a2ui.ActionEvent{
		Name:              "setProvider",
		SurfaceID:         "main",
		SourceComponentID: "providerBtn",
		Context: map[string]any{
			"provider": "anthropic",
		},
	}

	// Dispatch the event.
	cmd, err := d.Dispatch(ev)
	if err != nil {
		log.Fatalf("dispatch: %v", err)
	}
	_ = cmd // cmd would flow through the host's Update loop

	// Now try an unregistered name — denied by default.
	_, err = d.Dispatch(a2ui.ActionEvent{Name: "deleteEverything"})
	if errors.Is(err, action.ErrNoHandler) {
		fmt.Println("refused: no handler")
	}

	// Output:
	// switching to anthropic
	// refused: no handler
}
