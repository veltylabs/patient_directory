package tests

import (
	"strings"
	"testing"

	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// opOf is the operation of one action on patient.
func opOf(a model.Action) router.Operation {
	return router.Operation{Module: patientdirectory.ModelName, Entity: (&patientdirectory.Patient{}).ModelName(), Action: a}
}

func TestTenant_ReadOtherTenantIsNotFound(t *testing.T) {
	conn := mem.New()
	db := orm.New(conn)
	modA, _, err := setupModuleWithTenantAndDB("tenant-a", db)
	if err != nil {
		t.Fatal(err)
	}
	modB, _, err := setupModuleWithTenantAndDB("tenant-b", db)
	if err != nil {
		t.Fatal(err)
	}

	callerA := crud.LoopbackActingAs(nil, "u1", modA)
	callerB := crud.LoopbackActingAs(nil, "u1", modB)

	var saved patientdirectory.Patient
	var errA error
	callerA.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient A",
		}}}, &saved, func(e error) { errA = e })
	if errA != nil {
		t.Fatalf("callerA create: %v", errA)
	}

	var list patientdirectory.PatientList
	var errB error
	callerB.Call(opOf(model.Read),
		&crud.ReadArgs{ID: "p1"}, &list, func(e error) { errB = e })
	if errB == nil || model.KindOf(errB) != model.ErrorNotFound {
		t.Fatalf("expected ErrorNotFound, got %v", errB)
	}
}

func TestTenant_ClientTenantRejected(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	var saved patientdirectory.Patient
	var callErr error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			TenantId:     "b",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient",
		}}}, &saved, func(e error) { callErr = e })
	if callErr == nil || callErr.Error() != "loopback: 400 "+string(crud.ErrServerOnlyField) {
		t.Fatalf("expected ErrServerOnlyField, got %v", callErr)
	}
}

func TestCreate_DuplicateDocumentRejected(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	var saved patientdirectory.Patient
	var err1 error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient 1",
		}}}, &saved, func(e error) { err1 = e })
	if err1 != nil {
		t.Fatalf("first create: %v", err1)
	}

	var err2 error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p2",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient 2",
		}}}, &saved, func(e error) { err2 = e })
	if model.FieldOf(err2) != "rut" || !strings.Contains(err2.Error(), string(patientdirectory.ErrRutAlreadyExists)) {
		t.Fatalf("expected field error on rut with ErrRutAlreadyExists, got %v", err2)
	}
}

func TestCreate_RetryIsIdempotent(t *testing.T) {
	mod, pub, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(pub, "u1", mod)

	var saved patientdirectory.Patient
	var err1 error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient 1",
		}}}, &saved, func(e error) { err1 = e })
	if err1 != nil {
		t.Fatalf("first create: %v", err1)
	}

	var err2 error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient 1",
		}}}, &saved, func(e error) { err2 = e })
	if err2 != nil {
		t.Fatalf("retry create: %v", err2)
	}

	createdCount := 0
	for _, e := range pub.published {
		if e.Topic == crud.Topic(patientdirectory.ModelName, "patient", model.Create) {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("expected exactly 1 created event, got %d", createdCount)
	}
}

func TestUpdate_DeactivateIsAnUpdatedEvent(t *testing.T) {
	mod, pub, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(pub, "u1", mod)

	var saved patientdirectory.Patient
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Patient 1",
			IsActive:     true,
		}}}, &saved, func(e error) {})

	pub.published = nil

	var updateErr error
	caller.Call(opOf(model.Update),
		&crud.UpdateArgs{
			IDs:    []string{"p1"},
			Fields: []string{"is_active"},
			Record: &patientdirectory.Patient{IsActive: false},
		}, nil, func(e error) { updateErr = e })
	if updateErr != nil {
		t.Fatalf("update: %v", updateErr)
	}

	updCount := 0
	for _, e := range pub.published {
		if e.Topic == crud.Topic(patientdirectory.ModelName, "patient", model.Update) {
			updCount++
			change, ok := e.Payload.(*crud.Change)
			if !ok || len(change.Fields) != 1 || change.Fields[0] != "is_active" {
				t.Fatalf("the updated event must carry *crud.Change with Fields [is_active], got %+v", e.Payload)
			}
		}
	}
	if updCount != 1 || len(pub.published) != 1 {
		t.Fatalf("a deactivation is one updated event and nothing else, got %d updated of %d events", updCount, len(pub.published))
	}
}

