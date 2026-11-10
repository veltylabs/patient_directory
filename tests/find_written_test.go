package tests

import (
	"testing"

	patientdirectory "github.com/veltylabs/patient_directory"
	"webtyp.com/events/mock"
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

// En Pa100T, 354 RUT quedaron repetidos porque recepción buscaba al paciente, no lo encontraba con el RUT
// escrito de otra forma y lo creaba de nuevo. Con el validador real (nunca el falso de setup_test.go): toda
// forma en que se escribe el mismo RUT encuentra al mismo paciente, y crearlo otra vez es rechazado.
func TestFindPatient_AnyWayTheRutIsWritten(t *testing.T) {
	m, err := patientdirectory.New(orm.New(mem.New()), patientdirectory.Deps{
		Publisher:   &mock.Broker{},
		TenantID:    "t1",
		IDs:         &testIDGen{},
		ValidateRUT: input.NormalizeRUT,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := m.CreatePatient("t1", "p1", "8.123.456-6", "Ana", "", ""); err != nil {
		t.Fatalf("CreatePatient: %v", err)
	}
	for i, written := range []string{"8123456-6", "81234566", "08.123.456-6", "8 123 456-6", " 8.123.456 - 6 "} {
		id, _, _, _, found, err := m.FindPatient("t1", written)
		if err != nil || !found || id != "p1" {
			t.Errorf("FindPatient(%q): id=%q found=%v err=%v", written, id, found, err)
		}
		_, err = m.CreatePatient("t1", "dup"+string(rune('a'+i)), written, "Ana", "", "")
		fe, ok := err.(*model.FieldError)
		if !ok || fe.Err != patientdirectory.ErrRutAlreadyExists {
			t.Errorf("CreatePatient(%q) otra vez: se esperaba conflicto (ya existe), se obtuvo %v", written, err)
		}
	}
}
