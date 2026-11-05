package tests

import (
	"errors"
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
)

func TestGetPatient(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		TenantId:     "t1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "12.345.678-5",
		Name:         "Ana",
		Phone:        "+56911223344",
		Email:        "ana@example.com",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	name, doc, phone, email, found, err := m.GetPatient("t1", "p1")
	if err != nil {
		t.Fatalf("GetPatient: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	if name != "Ana" || phone != "+56911223344" || email != "ana@example.com" {
		t.Errorf("unexpected fields: name=%s phone=%s email=%s", name, phone, email)
	}
	if doc == "" {
		t.Errorf("expected non-empty doc")
	}
}

func TestGetPatientMissing(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		TenantId:     "t1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "12.345.678-5",
		Name:         "Ana",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	_, _, _, _, found, err := m.GetPatient("t1", "zz")
	if err != nil || found {
		t.Errorf("expected found=false, err=nil; got found=%v err=%v", found, err)
	}

	_, _, _, _, found, err = m.GetPatient("t2", "p1")
	if err != nil || found {
		t.Errorf("expected found=false, err=nil for other tenant; got found=%v err=%v", found, err)
	}
}

func TestFindPatient(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		TenantId:     "t1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "12.345.678-5",
		Name:         "Ana",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	id, name, _, _, found, err := m.FindPatient("t1", "12345678-5")
	if err != nil {
		t.Fatalf("FindPatient: %v", err)
	}
	if !found || id != "p1" || name != "Ana" {
		t.Errorf("unexpected FindPatient: id=%s name=%s found=%v", id, name, found)
	}
}

func TestFindPatientMissing(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, _, _, _, found, err := m.FindPatient("t1", "11.111.111-1")
	if err != nil || found {
		t.Errorf("expected found=false, err=nil; got found=%v err=%v", found, err)
	}
}

func TestCreatePatient(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	id, err := m.CreatePatient("t1", "p2", "12.345.678-5", "Ana", "+56911223344", "ana@example.com")
	if err != nil {
		t.Fatalf("CreatePatient: %v", err)
	}
	if id != "p2" {
		t.Errorf("expected id p2, got %s", id)
	}

	readID, name, phone, email, found, err := m.FindPatient("t1", "12345678-5")
	if err != nil || !found {
		t.Fatalf("FindPatient failed: found=%v err=%v", found, err)
	}
	if readID != "p2" || name != "Ana" || phone != "+56911223344" || email != "ana@example.com" {
		t.Errorf("unexpected read patient data")
	}
}

func TestCreatePatientDuplicate(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		TenantId:     "t1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "12.345.678-5",
		Name:         "Ana",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	_, err = m.CreatePatient("t1", "p3", "12.345.678-5", "Ana 2", "", "")
	if !errors.Is(err, patientdirectory.ErrRutAlreadyExists) {
		t.Errorf("expected ErrRutAlreadyExists, got %v", err)
	}
}

