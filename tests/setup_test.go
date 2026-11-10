package tests

import (
	"errors"
	"strings"

	"webtyp.com/events"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"

	patientdirectory "github.com/veltylabs/patient_directory"
)

type fakePublisher struct {
	published []events.Event
}

func (f *fakePublisher) Publish(e events.Event) {
	f.published = append(f.published, e)
}

func fakeValidateRUT(rut string) (string, error) {
	if rut == "bad" {
		return "", errors.New("bad check digit")
	}
	cleaned := strings.ReplaceAll(rut, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ToUpper(cleaned)
	if cleaned == "" {
		return "", patientdirectory.ErrDocumentNumber
	}
	return cleaned, nil
}

func setupModule() (*patientdirectory.Module, *fakePublisher, error) {
	return setupModuleWithTenant("test-tenant")
}

func setupModuleWithTenant(tenantID string) (*patientdirectory.Module, *fakePublisher, error) {
	conn := mem.New()
	db := orm.New(conn)
	return setupModuleWithTenantAndDB(tenantID, db)
}

func setupModuleWithTenantAndDB(tenantID string, db *orm.DB) (*patientdirectory.Module, *fakePublisher, error) {
	pub := &fakePublisher{}
	deps := patientdirectory.Deps{
		Publisher:   pub,
		ValidateRUT: fakeValidateRUT,
		TenantID:    tenantID,
		IDs:         &testIDGen{},
	}
	m, err := patientdirectory.New(db, deps)
	if err != nil {
		return nil, nil, err
	}
	return m, pub, nil
}
