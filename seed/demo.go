//go:build wasm

package seed

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/migrate"
	"github.com/veltylabs/patient_directory/ui"
	"webtyp.com/crud"
	"webtyp.com/ddl"
	"webtyp.com/input"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
	"webtyp.com/view"
)

// Demo crea una demostración ejecutable del módulo patient_directory sobre mem db con datos semilla.
func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	ids, err := unixid.NewUnixID()
	if err != nil {
		return view.Demo{}, err
	}
	tenantID := "demo-tenant"
	mod, err := patientdirectory.New(db, patientdirectory.Deps{
		TenantID:    tenantID,
		IDs:         ids,
		ValidateRUT: input.NormalizeRUT,
	})
	if err != nil {
		return view.Demo{}, err
	}
	if _, err := Load(mod, tenantID); err != nil {
		return view.Demo{}, err
	}
	return view.Demo{
		Label:    ui.Label,
		Timezone: "America/Santiago",
		ActingAs: "u1",
		Modules:  []crud.Module{mod},
		Migrations: []ddl.Step{
			{Name: patientdirectory.ModelName, Run: migrate.Migrate},
		},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller)
		},
	}, nil
}
