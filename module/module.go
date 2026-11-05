// Package module es patient_directory tal como lo enchufa una app: un solo valor con su
// parte de servidor (server.go) y sus pantallas (browser.go). Va en una
// subcarpeta porque importa la raíz, ui/ y migrate/, que a su vez importan la raíz.
package module

import "webtyp.com/appkit"

// Module es patient_directory para appkit: va una sola vez en App.Modules de la app.
//
// Ejemplo: Modules: []appkit.Module{patientdirectory.Module}
var Module = appkit.Module{Name: "patient_directory", Server: server, Browser: browser}
