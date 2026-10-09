package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestCreatePatient_AssignsIDAndNormalisesRut(t *testing.T) {
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p := patientdirectory.Patient{
		Id:       "id-1",
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
}

func TestCreatePatient_IdempotentRetry(t *testing.T) {
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}

	created1, err := m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 1 falló: %v", err)
	}

	// Reintento con el mismo ID, en el mismo tenant
	created2, err := m.CreatePatient(p1)
	if err != nil {
		t.Fatalf("CreatePatient 2 (reintento) falló: %v", err)
	}

	if created1.Id != created2.Id {
		t.Errorf("se esperaba el mismo ID en el reintento")
	}

	// Verificar que solo haya un paciente en ListPatients
	patients, err := m.ListPatients("tenant-1", patientdirectory.PatientFilter{})
	if err != nil {
		t.Fatalf("ListPatients falló: %v", err)
	}
	if len(patients) != 1 {
		t.Fatalf("se esperaba 1 fila, se obtuvieron %d", len(patients))
	}

	// Verificar que se haya publicado un solo evento
	if len(pub.published) != 1 {
		t.Fatalf("se esperaba 1 evento publicado, se obtuvo %d", len(pub.published))
	}
}

func TestCreatePatient_SameIdDifferentTenant(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		Id:       "id-1",
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
		Id:       "id-1",
		TenantId: "tenant-2", // Diferente tenant, mismo ID
		Rut:      "11111111-1",
		Name:     "Jane Doe",
		IsActive: true,
	}

	_, err = m.CreatePatient(p2)
	if err == nil || err.Error() != patientdirectory.ErrIdTaken.Error() {
		t.Errorf("se esperaba ErrIdTaken, se obtuvo %v", err)
	}
}

func TestCreatePatient_DuplicateRutSameTenant(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		Id:       "id-1",
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
		Id:       "id-2",
		TenantId: "tenant-1",
		Rut:      "12.345.678-k",
		Name:     "Jane Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(p2)
	if err == nil || err.Error() != "patient rut already exists in this tenant" {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
}

func TestCreatePatient_SameRutDifferentTenant(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1 := patientdirectory.Patient{
		Id:       "id-1",
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
		Id:       "id-2",
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

func TestCreatePatient_MissingIdRutOrName(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	pNoId := patientdirectory.Patient{
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoId)
	if err == nil || err.Error() != patientdirectory.ErrIdRequired.Error() {
		t.Errorf("se esperaba ErrIdRequired, se obtuvo %v", err)
	}

	pNoRut := patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.CreatePatient(pNoRut)
	if _, ok := err.(patientdirectory.ValidationError); !ok {
		t.Errorf("se esperaba ValidationError para RUT faltante, se obtuvo %v", err)
	}

	pNoName := patientdirectory.Patient{
		Id:       "id-2",
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
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.CreatePatient(patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Rut:      "11111111-1",
		Name:     "Patient 1",
		IsActive: true,
	})
	p2, _ := m.CreatePatient(patientdirectory.Patient{
		Id:       "id-2",
		TenantId: "tenant-1",
		Rut:      "22222222-2",
		Name:     "Patient 2",
		IsActive: true,
	})

	// Intentar actualizar el RUT de p2 al RUT de p1
	p2.Rut = "11111111-1"
	_, err = m.UpdatePatient(p2)
	if err == nil || err.Error() != "patient rut already exists in this tenant" {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
	_ = p1
}

func TestUpdatePatient_KeepingOwnRut(t *testing.T) {
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.CreatePatient(patientdirectory.Patient{
		Id:       "id-1",
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
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		Id:       "id-1",
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
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	created, _ := m.CreatePatient(patientdirectory.Patient{
		Id:       "id-1",
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

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{patientdirectory.ErrNotFound, "patient not found"},
		{patientdirectory.ErrRutAlreadyExists, "patient rut already exists in this tenant"},
		{patientdirectory.ErrRutRequired, "patient rut is required"},
		{patientdirectory.ErrNameRequired, "patient name is required"},
		{patientdirectory.ErrTenantRequired, "patient tenant_id is required"},
		{patientdirectory.ErrIdRequired, "patient id is required"},
		{patientdirectory.ErrIdTaken, "patient id belongs to another tenant"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("se esperaba el texto '%s', se obtuvo '%s'", tt.expected, tt.err.Error())
		}
	}
}
