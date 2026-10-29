package ui

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const descPatients lang.Text = "Patient directory: personal data, contact information and identification."

// Screens declara las pantallas que este módulo ofrece al menú de la app.
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{
		{
			ID:         ID,
			Label:      Label,
			Icon:       ID,
			Permission: model.ResourceOf(&patientdirectory.PatientModel),
			Presenter:  patientdirectory.NewView(caller, view.WithDescription(descPatients)),
		},
	}
}
