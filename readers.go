package patientdirectory

import (
	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/model"
)

// GetPatient lee un paciente por su id. Ausente u otro tenant ⇒ found=false, err=nil.
// Ejemplo: name, doc, phone, email, ok, err := m.GetPatient("t1", "p1")
func (m *Module) GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error) {
	if tenantID == "" || id == "" {
		return "", "", "", "", false, nil
	}
	p, err := m.getPatient(tenantID, id)
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
	p, err := m.findByDocument(tenantID, documentNumber)
	if model.KindOf(err) == model.ErrorNotFound {
		return "", "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", "", false, err
	}
	return p.Id, p.Name, p.Phone, p.Email, true, nil
}

// CreatePatient crea un paciente activo con RUT y devuelve su id. Documento ya registrado ⇒ ErrRutAlreadyExists.
// Ejemplo: id, err := m.CreatePatient("t1", newID, "12.345.678-5", "Ana", "+56911223344", "ana@correo.cl")
func (m *Module) CreatePatient(tenantID, id, documentNumber, name, phone, email string) (string, error) {
	p, err := m.createPatient(Patient{
		Id: id, TenantId: tenantID, DocumentType: kinds.DocumentRUT, Rut: documentNumber,
		Name: name, Phone: phone, Email: email, IsActive: true,
	})
	if err != nil {
		return "", err
	}
	return p.Id, nil
}
