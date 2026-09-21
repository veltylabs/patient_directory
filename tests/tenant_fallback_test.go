package tests

import (
	"testing"

	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// TestOpListPatients_FallsBackToModuleTenant reproduces a real bug found
// while manually testing a downstream app's patient list screen: it showed
// "0 / 0" even though a real patient existed for the app's tenant.
//
// Root cause: webtyp.com/view's callerLister.list() (the code every
// crudview-backed list runs on Reload()) always calls the List op with NO
// args — correct by design (router.Caller.Call's own doc: args may be nil).
// opListPatients therefore receives ListPatientsArgs{TenantId: ""} on every
// real page load, and Module.ListPatients (module.go) has an EXPLICIT early
// return for that case:
//
//	func (m *Module) ListPatients(tenantID string, filter PatientFilter) ([]Patient, error) {
//		if tenantID == "" {
//			return []Patient{}, nil
//		}
//		...
//
// silently returning nothing, no error, for every real page load.
//
// github.com/veltylabs/staff_manager already solves this exact class of bug:
// Deps carries a TenantID, New requires it, and its List op falls back to it
// when the caller's args are empty. patient_directory has no such field or
// fallback. This test seeds one patient under a configured module tenant and
// invokes OpListPatients with an EMPTY body — exactly what
// callerLister.list() sends — expecting that seeded row back.
//
// It does not compile today (patientdirectory.Deps has no TenantID field) —
// that is the intended red state; adding the field and the fallback is
// stage 1 of docs/PLAN.md.
func TestOpListPatients_FallsBackToModuleTenant(t *testing.T) {
	const moduleTenant = "mjosefa-cms"

	db := orm.New(mem.New())
	m, err := patientdirectory.New(db, patientdirectory.Deps{
		IDs:         &fakeIDGen{},
		ValidateRUT: fakeValidateRUT,
		TenantID:    moduleTenant,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	seeded, err := m.CreatePatient(patientdirectory.Patient{
		TenantId: moduleTenant,
		Rut:      "12345678-5",
		Name:     "Ana Torres",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreatePatient: %v", err)
	}

	reg := &mock.Router{}
	m.MountOperations(reg)
	reg.Configure(mock.Config{
		Authn:     func(next router.HandlerFunc) router.HandlerFunc { return next },
		Authorize: func(userID string, resource model.Resource, action model.Action) bool { return true },
	})

	// The exact wire shape view.NewCallerLister sends: no args at all.
	ctx := &mock.Context{InBody: []byte(`{}`)}
	ctx.SetUserID("test-user")
	reg.Invoke("OP", "/"+patientdirectory.OpListPatients, ctx)

	if ctx.Status != 0 && ctx.Status != 200 {
		t.Fatalf("list_patients with empty args: status = %d, body=%s", ctx.Status, ctx.ResponseBody())
	}
	body := string(ctx.ResponseBody())
	if !containsSubstr(body, seeded.Id) {
		t.Fatalf("list_patients with empty args did not return the seeded patient (id %q) for the "+
			"module's own tenant %q — got: %s — this reproduces the \"0 / 0\" bug the real UI shows "+
			"on every page load, since it never sends a tenant_id", seeded.Id, moduleTenant, body)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
