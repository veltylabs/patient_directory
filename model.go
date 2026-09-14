package patientdirectory

import (
	"webtyp.com/fmt"
	"webtyp.com/input"
	"webtyp.com/model"
)

// PatientModel es el registro del establecimiento para las personas que atiende:
// IDENTIDAD y CONTACTO únicamente. Los datos clínicos —diagnósticos, atenciones, recetas—
// pertenecen a clinical_encounter y nunca deben agregarse aquí.
//
// rut NO lleva la marca Unique a propósito: la propiedad Unique de model.FieldDB es
// de columna única, y un RUT es único por TENANT, no globalmente. La restricción
// se aplica en CreatePatient contra ErrRutAlreadyExists —la misma regla de capa de
// aplicación que device_manager aplica a una IP de dispositivo.
var PatientModel = model.Definition{
	Name: "patient",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "rut", Type: input.Rut(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 12}},
		{Name: "name", Type: input.Text(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}},
		{Name: "birthdate", Type: input.Date(), OmitEmpty: true},
		{Name: "phone", Type: input.Phone(), OmitEmpty: true},
		{Name: "email", Type: input.Email(), OmitEmpty: true},
		{Name: "address", Type: input.Address(), OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), NotNull: true},
		{Name: "updated_at", Type: model.Int(), OmitEmpty: true},
	},
}

var ListPatientsArgsModel = model.Definition{
	Name: "list_patients_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

var GetPatientArgsModel = model.Definition{
	Name: "get_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var FindPatientByRutArgsModel = model.Definition{
	Name: "find_patient_by_rut_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "rut", Type: input.Rut()},
	},
}

var DeactivatePatientArgsModel = model.Definition{
	Name: "deactivate_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var (
	ErrNotFound         = fmt.Err("patient not found")
	ErrRutAlreadyExists = fmt.Err("patient rut already exists in this tenant")
	ErrRutRequired      = fmt.Err("patient rut is required")
	ErrNameRequired     = fmt.Err("patient name is required")
	ErrTenantRequired   = fmt.Err("patient tenant_id is required")
)

type ValidationError struct{ Err error }

func (v ValidationError) Error() string { return v.Err.Error() }

// <module>.<entity>.<past-tense-verb> — tenant_id va en la carga útil (payload),
// nunca en el nombre del tema (topic).
const (
	TopicPatientCreated     = "patient_directory.patient.created"
	TopicPatientUpdated     = "patient_directory.patient.updated"
	TopicPatientDeactivated = "patient_directory.patient.deactivated"
)
