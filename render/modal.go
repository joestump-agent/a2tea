package render

// Modal open/close state. A Modal component is itself the focusable element
// (its trigger child is drawn as the modal's chrome — see renderModal), and
// Enter on the focused modal toggles it open. Open modals are tracked as a
// stack in the order they were opened so Esc closes the innermost one first
// (a modal's content may legally contain another modal).
//
// Open state lives beside fieldValues as user-interaction state: it survives
// Apply merges (an updateComponents that leaves a modal in place does not
// slam it shut) and is cleared by deleteSurface.

// modalOpen reports whether the modal with the given component ID is open.
func (s *Surface) modalOpen(id string) bool {
	for _, open := range s.openModals {
		if open == id {
			return true
		}
	}
	return false
}

// HasOpenModal reports whether any modal on the surface is currently open,
// i.e. Esc is a close-the-modal command rather than free for the host.
// Hosts (and a2tea.Standalone) use this to decide whether Esc should quit
// or be forwarded to the surface. It mirrors EditingText.
func (s *Surface) HasOpenModal() bool {
	return len(s.openModals) > 0
}

// toggleModal opens the modal when closed and closes it when open, then
// re-derives the focus ring: content focusables exist only while the modal
// is open. Focus stays on the modal itself, which survives the toggle.
func (s *Surface) toggleModal(id string) {
	if s.modalOpen(id) {
		s.removeOpenModal(id)
	} else {
		s.openModals = append(s.openModals, id)
	}
	s.refreshFocusables(id)
}

// closeTopModal closes the most recently opened modal and returns its ID,
// or ("", false) when no modal is open.
func (s *Surface) closeTopModal() (string, bool) {
	if len(s.openModals) == 0 {
		return "", false
	}
	id := s.openModals[len(s.openModals)-1]
	s.openModals = s.openModals[:len(s.openModals)-1]
	return id, true
}

// removeOpenModal deletes id from the open stack wherever it sits.
func (s *Surface) removeOpenModal(id string) {
	kept := s.openModals[:0]
	for _, open := range s.openModals {
		if open != id {
			kept = append(kept, open)
		}
	}
	s.openModals = kept
}

// pruneOpenModals drops open state for components that no longer exist as
// modals after an Apply merge, so a replaced or deleted modal cannot leave
// phantom open state behind.
func (s *Surface) pruneOpenModals() {
	if len(s.openModals) == 0 {
		return
	}
	kept := s.openModals[:0]
	for _, id := range s.openModals {
		if c, ok := s.byID[id]; ok && c.Modal != nil {
			kept = append(kept, id)
		}
	}
	s.openModals = kept
}

// refreshFocusables re-derives the focus ring after an open/close state
// change, keeping focus on the component with the given ID when it survives
// (it falls back to the first focusable otherwise — e.g. focus was inside
// content that just closed).
func (s *Surface) refreshFocusables(focusID string) {
	s.focusables = s.collectFocusables()
	s.focusIdx = 0
	for i, id := range s.focusables {
		if id == focusID {
			s.focusIdx = i
			break
		}
	}
}
