package render

import (
	"strings"

	a2ui "github.com/tmc/a2ui"
)

// Apply applies a sequence of A2UI server messages to the surface, compositing
// state in order: updateComponents merges components by ID (siblings survive),
// updateDataModel sets bound values, and deleteSurface clears the surface.
//
// Apply returns false if the surface was deleted (no renderable state
// remains); the caller should treat the surface as gone.
func (s *Surface) Apply(msgs []a2ui.ServerMessage) bool {
	for _, m := range msgs {
		switch {
		case m.UpdateComponents != nil:
			s.applyComponents(m.UpdateComponents.Components)
		case m.UpdateDataModel != nil:
			s.applyDataModel(m.UpdateDataModel.Path, m.UpdateDataModel.Value)
		case m.DeleteSurface != nil:
			if m.DeleteSurface.SurfaceID == s.id {
				s.byID = make(map[string]a2ui.Component)
				s.rootID = ""
				s.focusables = nil
				s.focusIdx = 0
				return false
			}
		}
	}
	return s.rootID != ""
}

// applyComponents merges the given components into the surface's component map
// by ID: an update to component X replaces X, leaving siblings intact. It
// re-derives the root and focus ring, preserving focus on the same component
// if it survives the merge.
func (s *Surface) applyComponents(components []a2ui.Component) {
	// Remember which button held focus before the merge.
	var focusedID string
	if len(s.focusables) > 0 && s.focusIdx < len(s.focusables) {
		focusedID = s.focusables[s.focusIdx]
	}

	for _, c := range components {
		s.byID[c.ID] = c
	}
	s.rootID = s.deriveRootID()
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
// other component's child, falling back to the first in iteration order. It
// prefers the current rootID if it is still unreferenced, so adding new
// standalone components does not change the root.
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
	var firstID string
	for id := range s.byID {
		if firstID == "" || id < firstID {
			firstID = id
		}
	}
	for id := range s.byID {
		if !referenced[id] {
			return id
		}
	}
	return firstID
}
