// Command hello embeds a tiny A2UI JSON document, hands it to a2tea.Render,
// and runs the resulting model in a bubbletea program.
//
// This exists to prove the public API wires end-to-end. The rendered output
// is the "[a2tea: card]" placeholder until the renderers are implemented.
//
// The renderer returned by a2tea.Render is an embeddable child component that
// deliberately does not handle quit, so the example wraps it in
// a2tea.Standalone, which quits on q / Esc / Ctrl+C.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea"
)

// sampleJSON is the A2UI document rendered by this example. Embedding it makes
// the program work no matter the caller's cwd — unlike a cwd- or
// os.Executable-relative file lookup, which fails under `go run` (the binary
// lives in the build cache) and from any directory other than the repo root.
//
//go:embed sample.json
var sampleJSON []byte

func main() {
	model, err := a2tea.Render(json.RawMessage(sampleJSON))
	if err != nil {
		log.Fatalf("a2tea/examples/hello: render: %v", err)
	}

	if _, err := tea.NewProgram(a2tea.Standalone(model)).Run(); err != nil {
		log.Fatalf("a2tea/examples/hello: run: %v", err)
	}
	fmt.Println("a2tea/examples/hello: bye")
}
