//go:build !browser

package module

import (
	"webtyp.com/appkit"
	trustedip "webtyp.com/auth/trusted_ip"
	"webtyp.com/model"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// server es lo que el módulo aporta al servidor: cómo se construye.
var server = appkit.ModuleServer{Build: build}

// En el servidor el módulo no aporta pantallas.
var browser appkit.ModuleBrowser

func build(d appkit.Deps) (model.ModuleNaming, error) {
	return patientdirectory.New(d.DB, patientdirectory.Deps{
		Publisher: d.Events, TenantID: d.TenantID, ValidateRUT: trustedip.ValidateRUT,
	})
}
