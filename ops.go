package patientdirectory

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

const (
	OpListPatients      = "list_patients"
	OpGetPatient        = "get_patient"
	OpFindPatientByRut  = "find_patient_by_rut"
	OpUpsertPatient     = "upsert_patient"
	OpDeactivatePatient = "deactivate_patient"
)

func (m *Module) ModelName() string { return "patient_directory" }

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListPatients, m.opListPatients).Requires("patient", model.Read).Accepts(&ListPatientsArgs{})
	reg.Operation(OpGetPatient, m.opGetPatient).Requires("patient", model.Read).Accepts(&GetPatientArgs{})
	reg.Operation(OpFindPatientByRut, m.opFindPatientByRut).Requires("patient", model.Read).Accepts(&FindPatientByRutArgs{})
	reg.Operation(OpUpsertPatient, m.opUpsertPatient).Requires("patient", model.Create|model.Update).Accepts(&Patient{})
	reg.Operation(OpDeactivatePatient, m.opDeactivatePatient).Requires("patient", model.Update).Accepts(&DeactivatePatientArgs{})
}

var _ router.OperationModule = (*Module)(nil)

func (m *Module) handleErr(ctx router.Context, err error) {
	if err == nil {
		return
	}
	if _, ok := err.(ValidationError); ok {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	if err == ErrRutAlreadyExists {
		ctx.WriteStatus(409)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	if err == ErrNotFound {
		ctx.WriteStatus(404)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	ctx.WriteStatus(500)
	_, _ = ctx.Write([]byte(err.Error()))
}

func (m *Module) opListPatients(ctx router.Context) {
	var args ListPatientsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	patients, err := m.ListPatients(args.TenantId, PatientFilter{
		ActiveOnly: args.ActiveOnly,
		Limit:      args.Limit,
		Offset:     args.Offset,
	})
	if err != nil {
		m.handleErr(ctx, err)
		return
	}
	var list PatientList
	for i := range patients {
		list = append(list, &patients[i])
	}
	_ = ctx.Encode(&list)
}

func (m *Module) opGetPatient(ctx router.Context) {
	var args GetPatientArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	p, err := m.GetPatient(args.TenantId, args.Id)
	if err != nil {
		m.handleErr(ctx, err)
		return
	}
	_ = ctx.Encode(&p)
}

func (m *Module) opFindPatientByRut(ctx router.Context) {
	var args FindPatientByRutArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	p, err := m.FindByRut(args.TenantId, args.Rut)
	if err != nil {
		m.handleErr(ctx, err)
		return
	}
	_ = ctx.Encode(&p)
}

func (m *Module) opUpsertPatient(ctx router.Context) {
	var p Patient
	if err := ctx.Decode(&p); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	var res Patient
	var err error
	if p.Id == "" {
		res, err = m.CreatePatient(p)
	} else {
		res, err = m.UpdatePatient(p)
	}
	if err != nil {
		m.handleErr(ctx, err)
		return
	}
	_ = ctx.Encode(&res)
}

func (m *Module) opDeactivatePatient(ctx router.Context) {
	var args DeactivatePatientArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	err := m.DeactivatePatient(args.TenantId, args.Id)
	if err != nil {
		m.handleErr(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}
