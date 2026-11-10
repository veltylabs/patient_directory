package patientdirectory

import (
	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// Create es la regla de negocio de crear patient: normaliza y valida el documento y rechaza un documento ya registrado en esta clínica.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := m.checkDocument(p); err != nil {
		return 0, err
	}
	if err := m.sameDocument(tx, p); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}

// sameDocument rechaza otro paciente de esta clínica con el mismo documento (el tenant lo pone el Store).
func (m *Module) sameDocument(tx crud.Tx, p *Patient) error {
	if p.Rut == "" || p.DocumentType == kinds.DocumentUnverified {
		return nil
	}
	var taken bool
	err := tx.ReadAll(func() model.Model { return &Patient{} }, crud.Query{Conditions: []storage.Condition{
		storage.Eq(Patient_.DocumentType, p.DocumentType), storage.Eq(Patient_.Rut, p.Rut),
	}}, func(r model.Model) {
		other := r.(*Patient)
		if other.Id != p.Id && other.DocumentCountry == p.DocumentCountry {
			taken = true
		}
	})
	if err != nil {
		return err
	}
	if taken {
		return &model.FieldError{Field: Patient_.Rut, Err: ErrRutAlreadyExists}
	}
	return nil
}
