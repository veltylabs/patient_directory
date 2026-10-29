package tests

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"testing"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
	"webtyp.com/view"
)

func TestScreenSave(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	caller := loopback.ActingAs("user", "test-tenant", m)

	ops := view.Ops{Module: patientdirectory.ModelName, List: patientdirectory.OpListPatients, Save: patientdirectory.OpUpsertPatient}
	records := view.NewRecords().Caller(caller)
	lister := view.NewCallerLister(records, ops, func() model.ModelSlice { return &patientdirectory.PatientList{} })

	p := &patientdirectory.Patient{
		Id:       "p1",
		TenantId: "test-tenant",
		Rut:      "12345678-5",
		Name:     "Test",
		IsActive: true,
	}

	saver := lister.(view.Saver)
	var serr error
	saver.Save([]model.Model{p}, func(e error) { serr = e })

	if serr != nil {
		t.Fatalf("Save failed: %v", serr)
	}

	p2 := &patientdirectory.Patient{
		Id:       "p2",
		TenantId: "test-tenant",
		Rut:      "12345678-5",
		Name:     "Test 2",
		IsActive: true,
	}

	saver.Save([]model.Model{p2}, func(e error) { serr = e })
	if model.KindOf(serr) != model.ErrorConflict {
		t.Fatalf("expected conflict on duplicate rut, got %v", serr)
	}
}
