package seed

import (
	patientdirectory "github.com/veltylabs/patient_directory"
)

// Data contiene las filas de demostración creadas por Load.
type Data struct {
	Patients []patientdirectory.Patient
}

// Load inserta los datos semilla en el módulo usando sus métodos de dominio.
func Load(m *patientdirectory.Module, tenantID string) (Data, error) {
	items := []patientdirectory.Patient{
		{
			TenantId:  tenantID,
			Rut:       "11111111-1",
			Name:      "Juan Pérez",
			Birthdate: "1980-05-12",
			Phone:     "+56911111111",
			Email:     "juan.perez@example.cl",
			Address:   "Chillán",
			IsActive:  true,
		},
		{
			TenantId:  tenantID,
			Rut:       "12345678-5",
			Name:      "María González",
			Birthdate: "1992-11-03",
			Phone:     "+56922222222",
			Email:     "maria.gonzalez@example.cl",
			Address:   "Chillán",
			IsActive:  true,
		},
		{
			TenantId:  tenantID,
			Rut:       "15678432-K",
			Name:      "Pedro Soto",
			Birthdate: "1975-02-20",
			Phone:     "+56933333333",
			Email:     "pedro.soto@example.cl",
			Address:   "San Carlos",
			IsActive:  true,
		},
	}

	var data Data
	for i := range items {
		p, err := m.CreatePatient(items[i])
		if err != nil {
			return Data{}, err
		}
		data.Patients = append(data.Patients, p)
	}

	return data, nil
}
