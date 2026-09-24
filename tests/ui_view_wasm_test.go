//go:build wasm

package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/ui"
	"webtyp.com/dom"
	"webtyp.com/json"
	"webtyp.com/model"
)

const tenantID = "demo"

// initView monta la vista del módulo como lo hace el chasis: Init una vez y
// luego Render, y devuelve el HTML resultante.
func initView(m interface{ View() dom.Component }) string {
	comp := m.View()
	if initer, ok := comp.(interface{ Init(ctx dom.Ctx) }); ok {
		initer.Init(nil)
	}
	if r, ok := comp.(dom.ViewRenderer); ok {
		return r.Render().String()
	}
	return comp.String()
}

// called reporta si alguna de las ops pedidas fue invocada.
func called(ops []string, want string) bool {
	for _, op := range ops {
		if op == want {
			return true
		}
	}
	return false
}

func TestWASM_PatientDirectory_ViewListsOnInit(t *testing.T) {
	var ops []string
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			ops = append(ops, op)
			if op == patientdirectory.ModelName+"."+patientdirectory.OpListPatients {
				list := patientdirectory.PatientList{
					{Id: "p1", TenantId: tenantID, Rut: "11111111-1", Name: "Paciente Uno", IsActive: true},
				}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			}
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, tenantID)
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}
	initView(m)

	if !called(ops, patientdirectory.ModelName+"."+patientdirectory.OpListPatients) {
		t.Errorf("la pantalla de pacientes debe pedir su lista al iniciarse, ops=%v", ops)
	}
}
