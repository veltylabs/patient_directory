package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// El puerto que appointment_booking declara de su lado. Re-declarado localmente a
// propósito: este repositorio no debe depender de un módulo de agendamiento para
// probar que satisface una interfaz estructural.
type directoryReader interface {
	ClientExists(tenantId, clientId string) (bool, error)
}

var _ directoryReader = (*patientdirectory.Module)(nil)

func TestClientExists_True(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	exists, err := m.ClientExists("tenant-1", p.Id)
	if err != nil {
		t.Fatalf("ClientExists retornó error: %v", err)
	}
	if !exists {
		t.Errorf("se esperaba exists=true, se obtuvo false")
	}
}

func TestClientExists_UnknownID(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	exists, err := m.ClientExists("tenant-1", "non-existent-id")
	if err != nil {
		t.Fatalf("ClientExists retornó error: %v", err)
	}
	if exists {
		t.Errorf("se esperaba exists=false, se obtuvo true")
	}
}

func TestClientExists_WrongTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	exists, err := m.ClientExists("tenant-2", p.Id)
	if err != nil {
		t.Fatalf("ClientExists retornó error: %v", err)
	}
	if exists {
		t.Errorf("se esperaba exists=false para tenant incorrecto, se obtuvo true")
	}
}

func TestClientExists_InactivePatientStillExists(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
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
		t.Fatalf("ClientExists retornó error: %v", err)
	}
	if !exists {
		t.Errorf("se esperaba que el paciente inactivo aún existiera (exists=true), se obtuvo false")
	}
}

func TestClientExists_EmptyArgs(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	exists, err := m.ClientExists("", "")
	if err != nil {
		t.Fatalf("ClientExists retornó error: %v", err)
	}
	if exists {
		t.Errorf("se esperaba exists=false para argumentos vacíos, se obtuvo true")
	}
}
