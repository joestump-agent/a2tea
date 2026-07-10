package render

// This file pins the rendering dependencies that the real renderers will
// use so they appear as direct (not indirect) dependencies in go.mod from
// day one. The blank imports keep the toolchain honest: tooling that
// upgrades dependencies can see what this package is supposed to depend on
// even before the concrete renderers are wired up.
//
// Cost to remember: because these are blank imports rather than real uses,
// every downstream consumer of the render package compiles glamour,
// bubbles/list, etc. today even if it never renders those kinds. That is
// harmless for crush (it already depends on all of them) but is exactly why
// each import below should be dropped the moment its renderer starts using the
// package directly — leaving them in place forces dead compilation on everyone
// else.
//
// TODO(a2tea): drop these blank imports as the real renderers start using
// each package directly.

import (
	_ "charm.land/bubbles/v2/list"
	_ "charm.land/bubbles/v2/progress"
	_ "charm.land/bubbles/v2/textinput"
	_ "charm.land/glamour/v2"
	_ "charm.land/lipgloss/v2"
	// TODO(a2tea): bring in huh for FormModel. Skipped at scaffold time
	// because the only published version compatible with this dependency
	// set (charmbracelet/huh v1.0.0) pins old charm.land/x/ansi internals
	// that conflict with the v2 ecosystem already locked in here. Revisit
	// once an updated huh release on the v2 stack is available.
)
