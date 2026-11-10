package patientdirectory

import (
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	titlePatients             = "Patients"
	searchPlaceholderPatients lang.Text = "Search a patient by RUT, name or phone"
)

// NewView construye el Presenter de pacientes —el motor agnóstico de renderizado
// que un renderizador (crudview o cualquier otro) envuelve. Este módulo lo
// construye (view + model + router únicamente); la aplicación decide qué
// renderizador lo dibuja.
//
// Ops NO lleva Delete: view.NewCallerLister implementa exactamente las
// capacidades de escritura gobernadas por PatientModel.Allow. Un paciente se desactiva,
// nunca se elimina (consulte AGENTS.md).
func NewView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			Page:   view.Page{First: 0, Size: 20, More: true},
		},
		func() model.TableSlice { return &PatientList{} })
	allOpts := append([]view.Option{
		view.WithTitle(titlePatients),
		view.WithSearchPlaceholder(searchPlaceholderPatients),
	}, opts...)
	return view.New(b, &Patient{}, allOpts...)
}
