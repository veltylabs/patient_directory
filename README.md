# Directorio de Pacientes (`patient_directory`)
<img src="docs/img/badges.svg">

Registro de pacientes delimitado por tenant (organización/clínica) para el ecosistema Velty.

Este módulo es propietario de la **identidad** y **datos de contacto** del paciente (quién es una persona y cómo contactarla). **No posee datos clínicos**. Los diagnósticos, atenciones, recetas e historial pertenecen a `clinical_encounter`; las citas médicas pertenecen a `appointment_booking`.

## View and demo

This module provides its own UI sub-package `ui/`, demo data loader `seed/`, and runnable demo in `web/`:

- `ui`: Exports `ID`, `Label`, and `Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)`.
- `seed`: Exports `Load(m *patientdirectory.Module, tenantID string) (Data, error)` which writes through module domain methods to populate initial seed patients.
- `web`: Run `webtyp` at the repository root to open the demo — in-browser, in-memory, no login.

## Operaciones

| Operación | Recurso | Acción | Descripción |
|---|---|---|---|
| `list_patients` | `patient` | `Read` | Lista pacientes de un tenant con opciones de `active_only`, `limit`, `offset` |
| `get_patient` | `patient` | `Read` | Obtiene un paciente por su ID dentro de un tenant |
| `find_patient_by_rut` | `patient` | `Read` | Busca un paciente por RUT dentro de un tenant |
| `upsert_patient` | `patient` | `Create\|Update` | Crea un nuevo paciente o actualiza uno existente |
| `deactivate_patient` | `patient` | `Update` | Desactiva un paciente (`is_active = false`) |

*Nota: No existe operación de eliminación física. Los pacientes se desactivan, nunca se eliminan.*

## Puertos satisfechos

Este módulo satisface la interfaz del puerto `DirectoryReader` declarado por `appointment_booking`:

```go
type DirectoryReader interface {
    ClientExists(tenantId, clientId string) (bool, error)
}
```

- `ClientExists(tenantID, clientID)` retorna `(true, nil)` si un paciente con `clientID` existe en `tenantID`, independientemente de si el paciente está activo o inactivo.
- Retorna `(false, nil)` si el paciente no existe o pertenece a otro tenant.

## Inicio rápido

### Migración

Ejecute las migraciones de esquema antes de iniciar el módulo de servicio:

```go
import (
    "github.com/veltylabs/patient_directory/migrate"
)

// conn satisface ddl.Execer y ddlCompiler satisface ddl.Compiler
err := migrate.Migrate(conn, ddlCompiler)
```

### Inicialización del módulo

```go
import (
    patientdirectory "github.com/veltylabs/patient_directory"
)

deps := patientdirectory.Deps{
    IDs: idGenerator, // model.IDGenerator
    Publisher: eventPublisher, // events.Publisher (opcional)
    ValidateRUT: func(rut string) (string, error) {
        // Normaliza y valida el RUT retornando la cadena canónica
        return normalisedRut, nil
    },
}

module, err := patientdirectory.New(db, deps)
```

## Archivos clave

- `model.go` — Definiciones de modelos para `Patient` y argumentos de operaciones.
- `model_orm.go` — Enlaces ORM generados (vía `ormc`).
- `module.go` — Lógica de dominio del servicio e implementación del puerto `ClientExists`.
- `ops.go` — Controladores de operaciones del router y su registro.
- `view.go` — Motor de presentación de vista (`NewView`).
- `migrate/migrate.go` — Sincronización del esquema de base de datos.
