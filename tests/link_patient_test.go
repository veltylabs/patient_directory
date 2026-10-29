package tests

import (
	"testing"

	"github.com/veltylabs/patient_directory"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"
)

func TestLinkPatient_MissingFields(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	cases := []struct {
		name string
		args patientdirectory.LinkPatientArgs
		err  error
	}{
		{"no tenant", patientdirectory.LinkPatientArgs{Id: "i1", Rut: "1-9", Name: "N", Phone: "1", Email: "e@e.c"}, patientdirectory.ErrTenantRequired},
		{"no id", patientdirectory.LinkPatientArgs{TenantId: "t1", Rut: "1-9", Name: "N", Phone: "1", Email: "e@e.c"}, patientdirectory.ErrIdRequired},
		{"no rut", patientdirectory.LinkPatientArgs{TenantId: "t1", Id: "i1", Name: "N", Phone: "1", Email: "e@e.c"}, patientdirectory.ErrRutRequired},
		{"no name", patientdirectory.LinkPatientArgs{TenantId: "t1", Id: "i1", Rut: "1-9", Phone: "1", Email: "e@e.c"}, patientdirectory.ErrNameRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.LinkPatient(c.args)
			if err == nil || err.Error() != c.err.Error() {
				t.Fatalf("expected error %v, got %v", c.err, err)
			}
		})
	}
}

func TestLinkPatient_ExistingRutReturnsStoredIdUntouched(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return "1-9", nil }, // normalizes
		TenantID:    "t1",
	})
	_, err := m.CreatePatient(patientdirectory.Patient{
		Id:       "stored-id",
		TenantId: "t1",
		Rut:      "1-9",
		Name:     "Old Name",
		Phone:    "Old Phone",
		Email:    "old@email.com",
		IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	id, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1.9", // non-normalized
		Name:     "New Name",
		Phone:    "New Phone",
		Email:    "new@email.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "stored-id" {
		t.Fatalf("expected stored-id, got %v", id)
	}

	// verify untouched
	p, _ := m.GetPatient("t1", "stored-id")
	if p.Name != "Old Name" || p.Phone != "Old Phone" || p.Email != "old@email.com" {
		t.Fatalf("patient modified")
	}
}

func TestLinkPatient_InactivePatientStillLinks(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	_, _ = m.CreatePatient(patientdirectory.Patient{
		Id:       "stored-id",
		TenantId: "t1",
		Rut:      "1-9",
		Name:     "Old Name",
		IsActive: true,
	})
	_ = m.DeactivatePatient("t1", "stored-id")

	id, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "New Name",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "stored-id" {
		t.Fatalf("expected stored-id, got %v", id)
	}
}

func TestLinkPatient_NewRutCreatesWithCallerId(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	id, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "Name",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "caller-id" {
		t.Fatalf("expected caller-id, got %v", id)
	}
	p, _ := m.GetPatient("t1", "caller-id")
	if p.Rut != "1-9" || p.Name != "Name" {
		t.Fatalf("patient not created correctly")
	}
}

type brokenConn struct{ storage.Conn }
type brokenScanner struct{}

func (brokenScanner) Scan(dest ...any) error                             { return fmt.Err("error simulado") }
func (c *brokenConn) QueryRow(query string, args ...any) storage.Scanner { return brokenScanner{} }
func (c *brokenConn) Query(query string, args ...any) (storage.Rows, error) {
	return nil, fmt.Err("error simulado")
}

func TestLinkPatient_StorageErrorPropagates(t *testing.T) {
	db := orm.New(&brokenConn{Conn: mem.New()})
	m, _ := patientdirectory.New(db, patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	_, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "Name",
		Phone:    "123",
		Email:    "e@e.c",
	})
	if err == nil || err.Error() != "error simulado" {
		t.Fatalf("expected error simulado, got %v", err)
	}
}

type lieOnceConn struct {
	storage.Conn
	lied bool
}
type noRowsScanner struct{}

func (noRowsScanner) Scan(dest ...any) error { return storage.ErrNoRows }

func (c *lieOnceConn) QueryRow(q string, args ...any) storage.Scanner {
	if !c.lied {
		c.lied = true
		return noRowsScanner{}
	}
	return c.Conn.QueryRow(q, args...)
}

func TestLinkPatient_RaceOnCreateReturnsExistingId(t *testing.T) {
	conn := mem.New()
	m1, err1 := patientdirectory.New(orm.New(conn), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	if err1 != nil {
		t.Fatal(err1)
	}

	_, err := m1.CreatePatient(patientdirectory.Patient{
		Id:       "existente",
		TenantId: "t1",
		Rut:      "1-9",
		Name:     "Name",
		IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	m2, err2 := patientdirectory.New(orm.New(&lieOnceConn{Conn: conn}), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	if err2 != nil {
		t.Fatal(err2)
	}

	id, err := m2.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "nuevo",
		Rut:      "1-9",
		Name:     "Name2",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "existente" {
		t.Fatalf("expected existente id, got %v", id)
	}

	patients, _ := m2.ListPatients("t1", patientdirectory.PatientFilter{})
	count := 0
	for _, p := range patients {
		if p.Rut == "1-9" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 patient with RUT 1-9, got %d", count)
	}
}

func TestLinkPatient_IdTakenByOtherTenant(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	_, _ = m.CreatePatient(patientdirectory.Patient{
		Id:       "caller-id",
		TenantId: "t2",
		Rut:      "2-7",
		Name:     "Name",
		IsActive: true,
	})
	_, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "Name",
	})
	if model.KindOf(err) != model.ErrorConflict {
		t.Fatalf("expected ErrIdTaken, got %v", err)
	}
}

func TestLinkPatient_InvalidRutIsRejected(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) {
			if s == "bad" {
				return "", &model.FieldError{Field: "rut", Err: fmt.Err("invalid rut")}
			}
			return s, nil
		},
		TenantID: "t1",
	})
	_, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "bad",
		Name:     "Name",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLinkPatient_RetryIsIdempotent(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	id1, err1 := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "Name",
	})
	id2, err2 := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t1",
		Id:       "caller-id",
		Rut:      "1-9",
		Name:     "Name",
	})
	if err1 != nil || err2 != nil || id1 != "caller-id" || id2 != "caller-id" {
		t.Fatalf("retry failed: %v %v %v %v", err1, err2, id1, id2)
	}
}

