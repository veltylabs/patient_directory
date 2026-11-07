package tests

import (
	"strconv"
	"testing"

	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/view"

	patientdirectory "github.com/veltylabs/patient_directory"
	"github.com/veltylabs/patient_directory/kinds"
)

func setupTestRouter(t *testing.T) (*patientdirectory.Module, *mock.Router) {
	t.Helper()
	m, _, err := setupModule()
	if err != nil {
		t.Fatalf("setupModule: %v", err)
	}
	r := &mock.Router{}
	m.MountOperations(r)
	r.Configure(mock.Config{
		Authn: func(next router.HandlerFunc) router.HandlerFunc {
			return func(ctx router.Context) {
				ctx.SetUserID("u1")
				next(ctx)
			}
		},
		Authorize: func(userID string, resource model.Resource, action model.Action) bool {
			return true
		},
	})
	return m, r
}

func callListPatients(t *testing.T, r *mock.Router, args view.ListArgs) patientdirectory.PatientList {
	t.Helper()
	var body []byte
	if err := json.Encode(&args, &body); err != nil {
		t.Fatalf("encode args: %v", err)
	}
	ctx := &mock.Context{InBody: body}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+patientdirectory.OpListPatients, ctx)
	if ctx.Status != 0 && ctx.Status != 200 {
		t.Fatalf("call list_patients failed: %d, body=%s", ctx.Status, ctx.ResponseBody())
	}
	var list patientdirectory.PatientList
	if err := json.Decode(ctx.ResponseBody(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return list
}

func TestPatient_ListAcceptsListArgs(t *testing.T) {
	m, r := setupTestRouter(t)

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "p1",
		TenantId:     "test-tenant",
		DocumentType: kinds.DocumentRUT,
		Rut:          "11111111-1",
		Name:         "carlos gomez",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("ImportPatient p1: %v", err)
	}
	_, err = m.ImportPatient(patientdirectory.Patient{
		Id:           "p2",
		TenantId:     "test-tenant",
		DocumentType: kinds.DocumentRUT,
		Rut:          "22222222-2",
		Name:         "beatriz silva",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("ImportPatient p2: %v", err)
	}

	// 1. Limit: 1
	list := callListPatients(t, r, view.ListArgs{Limit: 1})
	if len(list) != 1 {
		t.Fatalf("expected 1 patient with Limit: 1, got %d", len(list))
	}

	// 2. Search: Carlos Gomez
	list = callListPatients(t, r, view.ListArgs{Search: "Carlos Gomez"})
	if len(list) != 1 || list[0].Id != "p1" {
		t.Fatalf("expected p1 with search 'Carlos Gomez', got %v", list)
	}

	// 3. After: p1
	list = callListPatients(t, r, view.ListArgs{After: "p1"})
	if len(list) != 1 || list[0].Id != "p2" {
		t.Fatalf("expected p2 after p1, got %v", list)
	}
}

func TestListPatients_SearchByRutAnyWayWritten(t *testing.T) {
	m, r := setupTestRouter(t)

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "p-rut",
		TenantId:     "test-tenant",
		DocumentType: kinds.DocumentRUT,
		Rut:          "8123456-6",
		Name:         "Juan Perez",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("ImportPatient: %v", err)
	}

	// Search formatted with dots and hyphen: 8.123.456-6
	list := callListPatients(t, r, view.ListArgs{Search: "8.123.456-6"})
	if len(list) != 1 || list[0].Id != "p-rut" {
		t.Fatalf("expected p-rut for search '8.123.456-6', got %v", list)
	}

	// Search with leading zero: 08123456-6
	list = callListPatients(t, r, view.ListArgs{Search: "08123456-6"})
	if len(list) != 1 || list[0].Id != "p-rut" {
		t.Fatalf("expected p-rut for search '08123456-6', got %v", list)
	}
}

func TestListPatients_SearchByName(t *testing.T) {
	m, r := setupTestRouter(t)

	_, err := m.ImportPatient(patientdirectory.Patient{
		Id:           "p-name",
		TenantId:     "test-tenant",
		DocumentType: kinds.DocumentRUT,
		Rut:          "12345678-5",
		Name:         "pedro alcantara",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("ImportPatient: %v", err)
	}

	list := callListPatients(t, r, view.ListArgs{Search: "Pedro"})
	if len(list) != 1 || list[0].Id != "p-name" {
		t.Fatalf("expected p-name for search 'Pedro', got %v", list)
	}
}

func TestListPatients_PageOf20ThenMore(t *testing.T) {
	m, r := setupTestRouter(t)

	// Seed 25 patients named "maria ..."
	for i := 1; i <= 25; i++ {
		id := "p" + strconv.Itoa(100+i)
		_, err := m.ImportPatient(patientdirectory.Patient{
			Id:           id,
			TenantId:     "test-tenant",
			DocumentType: kinds.DocumentRUT,
			Rut:          strconv.Itoa(i) + "0000000-0",
			Name:         "maria paciente " + strconv.Itoa(i),
			IsActive:     true,
		})
		if err != nil {
			t.Fatalf("ImportPatient %d: %v", i, err)
		}
	}

	// Limit 20 -> 20 rows
	first20 := callListPatients(t, r, view.ListArgs{Search: "maria", Limit: 20})
	if len(first20) != 20 {
		t.Fatalf("expected 20 rows for first page, got %d", len(first20))
	}

	// After the 20th -> 5 rows
	lastId := first20[len(first20)-1].Id
	remaining := callListPatients(t, r, view.ListArgs{Search: "maria", After: lastId, Limit: 20})
	if len(remaining) != 5 {
		t.Fatalf("expected 5 rows after %s, got %d", lastId, len(remaining))
	}
}
