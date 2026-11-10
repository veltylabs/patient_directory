//go:build !browser

package module

import (
	"webtyp.com/appkit"
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/view"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// server es lo que el módulo aporta al servidor: cómo se construye.
var server = appkit.ModuleServer{Build: build}

// En el servidor el módulo no aporta pantallas.
var browser view.ModuleBrowser

func build(d appkit.Deps) (model.ModuleNaming, error) {
	return patientdirectory.New(d.DB, patientdirectory.Deps{
		Publisher:   d.Events,
		TenantID:    d.TenantID,
		IDs:         d.IDs,
		ValidateRUT: input.NormalizeRUT,
	})
}
