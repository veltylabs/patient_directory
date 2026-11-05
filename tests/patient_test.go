package tests

import (
	"encoding/json"
	"testing"
	"webtyp.com/model"

	patientdirectory "github.com/veltylabs/patient_directory"
)

func TestCreatePatient_AssignsIDAndNormalisesRut(t *testing.T) {
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	id, err := m.CreatePatient("tenant-1", "id-1", "12.345.678-k", "John Doe", "+56911223344", "john@example.com")
	if err != nil {
		t.Fatalf("CreatePatient falló: %v", err)
	}
	if id != "id-1" {
		t.Errorf("se esperaba ID 'id-1', se obtuvo '%s'", id)
	}

	foundID, name, phone, email, found, err := m.FindPatient("tenant-1", "12345678K")
	if err != nil || !found {
		t.Fatalf("FindPatient falló: found=%v err=%v", found, err)
	}
	if foundID != "id-1" || name != "John Doe" || phone != "+56911223344" || email != "john@example.com" {
		t.Errorf("datos inesperados del paciente: id=%s name=%s phone=%s email=%s", foundID, name, phone, email)
	}

	if len(pub.published) != 1 {
		t.Fatalf("se esperaba 1 evento publicado, se obtuvo %d", len(pub.published))
	}
	if pub.published[0].Topic != patientdirectory.TopicPatientCreated {
		t.Errorf("se esperaba el topic %s, se obtuvo %s", patientdirectory.TopicPatientCreated, pub.published[0].Topic)
	}
}

func TestCreatePatient_IdempotentRetry(t *testing.T) {
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

	created1, err := m.ImportPatient(p1)
	if err != nil {
		t.Fatalf("ImportPatient 1 falló: %v", err)
	}

	// Reintento con el mismo ID, en el mismo tenant
	created2, err := m.ImportPatient(p1)
	if err != nil {
		t.Fatalf("ImportPatient 2 (reintento) falló: %v", err)
	}

	if created1.Id != created2.Id {
		t.Errorf("se esperaba el mismo ID en el reintento")
	}

	name, _, _, _, found, err := m.GetPatient("tenant-1", "id-1")
	if err != nil || !found || name != "John Doe" {
		t.Fatalf("GetPatient falló tras reintento: found=%v err=%v name=%s", found, err, name)
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

	_, err = m.ImportPatient(p1)
	if err != nil {
		t.Fatalf("ImportPatient 1 falló: %v", err)
	}

	p2 := patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-2", // Diferente tenant, mismo ID
		Rut:      "11111111-1",
		Name:     "Jane Doe",
		IsActive: true,
	}

	_, err = m.ImportPatient(p2)
	if err == nil || model.KindOf(err) != model.ErrorConflict {
		t.Errorf("se esperaba ErrIdTaken, se obtuvo %v", err)
	}
}

func TestCreatePatient_DuplicateRutSameTenant(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	_, err = m.CreatePatient("tenant-1", "id-1", "12345678-K", "John Doe", "", "")
	if err != nil {
		t.Fatalf("CreatePatient 1 falló: %v", err)
	}

	_, err = m.CreatePatient("tenant-1", "id-2", "12.345.678-k", "Jane Doe", "", "")
	if err == nil || model.KindOf(err) != model.ErrorConflict {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
}

func TestCreatePatient_SameRutDifferentTenant(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	_, err = m.CreatePatient("tenant-1", "id-1", "12345678-K", "John Doe", "", "")
	if err != nil {
		t.Fatalf("CreatePatient 1 falló: %v", err)
	}

	id2, err := m.CreatePatient("tenant-2", "id-2", "12345678-K", "Jane Doe", "", "")
	if err != nil {
		t.Fatalf("CreatePatient 2 debería tener éxito para un tenant distinto, se obtuvo: %v", err)
	}
	if id2 != "id-2" {
		t.Errorf("se esperaba id-2, se obtuvo %s", id2)
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
	_, err = m.ImportPatient(pNoId)
	if err == nil || model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("se esperaba ErrIdRequired, se obtuvo %v", err)
	}

	pNoRut := patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Name:     "John Doe",
		IsActive: true,
	}
	_, err = m.ImportPatient(pNoRut)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("se esperaba ValidationError para RUT faltante, se obtuvo %v", err)
	}

	pNoName := patientdirectory.Patient{
		Id:       "id-2",
		TenantId: "tenant-1",
		Rut:      "12345678-K",
		IsActive: true,
	}
	_, err = m.ImportPatient(pNoName)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("se esperaba ValidationError para Name faltante, se obtuvo %v", err)
	}
}

