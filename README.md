# Directorio de Pacientes (`patient_directory`)
<img src="docs/img/badges.svg">

Registro de pacientes delimitado por tenant (organización/clínica) para el ecosistema Velty.

Este módulo es propietario de la **identidad** y **datos de contacto** del paciente (quién es una persona y cómo contactarla). **No posee datos clínicos**. Los diagnósticos, atenciones, recetas e historial pertenecen a `clinical_encounter`; las citas médicas pertenecen a `appointment_booking`.

## View and demo

Este módulo provee su propio subpaquete UI `ui/`, cargador de datos de prueba `seed/`, y demostración ejecutable en `web/`:

- `ui`: Exporta `ID`, `Label`, y `Screens(caller router.Caller) []view.Screen`.
- `seed`: Exporta `Load(m *patientdirectory.Module, tenantID string) (Data, error)` que escribe mediante métodos de dominio del módulo para rellenar pacientes de prueba iniciales.
- `web`: Ejecute `webtyp` en la raíz del repositorio para abrir la demo — en el navegador, en memoria, sin inicio de sesión.

## Operaciones

| Operación | Recurso | Acción | Descripción |
|---|---|---|---|
| `list_patients` | `patient` | `Read` | Lista pacientes de un tenant con opciones de `active_only`, `limit`, `offset` |
| `get_patient` | `patient` | `Read` | Obtiene un paciente por su ID dentro de un tenant |
| `find_patient_by_rut` | `patient` | `Read` | Busca un paciente por RUT dentro de un tenant |
| `upsert_patient` | `patient` | `Create\|Update` | Crea un nuevo paciente o actualiza uno existente |
| `deactivate_patient` | `patient` | `Update` | Desactiva un paciente (`is_active = false`) |

*Nota: No existe operación de eliminación física. Los pacientes se desactivan, nunca se eliminan.*

## Interoperabilidad (Regla de los seis verbos)

Para comunicarse con otros módulos sin compartir estructuras de datos, este módulo exporta tres métodos de tipos planos:

```go
GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error)
FindPatient(tenantID, documentNumber string) (id, name, phone, email string, found bool, err error)
CreatePatient(tenantID, id, documentNumber, name, phone, email string) (patientID string, err error)
```

Cualquier módulo consumidor (`appointment_booking`, `clinical_encounter`) declara en su propio `interfaces.go` las firmas exactas que necesita y este módulo las satisface estructuralmente.

## Inicio rápido

### Paso a paso con appkit

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

### Inicialización directa

```go
import (
    patientdirectory "github.com/veltylabs/patient_directory"
)

deps := patientdirectory.Deps{
    Publisher:   eventPublisher, // events.Publisher (opcional)
    TenantID:    "tenant-1",
    ValidateRUT: func(rut string) (string, error) {
        return normalisedRut, nil
    },
}

module, err := patientdirectory.New(db, deps)
```

## Archivos clave

- `model.go` — Definición de modelo `Patient` y argumentos de operaciones.
- `model_orm.go` — Enlaces ORM generados (vía `ormc`).
- `readers.go` — Métodos exportados con tipos planos (`GetPatient`, `FindPatient`, `CreatePatient`).
- `module.go` — Constructor `New`, lógica interna y métodos de servicio no exportados.
- `module/` — Empaquetado `appkit.Module` para servidor (`server.go`) y navegador (`browser.go`).
- `ops.go` — Controladores de operaciones del router y su registro.
- `view.go` — Motor de presentación de vista (`NewView`).
- `migrate/migrate.go` — Sincronización del esquema de base de datos.

## Documento de identidad

Un paciente se identifica por su **id interno**; el documento es un dato que puede faltar. `document_type` puede ser
`rut` (por defecto, se valida el dígito verificador), `passport`, `foreign_id` (cédula de otro país), `none` (sin
documento a mano) o `unverified` (ingresado pero inválido: recepción lo corrige). El número va en la columna `rut`
y el país emisor en `document_country`. Detalle y razones: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#documento-de-identidad).

## Importar pacientes históricos

`ImportPatient` guarda un paciente de un sistema anterior con el id que tenía allá; si ese id ya existe, lo actualiza.
Valida igual que `createPatient` y no publica eventos.

```go
p, err := dir.ImportPatient(patientdirectory.Patient{
	Id:           "1640011384447734400",
	TenantId:     tenantID,
	DocumentType: kinds.DocumentRUT,
	Rut:          "12345678-5",
	Name:         "Ana",
})
```
