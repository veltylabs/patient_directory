package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

type directoryReader interface {
	ClientExists(tenantId, clientId string) (bool, error)
}

var _ directoryReader = (*patientdirectory.Module)(nil)

func TestClientExists_True(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	exists, err := m.ClientExists("tenant-1", p.Id)
	if err != nil {
		t.Fatalf("ClientExists returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected exists=true, got false")
	}
}

func TestClientExists_UnknownID(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	exists, err := m.ClientExists("tenant-1", "non-existent-id")
	if err != nil {
		t.Fatalf("ClientExists returned error: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false, got true")
	}
}

func TestClientExists_WrongTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	exists, err := m.ClientExists("tenant-2", p.Id)
	if err != nil {
		t.Fatalf("ClientExists returned error: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false for wrong tenant, got true")
	}
}

func TestClientExists_InactivePatientStillExists(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	_ = m.DeactivatePatient("tenant-1", p.Id)

	exists, err := m.ClientExists("tenant-1", p.Id)
	if err != nil {
		t.Fatalf("ClientExists returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected inactive patient to still exist (exists=true), got false")
	}
}

func TestClientExists_EmptyArgs(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	exists, err := m.ClientExists("", "")
	if err != nil {
		t.Fatalf("ClientExists returned error: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false for empty args, got true")
	}
}
