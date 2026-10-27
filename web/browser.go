//go:build browser

package main

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/seed"
	"github.com/veltylabs/patient_directory/ui"
	"webtyp.com/auth/trusted_ip"
	. "webtyp.com/dom"
	"webtyp.com/events/mock"
	"webtyp.com/html"
	"webtyp.com/orm"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

// demoTenantID es el único tenant de esta demo ejecutable en navegador.
const demoTenantID = "demo"

func main() {
	_, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}
	db := orm.New(mem.New())
	broker := &mock.Broker{}

	mod, err := patientdirectory.New(db, patientdirectory.Deps{
		Publisher:   broker,
		TenantID:    demoTenantID,
		ValidateRUT: trustedip.ValidateRUT,
	})
	if err != nil {
		panic(err)
	}

	if _, err := seed.Load(mod, demoTenantID); err != nil {
		panic(err)
	}

	caller := loopback.WithTenant(demoTenantID, mod)
	screens := ui.Screens(caller)

	appNode := html.Div().Child(
		html.H1().Text(string(screens[0].Label)),
		html.P().Text("Demo render sin webtyp.com/layout"),
	)

	Append("body", appNode)
	select {}
}
