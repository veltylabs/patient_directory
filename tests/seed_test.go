package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/seed"
	"webtyp.com/auth/trusted_ip"
	"webtyp.com/events/mock"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

func TestSeed_Load(t *testing.T) {
	db := orm.New(mem.New())
	mod, err := patientdirectory.New(db, patientdirectory.Deps{
		IDs:         &testIDGen{},
		Publisher:   &mock.Broker{},
		TenantID:    "demo",
		ValidateRUT: trustedip.ValidateRUT,
	})
	if err != nil {
		t.Fatalf("patientdirectory.New: %v", err)
	}

	data, err := seed.Load(mod, "demo")
	if err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	if len(data.Patients) != 3 {
		t.Fatalf("se esperaban 3 pacientes en seed.Data, se obtuvieron %d", len(data.Patients))
	}

	patients, err := mod.ListPatients("demo", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("mod.ListPatients: %v", err)
	}

	if len(patients) != 3 {
		t.Fatalf("se esperaban 3 pacientes en el módulo, se obtuvieron %d", len(patients))
	}
}