func TestUpdate_OnlyListedFields(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	var saved patientdirectory.Patient
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id:           "p1",
			DocumentType: kinds.DocumentRUT,
			Rut:          "11111111-1",
			Name:         "Original Name",
			Phone:        "+56911111111",
		}}}, &saved, func(e error) {})

	var updateErr error
	caller.Call(opOf(model.Update),
		&crud.UpdateArgs{
			IDs:    []string{"p1"},
			Fields: []string{"phone"},
			Record: &patientdirectory.Patient{Name: "Hacked Name", Phone: "+56922222222"},
		}, nil, func(e error) { updateErr = e })
	if updateErr != nil {
		t.Fatalf("update: %v", updateErr)
	}

	var list patientdirectory.PatientList
	caller.Call(opOf(model.Read),
		&crud.ReadArgs{ID: "p1"}, &list, func(e error) {})
	if len(list) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(list))
	}
	if list[0].Name != "Original Name" {
		t.Fatalf("expected name 'Original Name', got %q", list[0].Name)
	}
	if list[0].Phone != "+56922222222" {
		t.Fatalf("expected phone '+56922222222', got %q", list[0].Phone)
	}
}

func TestUpdate_DocumentRecheckedOnlyWhenTouched(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	_, err = mod.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		DocumentType: kinds.DocumentUnverified,
		Rut:          "invalid-rut-123",
		Name:         "Patient",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	var updateErr error
	caller.Call(opOf(model.Update),
		&crud.UpdateArgs{
			IDs:    []string{"p1"},
			Fields: []string{"phone"},
			Record: &patientdirectory.Patient{Phone: "+56999999999"},
		}, nil, func(e error) { updateErr = e })
	if updateErr != nil {
		t.Fatalf("expected update of phone to succeed without revalidating untransacted document: %v", updateErr)
	}
}

func TestUpdate_DocumentTypeOnlyKeepsStoredRut(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	_, err = mod.ImportPatient(patientdirectory.Patient{
		Id:              "p1",
		DocumentType:    kinds.DocumentPassport,
		DocumentCountry: "US",
		Rut:             "11111111-1",
		Name:            "Foreigner",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	var updateErr error
	caller.Call(opOf(model.Update),
		&crud.UpdateArgs{
			IDs:    []string{"p1"},
			Fields: []string{"document_type"},
			Record: &patientdirectory.Patient{DocumentType: kinds.DocumentRUT},
		}, nil, func(e error) { updateErr = e })
	if updateErr != nil {
		t.Fatalf("update: %v", updateErr)
	}

	var list patientdirectory.PatientList
	caller.Call(opOf(model.Read),
		&crud.ReadArgs{ID: "p1"}, &list, func(e error) {})
	if len(list) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(list))
	}
	if list[0].DocumentType != kinds.DocumentRUT {
		t.Fatalf("expected DocumentRUT, got %v", list[0].DocumentType)
	}
	if list[0].Rut != "11111111-1" {
		t.Fatalf("expected Rut 11111111-1 unchanged, got %q", list[0].Rut)
	}
}

func TestSearch_ByRutPhoneName(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	patients := []patientdirectory.Patient{
		{Id: "p1", DocumentType: kinds.DocumentRUT, Rut: "11111111-1", Name: "Ana Morales", Phone: "+56911112222"},
		{Id: "p2", DocumentType: kinds.DocumentRUT, Rut: "12345678-5", Name: "Bernardo Soto", Phone: "+56933334444"},
		{Id: "p3", DocumentType: kinds.DocumentRUT, Rut: "15678432-K", Name: "Carlos Vega", Phone: "+56955556666"},
	}
	for i := range patients {
		var s patientdirectory.Patient
		caller.Call(opOf(model.Create),
			&crud.CreateArgs{Records: []model.Model{&patients[i]}}, &s, func(e error) {
				if e != nil {
					t.Fatal(e)
				}
			})
	}

	// 1. Search by RUT
	var list1 patientdirectory.PatientList
	var err1 error
	caller.Call(opOf(model.Read),
		&crud.ReadArgs{Search: "12.345.678-5"}, &list1, func(e error) { err1 = e })
	if err1 != nil {
		t.Fatalf("search rut: %v", err1)
	}
	if len(list1) != 1 || list1[0].Id != "p2" {
		t.Fatalf("expected p2, got %v", list1)
	}

	// 2. Search by phone digits
	var list2 patientdirectory.PatientList
	var err2 error
	caller.Call(opOf(model.Read),
		&crud.ReadArgs{Search: "55556666"}, &list2, func(e error) { err2 = e })
	if err2 != nil {
		t.Fatalf("search phone: %v", err2)
	}
	if len(list2) != 1 || list2[0].Id != "p3" {
		t.Fatalf("expected p3, got %v", list2)
	}

	// 3. Search by name fragment
	var list3 patientdirectory.PatientList
	var err3 error
	caller.Call(opOf(model.Read),
		&crud.ReadArgs{Search: "moral"}, &list3, func(e error) { err3 = e })
	if err3 != nil {
		t.Fatalf("search name: %v", err3)
	}
	if len(list3) != 1 || list3[0].Id != "p1" {
		t.Fatalf("expected p1, got %v", list3)
	}
}

