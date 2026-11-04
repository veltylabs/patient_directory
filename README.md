# Directorio de Pacientes (`patient_directory`)
<img src="docs/img/badges.svg">

Registro de pacientes delimitado por tenant (organización/clínica) para el ecosistema Velty.

Este módulo es propietario de la **identidad** y **datos de contacto** del paciente (quién es una persona y cómo contactarla). **No posee datos clínicos**. Los diagnósticos, atenciones, recetas e historial pertenecen a `clinical_encounter`; las citas médicas pertenecen a `appointment_booking`.

## View and demo

Este módulo provee su propio subpaquete UI `ui/`, cargador de datos de prueba `seed/`, y demostración ejecutable en `web/`:

- `ui`: Exporta `ID`, `Label`, y `Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)`.
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
| `link_patient` | `patient_link` | `Create` | Devuelve solo el id; nunca modifica un paciente existente. Si no existe, lo crea con los datos proporcionados. Usado para reserva online |

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

## Documento de identidad

Un paciente se identifica por su **id interno**; el documento es un dato que puede faltar. `document_type` puede ser
`rut` (por defecto, se valida el dígito verificador), `passport`, `foreign_id` (cédula de otro país), `none` (sin
documento a mano) o `unverified` (ingresado pero inválido: recepción lo corrige). El número va en la columna `rut`
(mantiene su nombre: otros módulos usan `FindByRut`) y el país emisor en `document_country` (código de 2 letras: `VE`,
`PE`, `HT`). Detalle y razones: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#documento-de-identidad).

## Importar pacientes históricos

`ImportPatient` guarda un paciente de un sistema anterior con el id que tenía allá; si ese id ya existe, lo actualiza.
Valida igual que `CreatePatient` y no publica eventos.

```go
p, err := dir.ImportPatient(patientdirectory.Patient{
	Id:           "1640011384447734400",
	TenantId:     tenantID,
	DocumentType: kinds.DocumentRUT,
	Rut:          "12345678-5",
	Name:         "Ana",
})
```
