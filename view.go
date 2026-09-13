package patientdirectory

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// Item implementa view.Itemizer. Description es el RUT porque es lo que una
// persona en el mostrador sostiene y lee de vuelta para confirmar que seleccionó
// al paciente correcto.
func (p *Patient) Item() view.Item {
	return view.Item{ID: p.Id, Label: p.Name, Description: p.Rut}
}

const titlePatients = "Patients"

// NewView construye el Presenter de pacientes —el motor agnóstico de renderizado
// que un renderizador (crudview o cualquier otro) envuelve. Este módulo lo
// construye (view + model + router únicamente); la aplicación decide qué
// renderizador lo dibuja.
//
// Ops NO lleva Delete: view.NewCallerLister implementa exactamente las
// capacidades de escritura cuyos nombres de op no están vacíos, por lo que un
// renderizador no dibuja controles de eliminación. Un paciente se desactiva,
// nunca se elimina (consulte AGENTS.md).
func NewView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{List: OpListPatients, Save: OpUpsertPatient},
		func() model.ModelSlice { return &PatientList{} })
	return view.New(b, &Patient{}, view.WithTitle(titlePatients))
}
