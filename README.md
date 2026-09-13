# Patient Directory (`patient_directory`)

Tenant-scoped patient registry for the Velty ecosystem.

This module owns patient identity and contact details (who a person is and how to reach them). It owns **no clinical data**. Diagnoses, visits, prescriptions, and history belong to `clinical_encounter`; appointments belong to `appointment_booking`.

## Operations

| Op | Resource | Action | Description |
|---|---|---|---|
| `list_patients` | `patient` | `Read` | List patients for a tenant with optional `active_only`, `limit`, `offset` |
| `get_patient` | `patient` | `Read` | Retrieve a patient by ID within a tenant |
| `find_patient_by_rut` | `patient` | `Read` | Find a patient by RUT within a tenant |
| `upsert_patient` | `patient` | `Create\|Update` | Create a new patient or update an existing patient |
| `deactivate_patient` | `patient` | `Update` | Deactivate a patient (`is_active = false`) |

*Note: There is no delete operation. Patients are deactivated, never deleted.*

## Ports Satisfied

This module satisfies the narrow `DirectoryReader` port declared by `appointment_booking`:

```go
type DirectoryReader interface {
    ClientExists(tenantId, clientId string) (bool, error)
}
```

- `ClientExists(tenantID, clientID)` returns `(true, nil)` if a patient with `clientID` exists in `tenantID`, regardless of whether the patient is active or inactive.
- Returns `(false, nil)` if the patient does not exist or belongs to another tenant.

## Quick Start

### Migration

Execute schema migrations before starting the service module:

```go
import (
    "github.com/veltylabs/patient_directory/migrate"
)

// conn satisfies ddl.Execer and ddlCompiler satisfies ddl.Compiler
err := migrate.Migrate(conn, ddlCompiler)
```

### Module Initialization

```go
import (
    patientdirectory "github.com/veltylabs/patient_directory"
)

deps := patientdirectory.Deps{
    IDs: idGenerator, // model.IDGenerator
    Publisher: eventPublisher, // events.Publisher (optional)
    ValidateRUT: func(rut string) (string, error) {
        // Normalises and validates RUT returning canonical string
        return normalisedRut, nil
    },
}

module, err := patientdirectory.New(db, deps)
```

## Key Files

- `model.go` — Model definitions for `Patient` and operation arguments
- `model_orm.go` — Generated ORM bindings (via `ormc`)
- `module.go` — Domain service logic and `ClientExists` port implementation
- `ops.go` — Router operations handler and registration
- `view.go` — View presenter engine (`NewView`)
- `migrate/migrate.go` — Database schema synchronization
