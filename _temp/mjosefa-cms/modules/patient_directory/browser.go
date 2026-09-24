package patient_directory

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/svg"
)

// Browser builds this module's view for the registry in modules/browser.go.
// tenantID is unused — see server.go's comment on the same parameter.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID,
		Presenter: patientdirectory.NewView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	return platformd.NewUIModule(ID, Label, svg.Icon(ID), v), nil
}
