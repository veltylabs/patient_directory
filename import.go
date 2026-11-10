package patientdirectory

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/time"
)

var importUpdateFields = []string{
	Patient_.Rut,
	Patient_.DocumentType,
	Patient_.DocumentCountry,
	Patient_.Name,
	Patient_.Birthdate,
	Patient_.BirthdateEstimated,
	Patient_.Sex,
	Patient_.Phone,
	Patient_.Email,
	Patient_.Address,
	Patient_.Notes,
	Patient_.IsActive,
	Patient_.UpdatedAt,
}

// ImportPatient guarda un paciente de un sistema anterior con el ID que tenía allí:
// lo crea, o actualiza todos los campos si ese ID ya existe, de modo que importar
// dos veces deja la misma fila. Valida exactamente como CreatePatient y no publica eventos.
func (m *Module) ImportPatient(p Patient) (Patient, error) {
	if p.Id == "" {
		return Patient{}, &model.FieldError{Field: Patient_.Id, Err: ErrIdRequired}
	}
	if p.TenantId != "" && p.TenantId != m.tenantID {
		return Patient{}, ErrIdTaken
	}
	p.TenantId = m.tenantID
	if err := m.checkDocument(&p); err != nil {
		return Patient{}, err
	}
	if err := model.ValidateFields(model.ActionCreate, &p); err != nil {
		return Patient{}, err
	}
	p.UpdatedAt = time.NowNano()

	err := m.store.Atomic(func(tx crud.Tx) error {
		if err := m.sameDocument(tx, &p); err != nil {
			return err
		}
		var existing Patient
		err := tx.ReadOne(&existing, p.Id)
		if err == nil {
			return tx.UpdateFields(&p, p.Id, importUpdateFields)
		}
		if model.KindOf(err) == model.ErrorNotFound {
			return tx.Create(&p)
		}
		return err
	})
	if err != nil {
		return Patient{}, err
	}
	return p, nil
}
