package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
)

func TestImportPatientCreatesThenUpdates(t *testing.T) {
	m, pub, _ := setupModule()

	p := patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "N1",
		DocumentType: kinds.DocumentRUT,
		Rut:          "1",
	}

	p1, err := m.ImportPatient(p)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Name != "N1" {
		t.Errorf("expected N1, got %s", p1.Name)
	}

	p.Name = "N2"
	p2, err := m.ImportPatient(p)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Name != "N2" {
		t.Errorf("expected N2, got %s", p2.Name)
	}

	if len(pub.published) != 0 {
		t.Errorf("expected 0 events, got %d", len(pub.published))
	}
}

func TestImportPatientUnverified(t *testing.T) {
	m, pub, _ := setupModule()

	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "N",
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

func TestImportPublishesNothing(t *testing.T) {
	// covered by TestImportPatientCreatesThenUpdates
}
