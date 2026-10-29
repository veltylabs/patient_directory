package patientdirectory

import (
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
		{Name: "rut", Type: input.Rut(), Label: "Rut", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 12}, Item: model.ItemDescription},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}, Item: model.ItemLabel},
		{Name: "birthdate", Type: input.Date(), Label: "Birthdate", OmitEmpty: true},
		{Name: "phone", Type: input.Phone(), Label: "Phone", OmitEmpty: true},
		{Name: "email", Type: input.Email(), Label: "Email", OmitEmpty: true},
		{Name: "address", Type: input.Address(), Label: "Address", OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active", NotNull: true},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

var LinkPatientArgsModel = model.Definition{
	Name: "link_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, NotNull: true},
		{Name: "rut", Type: input.Rut(), Label: "Rut", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 12}, Item: model.ItemDescription},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}, Item: model.ItemLabel},
		{Name: "phone", Type: input.Phone(), Label: "Phone", NotNull: true},
		{Name: "email", Type: input.Email(), Label: "Email", NotNull: true},
	},
}

var LinkPatientResultModel = model.Definition{
	Name: "link_patient_result",
	Fields: model.Fields{
		{Name: "id", Type: model.Text()},
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
		{Name: "rut", Type: input.Rut(), Label: "Rut"},
	},
}

var DeactivatePatientArgsModel = model.Definition{
	Name: "deactivate_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

const (
	ErrNotFound         model.NotFound = "patient not found"
	ErrCannotRead       model.Invalid  = "patient_directory: cannot read the request"
	ErrRutAlreadyExists model.Conflict = "patient rut already exists in this tenant"
	ErrRutRequired      model.Invalid  = "patient rut is required"
	ErrNameRequired     model.Invalid  = "patient name is required"
	ErrTenantRequired   model.Invalid  = "patient tenant_id is required"
	ErrIdRequired       model.Invalid  = "patient id is required"
	ErrIdTaken          model.Conflict = "patient id belongs to another tenant"
)

// <module>.<entity>.<past-tense-verb> — tenant_id va en la carga útil (payload),
// nunca en el nombre del tema (topic).
const (
	TopicPatientCreated     = "patient_directory.patient.created"
	TopicPatientUpdated     = "patient_directory.patient.updated"
	TopicPatientDeactivated = "patient_directory.patient.deactivated"
)

var ResourcePatientLink = model.FeatureResource("patient_link")
