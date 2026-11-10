package patientdirectory

import (
	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// GetPatient lee un paciente por su id. Ausente u otro tenant ⇒ found=false, err=nil.
// Ejemplo: name, doc, phone, email, ok, err := m.GetPatient("t1", "p1")
func (m *Module) GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error) {
	if tenantID == "" || id == "" || tenantID != m.tenantID {
		return "", "", "", "", false, nil
	}
	var p Patient
	err = m.store.ReadOne(&p, id)
	if model.KindOf(err) == model.ErrorNotFound {
		return "", "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", "", false, err
	}
	return p.Name, p.Rut, p.Phone, p.Email, true, nil
}

// FindPatient busca un paciente por su número de documento (RUT). Ausente ⇒ found=false, err=nil.
// Ejemplo: id, name, phone, email, ok, err := m.FindPatient("t1", "12.345.678-5")
func (m *Module) FindPatient(tenantID, documentNumber string) (id, name, phone, email string, found bool, err error) {
	if tenantID == "" || documentNumber == "" || tenantID != m.tenantID {
		return "", "", "", "", false, nil
	}
	normRut, err := m.validateRUT(documentNumber)
	if err != nil {
		return "", "", "", "", false, nil
	}
	var foundPatient Patient
	var hasPatient bool
	err = m.store.ReadAll(func() model.Model { return &Patient{} }, crud.Query{Conditions: []storage.Condition{
		storage.Eq(Patient_.DocumentType, kinds.DocumentRUT),
		storage.Eq(Patient_.Rut, normRut),
	}}, func(r model.Model) {
		if !hasPatient {
			p := r.(*Patient)
			if p.DocumentCountry == "" {
				foundPatient = *p
				hasPatient = true
			}
		}
	})
	if err != nil {
		return "", "", "", "", false, err
	}
	if !hasPatient {
		return "", "", "", "", false, nil
	}
	return foundPatient.Id, foundPatient.Name, foundPatient.Phone, foundPatient.Email, true, nil
}

// CreatePatient crea un paciente activo con RUT y devuelve su id. Documento ya registrado ⇒ ErrRutAlreadyExists.
// Ejemplo: id, err := m.CreatePatient("t1", newID, "12.345.678-5", "Ana", "+56911223344", "ana@correo.cl")
func (m *Module) CreatePatient(tenantID, id, documentNumber, name, phone, email string) (string, error) {
	if tenantID != m.tenantID {
		return "", ErrIdTaken
	}
	if id == "" {
		id = m.store.NewID()
	}
	p := Patient{
		Id:           id,
		TenantId:     tenantID,
		DocumentType: kinds.DocumentRUT,
		Rut:          documentNumber,
		Name:         name,
		Phone:        phone,
		Email:        email,
		IsActive:     true,
	}
	err := m.store.Atomic(func(tx crud.Tx) error {
		if _, err := p.Create(crud.CurrentUser{}, tx, m); err != nil {
			return err
		}
		return tx.Create(&p)
	})
	if err != nil {
		return "", err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: crud.Topic(ModelName, (&Patient{}).ModelName(), model.Create), Payload: &crud.Change{Record: &p}})
	}
	return p.Id, nil
}
