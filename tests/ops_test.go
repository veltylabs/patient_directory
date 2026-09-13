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
func (m *mockRoute) Authenticated() router.Route { return m }
func (m *mockRoute) Public() router.Route        { return m }
func (m *mockRoute) Accepts(args model.Fielder) router.Route {
	return m
}

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
	m, _, _, err := setupModule()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	reg := newMockRegistry()
	m.MountOperations(reg)

	// Ensure delete_patient is NOT registered
	if _, exists := reg.ops["delete_patient"]; exists {
		t.Errorf("delete_patient op should NOT exist")
	}

	expectedOps := []string{
		patientdirectory.OpListPatients,
		patientdirectory.OpGetPatient,
		patientdirectory.OpFindPatientByRut,
		patientdirectory.OpUpsertPatient,
		patientdirectory.OpDeactivatePatient,
	}
	for _, op := range expectedOps {
		if _, exists := reg.ops[op]; !exists {
			t.Errorf("expected op %s to be mounted", op)
		}
	}

	// Test OpUpsertPatient creation success
	upsertHandler := reg.ops[patientdirectory.OpUpsertPatient]
	reqBody, _ := json.Marshal(map[string]any{
		"tenant_id": "tenant-1",
		"rut":       "12345678-K",
		"name":      "John Doe",
		"is_active": true,
	})
	ctx := &mockContext{body: reqBody}
	upsertHandler(ctx)

	if ctx.statusCode != 0 && ctx.statusCode != 200 {
		t.Errorf("expected status 200 or 0, got %d", ctx.statusCode)
	}

	// Test Duplicate RUT error -> 409
	ctxDup := &mockContext{body: reqBody}
	upsertHandler(ctxDup)
	if ctxDup.statusCode != 409 {
		t.Errorf("expected status 409 on duplicate RUT, got %d", ctxDup.statusCode)
	}

	// Test GetPatient not found -> 404
	getHandler := reg.ops[patientdirectory.OpGetPatient]
	getReq, _ := json.Marshal(map[string]any{
		"tenant_id": "tenant-1",
		"id":        "non-existent-id",
	})
	ctxGet := &mockContext{body: getReq}
	getHandler(ctxGet)
	if ctxGet.statusCode != 404 {
		t.Errorf("expected status 404 on non-existent patient, got %d", ctxGet.statusCode)
	}
}
