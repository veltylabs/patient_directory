package patientdirectory

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/rpc"
	"webtyp.com/view"
)

const (
	OpListPatients      = "list_patients"
	OpGetPatient        = "get_patient"
	OpFindPatientByRut  = "find_patient_by_rut"
	OpUpsertPatient     = "upsert_patient"
	OpDeactivatePatient = "deactivate_patient"
)

// ModelName is this module's identity: mcp.HarvestOps qualifies every op
// below as "patient_directory.<name>" on the wire, and view.go's NewView
// passes this same constant as view.Ops.Module so the client composes the
// identical qualified name.
const ModelName = "patient_directory"

func (m *Module) ModelName() string { return ModelName }

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListPatients, m.opListPatients).Requires(model.ResourceOf(&PatientModel), model.Read).Accepts(&view.ListArgs{})
	reg.Operation(OpGetPatient, m.opGetPatient).Requires(model.ResourceOf(&PatientModel), model.Read).Accepts(&GetPatientArgs{})
	reg.Operation(OpFindPatientByRut, m.opFindPatientByRut).Requires(model.ResourceOf(&PatientModel), model.Read).Accepts(&FindPatientByRutArgs{})
	view.MountSave(reg, OpUpsertPatient, func(p *Patient) error {
		if p.TenantId == "" {
			p.TenantId = m.tenantID
		}
		if p.Id == "" {
			return ErrIdRequired
		}
		_, getErr := m.getPatient(p.TenantId, p.Id)
		var out Patient
		var err error
		if getErr != nil {
			if model.KindOf(getErr) == model.ErrorNotFound {
				out, err = m.createPatient(*p)
			} else {
				return getErr
			}
		} else {
			out, err = m.updatePatient(*p)
		}
		if err == nil {
			*p = out
		}
		return err
	}).Requires(model.ResourceOf(&PatientModel), model.Create|model.Update)
	reg.Operation(OpDeactivatePatient, m.opDeactivatePatient).Requires(model.ResourceOf(&PatientModel), model.Update).Accepts(&DeactivatePatientArgs{})
}

var _ router.OperationModule = (*Module)(nil)

func (m *Module) opListPatients(ctx router.Context) {
	var args view.ListArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	patients, err := m.listPatients(m.tenantID, args.Search, args.After, view.Cap(args.Limit))
	if err != nil {
		rpc.WriteError(ctx, err)
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
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	p, err := m.getPatient(args.TenantId, args.Id)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	_ = ctx.Encode(&p)
}

func (m *Module) opFindPatientByRut(ctx router.Context) {
	var args FindPatientByRutArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	p, err := m.findByDocument(args.TenantId, args.Rut)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	_ = ctx.Encode(&p)
}

func (m *Module) opDeactivatePatient(ctx router.Context) {
	var args DeactivatePatientArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	err := m.deactivatePatient(args.TenantId, args.Id)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}
