package patientdirectory

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// ImportPatient stores a patient from a previous system with the id it had there: it creates it, or updates every
// field when that id already exists, so importing twice leaves the same row. It validates exactly like
// CreatePatient (name, document rules, sex, unique document) and publishes no event.
//
// Example: p, err := dir.ImportPatient(Patient{Id: "1640011384447734400", TenantId: t, DocumentType: kinds.DocumentRUT, Rut: "12345678-5", Name: "Ana"})
func (m *Module) ImportPatient(p Patient) (Patient, error) {
	if p.Id == "" {
		return Patient{}, &model.FieldError{Field: "id", Err: ErrIdRequired}
	}
	if p.TenantId == "" {
		return Patient{}, &model.FieldError{Field: "tenant_id", Err: ErrTenantRequired}
	}
	if p.Name == "" {
		return Patient{}, &model.FieldError{Field: "name", Err: ErrNameRequired}
	}

	if err := m.checkDocument(&p); err != nil {
		return Patient{}, err
	}

	if p.Sex != "" && p.Sex != "m" && p.Sex != "f" {
		return Patient{}, &model.FieldError{Field: "sex", Err: ErrSex}
	}

	if err := m.findSameDocument(p); err != nil {
		return Patient{}, err
	}

	var existing Patient
	qb := m.db.Query(&existing).Where(Patient_.Id).Eq(p.Id)
	_, err := ReadOnePatient(qb, &existing)

	p.UpdatedAt = time.NowNano()

	if err == nil {
		if existing.TenantId != p.TenantId {
			return Patient{}, ErrIdTaken
		}
		if err := m.db.Update(&p, orm.Eq(Patient_.Id, p.Id), orm.Eq(Patient_.TenantId, p.TenantId)); err != nil {
			return Patient{}, err
		}
		return p, nil
	} else if orm.IsNotFound(err) {
		if err := m.db.Create(&p); err != nil {
			return Patient{}, err
		}
		return p, nil
	} else {
		return Patient{}, err
	}
}
