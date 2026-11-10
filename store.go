package patientdirectory

import "webtyp.com/crud"

// StorePatient is the tenant-sealed store of the patients.
func (m *Module) StorePatient() crud.Store { return m.store }