func TestDelete_DoesNotExist(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(nil, "u1", mod)

	var delErr error
	caller.Call(opOf(model.Delete),
		&crud.DeleteArgs{IDs: []string{"p1"}}, nil, func(e error) { delErr = e })
	if delErr == nil || !strings.Contains(delErr.Error(), "unknown operation") {
		t.Fatalf("expected unknown operation error, got %v", delErr)
	}
}

func TestImport_Idempotent(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}

	p1 := patientdirectory.Patient{
		Id:       "imp-1",
		Rut:      "11111111-1",
		Name:     "First Version",
		Phone:    "+56911111111",
		IsActive: true,
	}
	_, err = mod.ImportPatient(p1)
	if err != nil {
		t.Fatalf("import 1: %v", err)
	}

	p2 := patientdirectory.Patient{
		Id:       "imp-1",
		Rut:      "11111111-1",
		Name:     "Second Version",
		Phone:    "+56922222222",
		IsActive: true,
	}
	_, err = mod.ImportPatient(p2)
	if err != nil {
		t.Fatalf("import 2: %v", err)
	}

	name, _, phone, _, found, err := mod.GetPatient("test-tenant", "imp-1")
	if err != nil || !found {
		t.Fatalf("GetPatient: found=%v err=%v", found, err)
	}
	if name != "Second Version" || phone != "+56922222222" {
		t.Fatalf("expected fields of second version, got name=%q phone=%q", name, phone)
	}
}

func TestMount_MatchesDefinition(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	if err := crud.Mount(nil, nil, nil, mod); err != nil {
		t.Fatalf("what MountOperations mounts must match Definition(), got %v", err)
	}
}

func TestCreate_PublishesChange(t *testing.T) {
	mod, pub, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	caller := crud.LoopbackActingAs(pub, "u1", mod)
	var saved patientdirectory.Patient
	var createErr error
	caller.Call(opOf(model.Create),
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id: "p1", DocumentType: kinds.DocumentRUT, Rut: "11111111-1", Name: "Patient 1",
		}}}, &saved, func(e error) { createErr = e })
	if createErr != nil {
		t.Fatalf("create: %v", createErr)
	}
	if len(pub.published) != 1 || pub.published[0].Topic != crud.Topic(patientdirectory.ModelName, "patient", model.Create) {
		t.Fatalf("expected one created event, got %+v", pub.published)
	}
	if _, ok := pub.published[0].Payload.(*crud.Change); !ok {
		t.Fatalf("the event body must be *crud.Change, got %T", pub.published[0].Payload)
	}
}

// noAnswerModule mounts patient with a Create rule that answers nothing (the zero RuleAnswer).
type noAnswerModule struct{ store crud.Store }

func (noAnswerModule) ModelName() string { return "no_answer" }
func (m noAnswerModule) Definition() model.ModuleDefinition {
	return model.ModuleDefinition{Name: m.ModelName(), Entities: []model.Definition{{
		Name: "patient", Allow: patientdirectory.PatientModel.Allow, Rules: patientdirectory.PatientModel.Rules}}}
}
func (m noAnswerModule) MountOperations(reg crud.Registry) {
	reg.AddStoredEntity(crud.StoredEntity{
		Model:   &patientdirectory.Patient{},
		NewList: func() model.ModelSlice { return &patientdirectory.PatientList{} },
		Store:   m.store,
		Rules:   patientdirectory.PatientModel.Rules,
		Create: func(a crud.CurrentUser, tx crud.Tx, rec model.Model) (crud.RuleAnswer, error) {
			return 0, nil
		},
		Update: func(a crud.CurrentUser, tx crud.Tx, old, next model.Model, fields []string) (crud.RuleAnswer, error) {
			return crud.ApplyAndPublish, nil
		},
		Search: func(a crud.CurrentUser, filter model.Model, text string) ([]storage.Condition, error) {
			return nil, nil
		},
	})
}

func TestRule_NoAnswerIsRefused(t *testing.T) {
	mod, _, err := setupModule()
	if err != nil {
		t.Fatal(err)
	}
	bad := noAnswerModule{store: mod.StorePatient()}
	caller := crud.LoopbackActingAs(nil, "u1", bad)
	var createErr error
	caller.Call(router.Operation{Module: "no_answer", Entity: "patient", Action: model.Create},
		&crud.CreateArgs{Records: []model.Model{&patientdirectory.Patient{
			Id: "p9", DocumentType: kinds.DocumentRUT, Rut: "22222222-2", Name: "Patient 9",
		}}}, nil, func(e error) { createErr = e })
	if createErr == nil {
		t.Fatal("a rule that answers nothing must be refused (crud.ErrNoAnswer, shown to the client as an internal error)")
	}
	if _, _, _, _, found, _ := mod.GetPatient("test-tenant", "p9"); found {
		t.Fatal("nothing may be stored when the rule answered nothing")
	}
}
