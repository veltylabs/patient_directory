package tests

import (
	"encoding/json"
	"testing"

	"webtyp.com/model"
	"webtyp.com/router"

	patientdirectory "github.com/veltylabs/patient_directory"
)

type mockRoute struct{}

func (m *mockRoute) Requires(resource model.Resource, action model.Action) router.Route {
	return m
}
func (m *mockRoute) RequiresMFA() router.Route { return m }
func (m *mockRoute) Authenticated() router.Route { return m }
func (m *mockRoute) Public() router.Route        { return m }
func (m *mockRoute) Accepts(args model.Fielder) router.Route {
	return m
}
func (m *mockRoute) Describe(desc string) router.Route { return m }

type mockRegistry struct {
	ops map[string]router.HandlerFunc
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{ops: make(map[string]router.HandlerFunc)}
}

func (r *mockRegistry) Operation(name string, h router.HandlerFunc) router.Route {
	r.ops[name] = h
	return &mockRoute{}
}

type mockContext struct {
	body       []byte
	statusCode int
	respBody   []byte
}

func (c *mockContext) Method() string                   { return "POST" }
func (c *mockContext) Path() string                     { return "/" }
func (c *mockContext) Body() []byte                     { return c.body }
func (c *mockContext) GetHeader(key string) string      { return "" }
func (c *mockContext) SetHeader(key, value string)      {}
func (c *mockContext) WriteStatus(code int)             { c.statusCode = code }
func (c *mockContext) Write(b []byte) (int, error)      { c.respBody = append(c.respBody, b...); return len(b), nil }
func (c *mockContext) SetValue(key, value string)       {}
func (c *mockContext) Value(key string) string          { return "" }
func (c *mockContext) Param(name string) string         { return "" }
func (c *mockContext) SetCookie(cookie router.Cookie)   {}
func (c *mockContext) Cookie(name string) (router.Cookie, bool) { return router.Cookie{}, false }
func (c *mockContext) SetUserID(id string)              {}
func (c *mockContext) UserID() string                   { return "" }
func (c *mockContext) Assurance() model.Assurance       { return 0 }
func (c *mockContext) SetAssurance(a model.Assurance)   {}

func (c *mockContext) Decode(into model.Decodable) error {
	var m map[string]any
	if err := json.Unmarshal(c.body, &m); err != nil {
		return err
	}
	r := &mapReader{m: m}
	into.DecodeFields(r)
	return nil
}

func (c *mockContext) Encode(v model.Encodable) error {
	w := &mapWriter{m: make(map[string]any)}
	v.EncodeFields(w)
	b, err := json.Marshal(w.m)
	if err != nil {
		return err
	}
	c.respBody = b
	return nil
}

type mapReader struct {
	m map[string]any
}

func (r *mapReader) String(name string) (string, bool) {
	v, ok := r.m[name].(string)
	return v, ok
}
func (r *mapReader) Int(name string) (int64, bool) {
	v, ok := r.m[name].(float64)
	return int64(v), ok
}
func (r *mapReader) Float(name string) (float64, bool) {
	v, ok := r.m[name].(float64)
	return v, ok
}
func (r *mapReader) Bool(name string) (bool, bool) {
	v, ok := r.m[name].(bool)
	return v, ok
}
func (r *mapReader) Bytes(name string) ([]byte, bool) {
	v, ok := r.m[name].(string)
	return []byte(v), ok
}
func (r *mapReader) Object(name string, into model.Decodable) bool {
	return false
}
func (r *mapReader) Array(name string) (model.ArrayReader, bool) {
	return nil, false
}
func (r *mapReader) Raw(name string) (string, bool) {
	return "", false
}

type mapWriter struct {
	m map[string]any
}

func (w *mapWriter) String(name, val string) { w.m[name] = val }
func (w *mapWriter) Int(name string, val int64) { w.m[name] = val }
func (w *mapWriter) Float(name string, val float64) { w.m[name] = val }
func (w *mapWriter) Bool(name string, val bool) { w.m[name] = val }
func (w *mapWriter) Bytes(name string, val []byte) { w.m[name] = string(val) }
func (w *mapWriter) Null(name string) { w.m[name] = nil }
func (w *mapWriter) Raw(name, val string) { w.m[name] = val }
func (w *mapWriter) Object(name string, val model.Encodable) {}
func (w *mapWriter) Array(name string, n int) model.ArrayWriter { return nil }

func TestOpsRegistrationAndExecution(t *testing.T) {
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}

	reg := newMockRegistry()
	m.MountOperations(reg)


	expectedOps := []string{
		patientdirectory.OpListPatients,
		patientdirectory.OpGetPatient,
		patientdirectory.OpFindPatientByRut,
		patientdirectory.OpUpsertPatient,
		patientdirectory.OpDeactivatePatient,
	}
	for _, op := range expectedOps {
		if _, exists := reg.ops[op]; !exists {
			t.Errorf("se esperaba que la op %s estuviera registrada", op)
		}
	}


	// Probar 400 en request decode (JSON body)
	findHandler := reg.ops[patientdirectory.OpFindPatientByRut]
	ctxBad := &mockContext{body: []byte(`{bad json}`)}
	findHandler(ctxBad)
	if ctxBad.statusCode != 400 {
		t.Errorf("expected 400 on bad json request, got %d", ctxBad.statusCode)
	}

	// Probar 404 en GetPatient
	getHandler := reg.ops[patientdirectory.OpGetPatient]
	getReq, _ := json.Marshal(map[string]any{
		"tenant_id": "tenant-1",
		"id":        "non-existent-id",
	})
	ctxGet := &mockContext{body: getReq}
	getHandler(ctxGet)
	if ctxGet.statusCode != 404 {
		t.Errorf("expected 404 on missing patient, got %d", ctxGet.statusCode)
	}


	// Probar 409 en conflicto
}