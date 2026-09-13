package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestTenantIsolation(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
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

	// GetPatient cross-tenant isolation
	_, err = m.GetPatient("tenant-B", p1.Id)
	if err != patientdirectory.ErrNotFound {
		t.Errorf("expected ErrNotFound querying tenant-A patient under tenant-B, got %v", err)
	}

	// FindByRut cross-tenant isolation
	_, err = m.FindByRut("tenant-B", p1.Rut)
	if err != patientdirectory.ErrNotFound {
		t.Errorf("expected ErrNotFound querying tenant-A RUT under tenant-B, got %v", err)
	}

	// ListPatients tenant filtering
	listA, err := m.ListPatients("tenant-A", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("ListPatients tenant-A failed: %v", err)
	}
	if len(listA) != 1 || listA[0].Id != p1.Id {
		t.Errorf("expected listA to contain p1 only, got %v", listA)
	}

	listB, err := m.ListPatients("tenant-B", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("ListPatients tenant-B failed: %v", err)
	}
	if len(listB) != 1 || listB[0].Id != p2.Id {
		t.Errorf("expected listB to contain p2 only, got %v", listB)
	}
}
