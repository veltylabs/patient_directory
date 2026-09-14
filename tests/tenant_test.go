package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestTenantIsolation(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-A",
		Rut:      "11111111-1",
		Name:     "Patient A",
		IsActive: true,
	})

	p2, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-B",
		Rut:      "22222222-2",
		Name:     "Patient B",
		IsActive: true,
	})

	// Aislamiento entre tenants para GetPatient
	_, err = m.GetPatient("tenant-B", p1.Id)
	if err != patientdirectory.ErrNotFound {
		t.Errorf("se esperaba ErrNotFound al consultar un paciente de tenant-A en tenant-B, se obtuvo %v", err)
	}

	// Aislamiento entre tenants para FindByRut
	_, err = m.FindByRut("tenant-B", p1.Rut)
	if err != patientdirectory.ErrNotFound {
		t.Errorf("se esperaba ErrNotFound al consultar un RUT de tenant-A en tenant-B, se obtuvo %v", err)
	}

	// Filtrado por tenant en ListPatients
	listA, err := m.ListPatients("tenant-A", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("ListPatients tenant-A falló: %v", err)
	}
	if len(listA) != 1 || listA[0].Id != p1.Id {
		t.Errorf("se esperaba que listA contuviera únicamente a p1, se obtuvo %v", listA)
	}

	listB, err := m.ListPatients("tenant-B", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("ListPatients tenant-B falló: %v", err)
	}
	if len(listB) != 1 || listB[0].Id != p2.Id {
		t.Errorf("se esperaba que listB contuviera únicamente a p2, se obtuvo %v", listB)
	}
}
