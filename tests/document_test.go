package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/model"
)

func TestCreatePatientDefaultsToRUT(t *testing.T) {
	m, _, _ := setupModule()

	// bad check digit
	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:       "1",
		TenantId: "t",
		Name:     "N",
		Rut:      "bad",
	})
	if model.KindOf(err) != model.ErrorInvalid {
		t.Fatalf("expected invalid error, got %v", err)
	}
	fe, ok := err.(*model.FieldError)
	if !ok || fe.Field != "rut" {
		t.Fatalf("expected field error on rut, got %v", err)
	}

	// good RUT
	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:       "2",
		TenantId: "t",
		Name:     "N",
		Rut:      "1.234.567-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.DocumentType != kinds.DocumentRUT {
		t.Errorf("expected RUT type, got %s", p.DocumentType)
	}
	if p.Rut != "12345678" {
		t.Errorf("expected normalized RUT, got %s", p.Rut)
	}
}

func TestPassportNeedsNumber(t *testing.T) {
	m, _, _ := setupModule()

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "N",
		DocumentType: kinds.DocumentPassport,
		Rut:          "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	fe, ok := err.(*model.FieldError)
	if !ok || fe.Field != "rut" || fe.Err != patientdirectory.ErrDocumentNumber {
		t.Fatalf("expected %v, got %v", patientdirectory.ErrDocumentNumber, err)
	}
}

func TestPassportStoredAsTyped(t *testing.T) {
	m, _, _ := setupModule()

	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:              "1",
		TenantId:        "t",
		Name:            "N",
		DocumentType:    kinds.DocumentPassport,
		Rut:             "  ab123456 ",
		DocumentCountry: "ve",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Rut != "AB123456" {
		t.Errorf("expected AB123456, got %s", p.Rut)
	}
	if p.DocumentCountry != "VE" {
		t.Errorf("expected VE, got %s", p.DocumentCountry)
	}
}

func TestNoDocument(t *testing.T) {
	m, _, _ := setupModule()

	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "N",
		DocumentType: kinds.DocumentNone,
		Rut:          "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Rut != "" {
		t.Errorf("expected empty rut, got %s", p.Rut)
	}
}

func TestSameDocumentTwice(t *testing.T) {
	m, _, _ := setupModule()

	p := patientdirectory.Patient{
		Id:              "1",
		TenantId:        "t",
		Name:            "N",
		DocumentType:    kinds.DocumentPassport,
		Rut:             "123",
		DocumentCountry: "VE",
	}
	_, err := m.ImportPatient(p)
	if err != nil {
		t.Fatal(err)
	}

	p.Id = "2"
	_, err = m.ImportPatient(p)
	if err != patientdirectory.ErrRutAlreadyExists {
		t.Fatalf("expected ErrRutAlreadyExists, got %v", err)
	}
}

func TestSamePassportOtherCountry(t *testing.T) {
	m, _, _ := setupModule()

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:              "1",
		TenantId:        "t",
		Name:            "N",
		DocumentType:    kinds.DocumentPassport,
		Rut:             "123",
		DocumentCountry: "VE",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:              "2",
		TenantId:        "t",
		Name:            "N",
		DocumentType:    kinds.DocumentPassport,
		Rut:             "123",
		DocumentCountry: "PE",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTwoWithoutDocument(t *testing.T) {
	m, _, _ := setupModule()

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "1",
		TenantId:     "t",
		Name:         "N",
		DocumentType: kinds.DocumentNone,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "2",
		TenantId:     "t",
		Name:         "N",
		DocumentType: kinds.DocumentNone,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSexValues(t *testing.T) {
	m, _, _ := setupModule()

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:       "1",
		TenantId: "t",
		Name:     "N",
		Rut:      "1",
		Sex:      "x",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	fe, ok := err.(*model.FieldError)
	if !ok || fe.Field != "sex" || fe.Err != patientdirectory.ErrSex {
		t.Fatalf("expected %v, got %v", patientdirectory.ErrSex, err)
	}
}

func TestInactivePatientSaves(t *testing.T) {
	m, _, _ := setupModule()

	p, err := m.ImportPatient(patientdirectory.Patient{
		Id:       "1",
		TenantId: "t",
		Name:     "N",
		Rut:      "1",
		IsActive: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.IsActive {
		t.Errorf("expected false, got true")
	}
}
