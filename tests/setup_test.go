package tests

import (
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
	cleaned := strings.ReplaceAll(rut, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ToUpper(cleaned)
	if cleaned == "" {
		return "", patientdirectory.ErrRutRequired
	}
	return cleaned, nil
}

func setupModule() (*patientdirectory.Module, *fakePublisher, error) {
	conn := mem.New()
	db := orm.New(conn)

	pub := &fakePublisher{}

	deps := patientdirectory.Deps{
		Publisher:   pub,
		ValidateRUT: fakeValidateRUT,
		TenantID:    "test-tenant",
	}

	m, err := patientdirectory.New(db, deps)
	if err != nil {
		return nil, nil, err
	}
	return m, pub, nil
}
