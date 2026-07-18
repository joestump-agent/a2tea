package render

import (
	"strings"

	a2ui "github.com/tmc/a2ui"
)

// Apply applies a sequence of A2UI server messages to the surface, compositing
// state in order: updateComponents merges components by ID (siblings survive),
// updateDataModel sets bound values, and deleteSurface clears the surface.
//
// Messages are scoped to this surface: an updateComponents or updateDataModel
// carrying a different non-empty SurfaceID is skipped, so a stream that
// interleaves several surfaces cannot corrupt this one. An empty SurfaceID on
// those messages is treated as targeting this surface — some producers omit
// the field when only one surface is in play. deleteSurface is stricter: being
// destructive, it fires only on an exact SurfaceID match, never on an empty
// one.
//
// createSurface is deliberately a no-op. a2tea treats the first
// updateComponents as implicit surface creation, and the hints createSurface
// carries have no honorable mapping here: theme (primaryColor, iconUrl,
// agentDisplayName) is superseded by host theming via WithStyles — component
// chrome stays monochrome so the host theme wins — and catalogId is ignored
// because a2tea's catalog is the compiled-in one, by design.
//
// deleteSurface clears all surface state (components, data model, edits,
// focus, active tabs) but processing continues: a later updateComponents in
// the same batch legally re-creates the surface.
//
// Apply returns false when no renderable state remains after all messages
// (the surface was deleted and not re-created); the caller should treat the
// surface as gone.
func (s *Surface) Apply(msgs []a2ui.ServerMessage) bool {
	for _, m := range msgs {
		switch {
		case m.CreateSurface != nil:
			// Deliberate no-op: surface creation is implied by the first
			// updateComponents, host WithStyles owns theming, and the
			// component catalog is compiled in. See the Apply doc comment
			// and docs/wire-format.md.
		case m.UpdateComponents != nil:
			if !s.targetsThisSurface(m.UpdateComponents.SurfaceID) {
				continue
			}
			s.applyComponents(m.UpdateComponents.Components)
		case m.UpdateDataModel != nil:
			if !s.targetsThisSurface(m.UpdateDataModel.SurfaceID) {
				continue
			}
			s.applyDataModel(m.UpdateDataModel.Path, m.UpdateDataModel.Value)
		case m.DeleteSurface != nil:
			if m.DeleteSurface.SurfaceID == s.id {
				s.byID = make(map[string]a2ui.Component)
				s.rootID = ""
				s.focusables = nil
				s.focusIdx = 0
				s.data = nil
				s.fieldValues = nil
				s.checkValues = nil
				s.choiceValues = nil
				s.sliderValues = nil
				s.choiceCursor = nil
				s.openModals = nil
				s.activeTabs = nil
			}
		}
	}
	return s.rootID != ""
}

// targetsThisSurface reports whether a message with the given SurfaceID
// applies to this surface. An empty SurfaceID is lenient — treated as "the
// current surface" — because some producers omit it when only one surface
// exists; any other mismatch is a different surface's message.
func (s *Surface) targetsThisSurface(surfaceID string) bool {
	return surfaceID == "" || surfaceID == s.id
}

// applyComponents merges the given components into the surface's component map
// by ID: an update to component X replaces X, leaving siblings intact. It
// re-derives the root and focus ring, preserving focus on the same component
// if it survives the merge. Modal open state also survives the merge — only
// entries whose component stopped being a modal are pruned — so an update
// does not slam an open modal shut under the user. Active-tab state
// (s.activeTabs) is likewise deliberately left untouched — like focus, it is
// user state the server must not clobber — and out-of-range indices left
// behind by a shrunken tab list are clamped at read time by activeTab.
func (s *Surface) applyComponents(components []a2ui.Component) {
	// Remember which component held focus before the merge.
	var focusedID string
	if len(s.focusables) > 0 && s.focusIdx < len(s.focusables) {
		focusedID = s.focusables[s.focusIdx]
	}

	for _, c := range components {
		s.byID[c.ID] = c
	}
	s.rootID = s.deriveRootID()
	s.pruneOpenModals()
	s.focusables = s.collectFocusables()

	// Preserve focus if the focused component survives.
	s.focusIdx = 0
	if focusedID != "" {
		for i, id := range s.focusables {
			if id == focusedID {
				s.focusIdx = i
				break
			}
		}
	}
}

// applyDataModel sets a value at the given JSON Pointer path in the surface's
// data model. Bound DynamicString/Value components resolve from this on the
// next render.
func (s *Surface) applyDataModel(path string, value any) {
	if s.data == nil {
		s.data = make(map[string]any)
	}
	if path == "" || path == "/" {
		s.data[""] = value
		return
	}
	s.data[strings.TrimPrefix(path, "/")] = value
}

// deriveRootID returns the ID of the component that is not referenced as any
// other component's child. It prefers the current rootID if it is still
// unreferenced, so adding new standalone components does not change the root.
// Otherwise it chooses deterministically — the lexicographically smallest
// unreferenced ID (or the smallest ID overall if every component is
// referenced, i.e. a cycle) — because map iteration order is randomized and
// the rendered root must be stable across renders.
func (s *Surface) deriveRootID() string {
	if len(s.byID) == 0 {
		return ""
	}
	referenced := make(map[string]bool)
	for _, c := range s.byID {
		for _, id := range childIDs(c) {
			referenced[id] = true
		}
	}
	// Prefer the existing root if it survived and is still unreferenced.
	if s.rootID != "" {
		if _, exists := s.byID[s.rootID]; exists && !referenced[s.rootID] {
			return s.rootID
		}
	}
	var firstID, firstUnref string
	for id := range s.byID {
		if firstID == "" || id < firstID {
			firstID = id
		}
		if !referenced[id] && (firstUnref == "" || id < firstUnref) {
			firstUnref = id
		}
	}
	if firstUnref != "" {
		return firstUnref
	}
	return firstID
}
