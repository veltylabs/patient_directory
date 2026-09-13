package patientdirectory

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// Item implements view.Itemizer. Description is the RUT because that is what a
// person at the counter is holding and reads back to confirm they picked the
// right patient.
func (p *Patient) Item() view.Item {
	return view.Item{ID: p.Id, Label: p.Name, Description: p.Rut}
}

const titlePatients = "Patients"

// NewView builds the patient Presenter — the renderer-agnostic engine a
// renderer (crudview, or any other) wraps. This module builds it (view + model
// + router only); the app decides which renderer draws it.
//
// Ops carries NO Delete: view.NewCallerLister implements exactly the write
// capabilities whose op names are non-empty, so a renderer paints no delete
// control. A patient is deactivated, never deleted (see AGENTS.md).
func NewView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{List: OpListPatients, Save: OpUpsertPatient},
		func() model.ModelSlice { return &PatientList{} })
	return view.New(b, &Patient{}, view.WithTitle(titlePatients))
}
