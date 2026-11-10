package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
)

func TestImportPatientCreatesThenUpdates(t *testing.T) {
	m, pub, _ := setupModuleWithTenant("t")

	p := patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "Ana 1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "1",
	}

	p1, err := m.ImportPatient(p)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Name != "Ana 1" {
		t.Errorf("expected Ana 1, got %s", p1.Name)
	}

	p.Name = "Ana 2"
	p2, err := m.ImportPatient(p)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Name != "Ana 2" {
		t.Errorf("expected Ana 2, got %s", p2.Name)
	}

	if len(pub.published) != 0 {
		t.Errorf("expected 0 events, got %d", len(pub.published))
	}
}

func TestImportPatientUnverified(t *testing.T) {
	m, pub, _ := setupModuleWithTenant("t")

	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "Ana",
		DocumentType: kinds.DocumentUnverified,
		Rut:          "12345678-0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Rut != "12345678-0" {
		t.Errorf("expected 12345678-0, got %s", p.Rut)
	}

	if len(pub.published) != 0 {
		t.Errorf("expected 0 events, got %d", len(pub.published))
	}
}