func TestLinkPatient_TenantIsolation(t *testing.T) {
	m, _ := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		ValidateRUT: func(s string) (string, error) { return s, nil },
		TenantID:    "t1",
	})
	_, _ = m.CreatePatient(patientdirectory.Patient{
		Id:       "id1",
		TenantId: "t1",
		Rut:      "1-9",
		Name:     "Name1",
		IsActive: true,
	})
	id, err := m.LinkPatient(patientdirectory.LinkPatientArgs{
		TenantId: "t2",
		Id:       "id2",
		Rut:      "1-9", // same rut
		Name:     "Name2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "id2" {
		t.Fatalf("expected id2, got %v", id)
	}
}

func TestOpLinkPatient_ResponseCarriesOnlyId(t *testing.T) {
	m, _, _ := setupModule()
	r := &mock.Router{}
	r.Configure(mock.Config{
		Authorize: func(userID string, resource model.Resource, action model.Action) bool {
			return true // mock allow
		},
	})
	m.MountOperations(r)

	ctx := &mock.Context{
		InBody: []byte(`{"tenant_id":"t1","id":"caller-id","rut":"1-9","name":"Name","phone":"+123456789","email":"e@e.c"}`),
	}
	ctx.SetUserID("user1")

	r.Invoke("OP", "/"+patientdirectory.OpLinkPatient, ctx)

	if ctx.Status != 0 && ctx.Status != 200 {
		t.Fatalf("expected 200, got %v, body: %s", ctx.Status, string(ctx.ResponseBody()))
	}

	if string(ctx.ResponseBody()) != `{"id":"caller-id"}` {
		t.Fatalf("expected only id: caller-id, got %v", string(ctx.ResponseBody()))
	}
}

func TestOpLinkPatient_BadBodyAndInvalidArgsAre400(t *testing.T) {
	m, _, _ := setupModule()
	r := &mock.Router{}
	r.Configure(mock.Config{
		Authorize: func(userID string, resource model.Resource, action model.Action) bool {
			return true // mock allow
		},
	})
	m.MountOperations(r)

	path := "/" + patientdirectory.OpLinkPatient

	ctx := &mock.Context{
		InBody: []byte("{"),
	}
	ctx.SetUserID("user1")
	r.Invoke("OP", path, ctx)
	if ctx.Status != 400 {
		t.Fatalf("expected 400, got %v", ctx.Status)
	}

	ctx2 := &mock.Context{
		InBody: []byte(`{"tenant_id":"t1","id":"caller-id","rut":"1-9","phone":"+123456789","email":"e@e.c"}`),
	}
	ctx2.SetUserID("user1")
	r.Invoke("OP", path, ctx2)
	if ctx2.Status != 400 {
		t.Fatalf("expected 400, got %v", ctx2.Status)
	}
}

func TestOpLinkPatient_LinkPatientErrorPropagates(t *testing.T) {
	m, _, _ := setupModule()
	r := &mock.Router{}
	r.Configure(mock.Config{
		Authorize: func(userID string, resource model.Resource, action model.Action) bool {
			return true // mock allow
		},
	})
	m.MountOperations(r)

	_, _ = m.CreatePatient(patientdirectory.Patient{
		Id:       "caller-id",
		TenantId: "t2",
		Rut:      "2-7",
		Name:     "Name",
		IsActive: true,
	})

	path := "/" + patientdirectory.OpLinkPatient
	ctx := &mock.Context{
		InBody: []byte(`{"tenant_id":"t1","id":"caller-id","rut":"1-9","name":"Name","phone":"+123456789","email":"e@e.c"}`),
	}
	ctx.SetUserID("user1")
	r.Invoke("OP", path, ctx)
	if ctx.Status != 409 {
		t.Fatalf("expected 409, got %v", ctx.Status)
	}
}

func TestOpLinkPatient_RequiresPatientLinkCreate(t *testing.T) {
	m, _, _ := setupModule()
	r := &mock.Router{}
	m.MountOperations(r)

	for _, info := range r.Routes() {
		// Mock routes operation names to paths, so we check path
		if info.Path == "/"+patientdirectory.OpLinkPatient {
			if info.Resource != patientdirectory.ResourcePatientLink {
				t.Fatalf("expected resource %v, got %v", patientdirectory.ResourcePatientLink, info.Resource)
			}
			if info.Action != model.Create {
				t.Fatalf("expected action %v, got %v", model.Create, info.Action)
			}
			return
		}
	}
	t.Fatal("route not found")
}

func TestLinkPatientModels_RoundTripAndValidate(t *testing.T) {
	args := patientdirectory.LinkPatientArgs{TenantId: "t", Id: "i", Rut: "1", Name: "n"}
	if args.ModelName() != "link_patient_args" {
		t.Fatal("wrong model name")
	}
	res := patientdirectory.LinkPatientResult{Id: "i"}
	if res.ModelName() != "link_patient_result" {
		t.Fatal("wrong model name")
	}
}
