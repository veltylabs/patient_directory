# Arquitectura y Decisiones de Diseño

## Límite del Dominio

`patient_directory` es propietario de la **identidad** y **datos de contacto** del paciente:
- Identidad: ID interno, identificador nacional (RUT), tipo de documento, país emisor, nombre completo, sexo, fecha de nacimiento.
- Contacto: número de teléfono, correo electrónico, dirección física.
- Estado: indicador de activo (`is_active`), marca de tiempo de actualización (`updated_at`).

Explícitamente **no** es propietario de:
- Atenciones clínicas, diagnósticos, fichas ni recetas (propiedad de `clinical_encounter`).
- Citas médicas ni estado de agenda (propiedad de `appointment_booking`).
- Cuentas de usuarios de personal o profesionales de la salud (propiedad de `staff_manager`).

## Regla de los Seis Verbos e Integración entre Módulos

Siguiendo la regla de diseño del ecosistema (**module-layout** «Six verbs, one name per job»), este módulo no exporta tipos complejos o estructuras propias a otros módulos. 

Los métodos exportados para interoperabilidad retornan y reciben únicamente tipos planos:
- `GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error)`
- `FindPatient(tenantID, documentNumber string) (id, name, phone, email string, found bool, err error)`
- `CreatePatient(tenantID, id, documentNumber, name, phone, email string) (patientID string, err error)`
- `ImportPatient(p Patient) (Patient, error)` (reservado para migraciones y carga inicial)

Las formas que manejan la estructura interna `Patient` (`getPatient`, `createPatient`, `updatePatient`, `listPatients`, `deactivatePatient`, `findByDocument`) son no exportadas y de uso exclusivo interno y para las operaciones del router (`ops.go`).

Cualquier módulo consumidor (`appointment_booking`, `clinical_encounter`, etc.) declara en su propio `interfaces.go` únicamente las firmas de los métodos planos que necesita, satisfaciéndose estructuralmente sin adaptadores y con cero dependencias cruzadas en `go.mod`.

## Integración con Appkit

El submódulo `module/` exporta el valor listo para `App.Modules`:

```go
package config

import (
	patientmodule "github.com/veltylabs/patient_directory/module"
	"webtyp.com/appkit"
)

var App = appkit.App{
	Modules: []appkit.Module{
		patientmodule.Module,
	},
}
```

## Decisiones Arquitectónicas Principales

1. **Identidad y Alcance por Tenant:**
   - La unicidad de documento está delimitada por tenant: el mismo documento puede existir en diferentes clínicas de forma independiente.

2. **Control de Unicidad en Capa de Aplicación:**
   - `patient_directory` verifica la existencia del documento en `createPatient` y `updatePatient` retornando `ErrRutAlreadyExists`. `PatientModel` no lleva la marca `Unique: true` a nivel de columna para permitir multitenancy limpio.

3. **Desactivación en lugar de Eliminación:**
   - Los pacientes nunca se eliminan físicamente. `deactivatePatient` establece `is_active = false`.
   - Los registros históricos y reservas que referencian a un ID de paciente siempre deben poder resolverse.

4. **Validación de RUT Inyectada:**
   - La función `Deps.ValidateRUT` se inyecta en `patientdirectory.New`.
   - El módulo no depende de ninguna librería específica de RUT ni implementa algoritmos de dígito verificador propios.

## Documento de identidad

El paciente se identifica por su **id interno**, no por un documento: muchos pacientes extranjeros no tienen RUT, a
veces solo el pasaporte de su país, y a veces nada a mano. Exigir el RUT frenaba la atención.

| `document_type` | Qué es | Validación |
|---|---|---|
| `rut` (por defecto) | RUT chileno | dígito verificador; se normaliza |
| `passport` | pasaporte de cualquier país (OACI 9303: hasta 9 letras y dígitos) | número obligatorio; se guarda en mayúsculas |
| `foreign_id` | cédula de identidad de otro país | número obligatorio; se guarda en mayúsculas |
| `none` | sin documento a mano | número y país vacíos |
| `unverified` | ingresado pero inválido | se guarda tal cual; recepción lo corrige |

No se repite la combinación (tipo, país, número) dentro de un tenant, y solo cuando hay número y el tipo no es
`unverified`: dos pacientes sin documento pueden convivir. Es el modelo de HL7 FHIR (`Patient.identifier` =
{tipo, emisor, número}).

### ¿Por qué el teléfono no es un identificador?

Una familia comparte el número, un menor usa el de su madre y la gente cambia de número: si fuera único, rechazaría a
pacientes reales. Sirve para **buscar**, junto con el nombre y la fecha de nacimiento.
