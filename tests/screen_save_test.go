package tests

import (
	"strings"
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
	"github.com/veltylabs/patient_directory/ui"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/view"
)

func TestScreenSave(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	caller := view.NewRecords().Caller(crud.LoopbackActingAs(nil, "user1", m))
	screens := ui.Screens(caller)
	if len(screens) == 0 {
		t.Fatal("expected screens")
	}

	saver, ok := screens[0].Presenter.(view.Saver)
	if !ok {
		t.Fatal("presenter is not a view.Saver")
	}

	p := &patientdirectory.Patient{
		Id:           "p-test-1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "11111111-1",
		Name:         "Test Patient",
	}

	var saveErr error
	saver.Save([]model.Model{p}, func(err error) { saveErr = err })
	if saveErr != nil {
		t.Fatalf("unexpected err on save: %v", saveErr)
	}

	p2 := &patientdirectory.Patient{
		Id:           "p-test-2",
		DocumentType: kinds.DocumentRUT,
		Rut:          "11111111-1",
		Name:         "Duplicate RUT",
	}
	saver.Save([]model.Model{p2}, func(err error) { saveErr = err })
	if model.FieldOf(saveErr) != "rut" || !strings.Contains(saveErr.Error(), string(patientdirectory.ErrRutAlreadyExists)) {
		t.Fatalf("expected conflict on duplicate rut, got %v", saveErr)
	}
}
