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

	for _, p := range data.Patients {
		name, _, _, _, found, err := mod.GetPatient("demo", p.Id)
		if err != nil || !found || name != p.Name {
			t.Fatalf("mod.GetPatient(%s): found=%v err=%v name=%s", p.Id, found, err, name)
		}
	}

	// Segundo intento no debería duplicar ni fallar
	data2, err := seed.Load(mod, "demo")
	if err != nil {
		t.Fatalf("seed.Load 2: %v", err)
	}

	if len(data2.Patients) != 3 {
		t.Fatalf("se esperaban 3 pacientes en seed.Data en el segundo intento, se obtuvieron %d", len(data2.Patients))
	}

	for _, p := range data2.Patients {
		name, _, _, _, found, err := mod.GetPatient("demo", p.Id)
		if err != nil || !found || name != p.Name {
			t.Fatalf("mod.GetPatient(%s) 2: found=%v err=%v name=%s", p.Id, found, err, name)
		}
	}
}