func TestUpdatePatient_RutTakenByAnother(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.ImportPatient(patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Rut:      "11111111-1",
		Name:     "Patient 1",
		IsActive: true,
	})
	p2, _ := m.ImportPatient(patientdirectory.Patient{
		Id:       "id-2",
		TenantId: "tenant-1",
		Rut:      "22222222-2",
		Name:     "Patient 2",
		IsActive: true,
	})

	// Intentar actualizar el RUT de p2 al RUT de p1 vía ImportPatient
	p2.Rut = "11111111-1"
	_, err = m.ImportPatient(p2)
	if err == nil || model.KindOf(err) != model.ErrorConflict {
		t.Errorf("se esperaba ErrRutAlreadyExists, se obtuvo %v", err)
	}
	_ = p1
}

func TestUpdatePatient_KeepingOwnRut(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	p1, _ := m.ImportPatient(patientdirectory.Patient{
		Id:       "id-1",
		TenantId: "tenant-1",
		Rut:      "11111111-1",
		Name:     "Patient 1",
		IsActive: true,
	})

	p1.Name = "Patient 1 Updated"
	updated, err := m.ImportPatient(p1)
	if err != nil {
		t.Fatalf("ImportPatient manteniendo el propio RUT falló: %v", err)
	}
	if updated.Name != "Patient 1 Updated" {
		t.Errorf("se esperaba nombre actualizado, se obtuvo %s", updated.Name)
	}
}

func TestFindPatient_NormalisesInput(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	_, _ = m.CreatePatient("tenant-1", "id-1", "12345678-K", "John Doe", "", "")

	id, name, _, _, found, err := m.FindPatient("tenant-1", "12.345.678-k")
	if err != nil {
		t.Fatalf("FindPatient falló: %v", err)
	}
	if !found || id != "id-1" || name != "John Doe" {
		t.Errorf("se esperaba id-1 John Doe, se obtuvo id=%s name=%s found=%v", id, name, found)
	}
}

func TestDeactivatePatient(t *testing.T) {
	m, pub, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	id, _ := m.CreatePatient("tenant-1", "id-1", "12345678-K", "John Doe", "", "")

	reg := newMockRegistry()
	m.MountOperations(reg)
	deactHandler := reg.ops[patientdirectory.OpDeactivatePatient]

	deactReq, _ := json.Marshal(map[string]any{
		"tenant_id": "tenant-1",
		"id":        id,
	})
	ctxDeact := &mockContext{body: deactReq}
	deactHandler(ctxDeact)
	if ctxDeact.statusCode != 200 {
		t.Fatalf("deactivate falló con código %d", ctxDeact.statusCode)
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

func TestItemsDeclaredOnFields(t *testing.T) {
	rec := patientdirectory.Patient{
		Id:   "123",
		Rut:  "12345678-5",
		Name: "Ana Torres",
	}

	item := rec.Item()

	if item.Label != "Ana Torres" {
		t.Errorf("expected Label to be 'Ana Torres', got %q", item.Label)
	}
	if item.Description != "12345678-5" {
		t.Errorf("expected Description to be '12345678-5', got %q", item.Description)
	}
	if item.Badge != "" {
		t.Errorf("expected Badge to be empty, got %q", item.Badge)
	}
}
