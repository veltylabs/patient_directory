package tests

import (
	"testing"

	"github.com/veltylabs/patient_directory/ui"
)

func TestUI_Screens(t *testing.T) {
	mock := &mockCaller{}
	screens := ui.Screens(mock)
	if len(screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(screens))
	}
	s := screens[0]
	if err := s.Validate(); err != nil {
		t.Fatalf("screen validation failed: %v", err)
	}
	if s.ID != ui.ID {
		t.Errorf("ID = %q, want %q", s.ID, ui.ID)
	}
	if string(s.Permission) != "patient" {
		t.Errorf("Permission = %q, want patient", s.Permission)
	}
	if s.Presenter.Description() == "" {
		t.Errorf("presenter description is empty")
	}
}
