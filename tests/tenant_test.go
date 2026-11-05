package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestTenantIsolation(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.ImportPatient(patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-A",
		Rut:      "11111111-1",
		Name:     "Patient A",
		IsActive: true,
	})

	p2, _ := m.ImportPatient(patientdirectory.Patient{
		Id:       "id-2",
		TenantId: "tenant-B",
		Rut:      "22222222-2",
		Name:     "Patient B",
		IsActive: true,
	})

	// Aislamiento entre tenants para GetPatient
	_, _, _, _, found, err := m.GetPatient("tenant-B", p1.Id)
	if err != nil || found {
		t.Errorf("se esperaba found=false al consultar un paciente de tenant-A en tenant-B, se obtuvo found=%v err=%v", found, err)
	}

	// Aislamiento entre tenants para FindPatient
	_, _, _, _, found, err = m.FindPatient("tenant-B", p1.Rut)
	if err != nil || found {
		t.Errorf("se esperaba found=false al consultar un RUT de tenant-A en tenant-B, se obtuvo found=%v err=%v", found, err)
	}

	// Cada tenant encuentra a su propio paciente
	_, _, _, _, found, err = m.GetPatient("tenant-A", p1.Id)
	if err != nil || !found {
		t.Errorf("tenant-A no encontró su paciente p1")
	}
	_, _, _, _, found, err = m.GetPatient("tenant-B", p2.Id)
	if err != nil || !found {
		t.Errorf("tenant-B no encontró su paciente p2")
	}
}
