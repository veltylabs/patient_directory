package patientdirectory

import "webtyp.com/crud"

// Update es la regla de negocio de actualizar patient: old es la fila guardada y el receptor trae solo los campos enviados (fields).
// Vuelve a validar el documento solo si cambió alguno de sus campos.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (next *Patient) Update(a crud.CurrentUser, tx crud.Tx, old *Patient, fields []string, m *Module) (crud.RuleAnswer, error) {
	if !touches(fields, Patient_.Rut, Patient_.DocumentType, Patient_.DocumentCountry) {
		return crud.ApplyAndPublish, nil
	}
	// next trae solo los campos enviados: se mezcla con lo guardado campo por campo.
	merged := *old
	if touches(fields, Patient_.Rut) {
		merged.Rut = next.Rut
	}
	if touches(fields, Patient_.DocumentType) {
		merged.DocumentType = next.DocumentType
	}
	if touches(fields, Patient_.DocumentCountry) {
		merged.DocumentCountry = next.DocumentCountry
	}
	if err := m.checkDocument(&merged); err != nil {
		return 0, err
	}
	// Solo se escriben las columnas de fields: se devuelve la forma normalizada de las enviadas.
	next.Rut, next.DocumentType, next.DocumentCountry = merged.Rut, merged.DocumentType, merged.DocumentCountry
	if err := m.sameDocument(tx, &merged); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}

func touches(fields []string, names ...string) bool {
	for _, f := range fields {
		for _, n := range names {
			if f == n {
				return true
			}
		}
	}
	return false
}
