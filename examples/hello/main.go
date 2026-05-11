// Command hello loads a tiny A2UI JSON document from disk, hands it to
// a2tea.Render, and runs the resulting model in a bubbletea program.
//
// This exists to prove the public API wires end-to-end. The rendered output
// is the "[a2tea: card]" placeholder until the renderers are implemented.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea"
)

func main() {
	path := samplePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("a2tea/examples/hello: read %s: %v", path, err)
	}

	model, err := a2tea.Render(json.RawMessage(raw))
	if err != nil {
		log.Fatalf("a2tea/examples/hello: render: %v", err)
	}

	if _, err := tea.NewProgram(model).Run(); err != nil {
		log.Fatalf("a2tea/examples/hello: run: %v", err)
	}
	fmt.Println("a2tea/examples/hello: bye")
}

// samplePath resolves examples/hello/sample.json relative to this source
// file so `go run ./examples/hello` works no matter the caller's cwd.
func samplePath() string {
	// Prefer a sibling sample.json when running via `go run` from anywhere.
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "sample.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Fall back to cwd-relative for `go run ./examples/hello`.
	return filepath.Join("examples", "hello", "sample.json")
}
