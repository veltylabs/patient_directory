package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestCreatePatient_AssignsIDAndNormalisesRut(t *testing.T) {
	m, pub, idGen, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12.345.678-k",
		Name:     "John Doe",
		IsActive: true,
	}

	created, err := m.CreatePatient(p)
	if err != nil {
		t.Fatalf("CreatePatient falló: %v", err)
	}

	if created.Id != "id-1" {
		t.Errorf("se esperaba ID 'id-1', se obtuvo '%s'", created.Id)
	}
	if created.Rut != "12345678K" {
		t.Errorf("se esperaba RUT normalizado '12345678K', se obtuvo '%s'", created.Rut)
	}
	if created.UpdatedAt == 0 {
		t.Errorf("se esperaba que UpdatedAt estuviera establecido")
	}

	if len(pub.published) != 1 {
		t.Fatalf("se esperaba 1 evento publicado, se obtuvo %d", len(pub.published))
	}
	if pub.published[0].Topic != patientdirectory.TopicPatientCreated {
		t.Errorf("se esperaba el topic %s, se obtuvo %s", patientdirectory.TopicPatientCreated, pub.published[0].Topic)
	}
	_ = idGen
}

func TestCreatePatient_DuplicateRutSameTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 1 falló: %v", err)
	}

	p2 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12.345.678-k",
		Name:     "Jane Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p2)
	if err != patientdirectory.ErrRutAlreadyExists {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
}

func TestCreatePatient_SameRutDifferentTenant(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 1 falló: %v", err)
	}

	p2 := patientdirectory.Patient{
		TenantId: "tenant-2",
		Rut:      "12345678-K",
		Name:     "Jane Doe",
		IsActive: true,
	}
	created2, err := m.CreatePatient(p2)
	if err != nil {
		t.Fatalf("CreatePatient 2 debería tener éxito para un tenant distinto, se obtuvo: %v", err)
	}
	if created2.TenantId != "tenant-2" {
		t.Errorf("se esperaba tenant-2, se obtuvo %s", created2.TenantId)
	}
}

func TestCreatePatient_MissingRutOrName(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	pNoRut := patientdirectory.Patient{
		TenantId: "tenant-1",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoRut)
	if _, ok := err.(patientdirectory.ValidationError); !ok {
		t.Errorf("se esperaba ValidationError para RUT faltante, se obtuvo %v", err)
	}

	pNoName := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoName)
	if _, ok := err.(patientdirectory.ValidationError); !ok {
		t.Errorf("se esperaba ValidationError para Name faltante, se obtuvo %v", err)
	}
}

func TestUpdatePatient_RutTakenByAnother(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
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

	// Intentar actualizar el RUT de p2 al RUT de p1
	p2.Rut = "11111111-1"
	_, err = m.UpdatePatient(p2)
	if err != patientdirectory.ErrRutAlreadyExists {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
	_ = p1
}

func TestUpdatePatient_KeepingOwnRut(t *testing.T) {
	m, pub, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
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
		t.Fatalf("UpdatePatient manteniendo el propio RUT falló: %v", err)
	}
	if updated.Name != "Patient 1 Updated" {
		t.Errorf("se esperaba nombre actualizado, se obtuvo %s", updated.Name)
	}

	var found bool
	for _, e := range pub.published {
		if e.Topic == patientdirectory.TopicPatientUpdated {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("se esperaba que el evento TopicPatientUpdated estuviera publicado")
	}
}

func TestFindByRut_NormalisesInput(t *testing.T) {
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	found, err := m.FindByRut("tenant-1", "12.345.678-k")
	if err != nil {
		t.Fatalf("FindByRut falló: %v", err)
	}
	if found.Id != created.Id {
		t.Errorf("se esperaba ID %s, se obtuvo %s", created.Id, found.Id)
	}
}

func TestDeactivatePatient(t *testing.T) {
	m, pub, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	})

	err = m.DeactivatePatient("tenant-1", created.Id)
	if err != nil {
		t.Fatalf("DeactivatePatient falló: %v", err)
	}

	p, err := m.GetPatient("tenant-1", created.Id)
	if err != nil {
		t.Fatalf("GetPatient falló: %v", err)
	}
	if p.IsActive {
		t.Errorf("se esperaba que is_active fuera false")
	}

	var found bool
	for _, e := range pub.published {
		if e.Topic == patientdirectory.TopicPatientDeactivated {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("se esperaba que el evento TopicPatientDeactivated estuviera publicado")
	}
}
