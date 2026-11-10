package patientdirectory

import (
	"strings"

	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/crud"
	"webtyp.com/input"
	"webtyp.com/storage"
)

// Search traduce el texto que escribió la persona y los campos que envió (filter) en las condiciones que buscan patient. Solo condiciones unidas con AND (nunca OR).
// crud ya aplicó el tope de página, el cursor y la clínica; tú solo dices qué filas corresponden. text es "" cuando no se envió texto.
func (p *Patient) Search(a crud.CurrentUser, filter *Patient, text string, m *Module) ([]storage.Condition, error) {
	if text == "" {
		return nil, nil
	}
	if norm, err := input.NormalizeRUT(text); err == nil {
		return []storage.Condition{storage.Eq(Patient_.DocumentType, kinds.DocumentRUT), storage.In(Patient_.Rut, []string{norm, strings.ReplaceAll(norm, "-", "")})}, nil
	}
	if isPhone(text) {
		return []storage.Condition{storage.Like(Patient_.Phone, "%"+digitsOnly(text)+"%")}, nil
	}
	return []storage.Condition{storage.Like(Patient_.Name, "%"+strings.ToLower(text)+"%")}, nil
}
