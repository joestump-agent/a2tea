// Command hello embeds a short LLM-style reply that contains an A2UI message,
// scans it with a2tea, and runs the first rendered surface in a bubbletea
// program.
//
// It mirrors what a host (crush) does: a2tea.Scan splits the reply into text
// and A2UI messages, and a2tea.Render turns a message's surface into an
// embeddable model. Here we wrap that model in a2tea.Standalone to run it as a
// self-contained program (quits on q / Esc / Ctrl+C).
//
// The surface draws a styled card: an h1 title, body text, a divider, and a
// row of two buttons. Tab / Shift+Tab move button focus and Enter emits
// event.ButtonClicked (not displayed anywhere yet).
package main

import (
	_ "embed"
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea"
)

// sample is an LLM-style reply: conversational text wrapping an <a2ui-json>
// block. Embedding it makes the program run from any directory.
//
//go:embed sample.txt
var sample string

func main() {
	parts, err := a2tea.Scan(sample)
	if err != nil {
		log.Fatalf("a2tea/examples/hello: scan: %v", err)
	}

	for _, p := range parts {
		if len(p.Messages) == 0 {
			continue
		}
		model, err := a2tea.Render(p.Messages)
		if err != nil {
			continue // text-only or non-renderable part
		}
		if _, err := tea.NewProgram(a2tea.Standalone(model)).Run(); err != nil {
			log.Fatalf("a2tea/examples/hello: run: %v", err)
		}
		fmt.Println("a2tea/examples/hello: bye")
		return
	}
	log.Fatal("a2tea/examples/hello: no renderable A2UI surface in sample")
}
