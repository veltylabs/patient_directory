package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestCreatePatient_AssignsIDAndNormalisesRut(t *testing.T) {
	m, pub, idGen, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12.345.678-k",
		Name:     "John Doe",
		IsActive: true,
	}

	created, err := m.CreatePatient(p)
	if err != nil {
		t.Fatalf("CreatePatient failed: %v", err)
	}

	if created.Id != "id-1" {
		t.Errorf("expected ID 'id-1', got '%s'", created.Id)
	}
	if created.Rut != "12345678K" {
		t.Errorf("expected normalised RUT '12345678K', got '%s'", created.Rut)
	}
	if created.UpdatedAt == 0 {
		t.Errorf("expected UpdatedAt to be set")
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(pub.published))
	}
	if pub.published[0].Topic != patientdirectory.TopicPatientCreated {
		t.Errorf("expected topic %s, got %s", patientdirectory.TopicPatientCreated, pub.published[0].Topic)
	}
	_ = idGen
}

func TestCreatePatient_DuplicateRutSameTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p1 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 1 failed: %v", err)
	}

	p2 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12.345.678-k",
		Name:     "Jane Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p2)
	if err != patientdirectory.ErrRutAlreadyExists {
		t.Errorf("expected ErrRutAlreadyExists, got %v", err)
	}
}

func TestCreatePatient_SameRutDifferentTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p1 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 1 failed: %v", err)
	}

	p2 := patientdirectory.Patient{
		TenantId: "tenant-2",
		Rut:      "12345678-K",
		Name:     "Jane Doe",
		IsActive: true,
	}
	created2, err := m.CreatePatient(p2)
	if err != nil {
		t.Fatalf("CreatePatient 2 should succeed for different tenant, got: %v", err)
	}
	if created2.TenantId != "tenant-2" {
		t.Errorf("expected tenant-2, got %s", created2.TenantId)
	}
}

func TestCreatePatient_MissingRutOrName(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	pNoRut := patientdirectory.Patient{
		TenantId: "tenant-1",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoRut)
	if _, ok := err.(patientdirectory.ValidationError); !ok {
		t.Errorf("expected ValidationError for missing RUT, got %v", err)
	}

	pNoName := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoName)
	if _, ok := err.(patientdirectory.ValidationError); !ok {
		t.Errorf("expected ValidationError for missing Name, got %v", err)
	}
}

func TestUpdatePatient_RutTakenByAnother(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p1, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "11111111-1",
		Name:     "Patient 1",
		IsActive: true,
	})
	p2, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "22222222-2",
		Name:     "Patient 2",
		IsActive: true,
	})

	// Try updating p2's RUT to p1's RUT
	p2.Rut = "11111111-1"
	_, err = m.UpdatePatient(p2)
	if err != patientdirectory.ErrRutAlreadyExists {
		t.Errorf("expected ErrRutAlreadyExists, got %v", err)
	}
	_ = p1
}

func TestUpdatePatient_KeepingOwnRut(t *testing.T) {
	m, pub, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	p1, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "11111111-1",
		Name:     "Patient 1",
		IsActive: true,
	})

	p1.Name = "Patient 1 Updated"
	updated, err := m.UpdatePatient(p1)
	if err != nil {
		t.Fatalf("UpdatePatient keeping own RUT failed: %v", err)
	}
	if updated.Name != "Patient 1 Updated" {
		t.Errorf("expected updated name, got %s", updated.Name)
	}

	var found bool
	for _, e := range pub.published {
		if e.Topic == patientdirectory.TopicPatientUpdated {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TopicPatientUpdated event published")
	}
}

func TestFindByRut_NormalisesInput(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	found, err := m.FindByRut("tenant-1", "12.345.678-k")
	if err != nil {
		t.Fatalf("FindByRut failed: %v", err)
	}
	if found.Id != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, found.Id)
	}
}

func TestDeactivatePatient(t *testing.T) {
	m, pub, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	err = m.DeactivatePatient("tenant-1", created.Id)
	if err != nil {
		t.Fatalf("DeactivatePatient failed: %v", err)
	}

	p, err := m.GetPatient("tenant-1", created.Id)
	if err != nil {
		t.Fatalf("GetPatient failed: %v", err)
	}
	if p.IsActive {
		t.Errorf("expected is_active to be false")
	}

	var found bool
	for _, e := range pub.published {
		if e.Topic == patientdirectory.TopicPatientDeactivated {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TopicPatientDeactivated event published")
	}
}
