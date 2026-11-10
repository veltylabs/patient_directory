//go:build browser

package module

import (
	"webtyp.com/appkit"
	"webtyp.com/view"

	"github.com/veltylabs/patient_directory/ui"
)

// En el navegador el módulo solo aporta sus pantallas.
var server appkit.ModuleServer

var browser = view.ModuleBrowser{Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) }}
