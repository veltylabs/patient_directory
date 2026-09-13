# Arquitectura y Decisiones de Diseño

## Límite del Dominio

`patient_directory` es propietario de la **identidad** y **datos de contacto** del paciente:
- Identidad: ID interno, identificador nacional (RUT), nombre completo, fecha de nacimiento.
- Contacto: número de teléfono, correo electrónico, dirección física.
- Estado: indicador de activo (`is_active`), marca de tiempo de actualización (`updated_at`).

Explícitamente **no** es propietario de:
- Atenciones clínicas, diagnósticos, fichas ni recetas (propiedad de `clinical_encounter`).
- Citas médicas ni estado de agenda (propiedad de `appointment_booking`).
- Cuentas de usuarios de personal o profesionales de la salud (propiedad de `staff_manager`).

## Decisiones Arquitectónicas Principales

1. **Identidad por RUT y Alcance por Tenant:**
   - Las clínicas chilenas identifican a los pacientes mediante el RUT nacional.
   - La unicidad está delimitada por tenant: el mismo RUT puede existir en diferentes tenants (clínicas) de forma independiente.

2. **Control de Unicidad en Capa de Aplicación:**
   - Una restricción de unicidad de una sola columna en `rut` fallaría en bases de datos multitenant.
   - `patient_directory` verifica la existencia del RUT en `CreatePatient` y `UpdatePatient` retornando `ErrRutAlreadyExists`. `PatientModel` no lleva la marca `Unique: true`.

3. **Desactivación en lugar de Eliminación:**
   - Los pacientes nunca se eliminan físicamente. `DeactivatePatient` establece `is_active = false`.
   - Los registros históricos y reservas que referencian a un ID de paciente siempre deben poder resolverse.

4. **Validación de RUT Inyectada:**
   - La función `Deps.ValidateRUT` se inyecta en `patientdirectory.New`.
   - El módulo no depende de ninguna librería específica de RUT ni implementa algoritmos de dígito verificador.

5. **Sin Cadenas de UI / Sin Traducciones de Idioma:**
   - Los controladores y modelos retornan constantes de dominio y errores estándar.
   - La traducción de etiquetas para los usuarios finales es responsabilidad de las interfaces web/móviles consumidoras.

## Ejemplo de Ensamblado en Raíz de Composición (Composition Root)

Conexión de `patient_directory` con `appointment_booking`:

```go
package main

import (
	"github.com/veltylabs/patient_directory"
	// "github.com/veltylabs/appointment_booking"
)

func main() {
	// 1. Inicializar módulo patient_directory
	patientMod, err := patientdirectory.New(db, patientdirectory.Deps{
		IDs: idGen,
		ValidateRUT: myRutValidator,
		Publisher: eventPub,
	})
	if err != nil {
		log.Fatalf("error al inicializar patient_directory: %v", err)
	}

	// 2. Conectar directamente con appointment_booking sin adaptador (satisface estructuralmente ClientExists)
	/*
	bookingMod, err := appointmentbooking.New(db, appointmentbooking.Deps{
		Directory: patientMod, // satisface la interfaz DirectoryReader
	})
	*/
	_ = patientMod
}
```
