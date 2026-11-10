# Directorio de Pacientes (`patient_directory`)

<img src="docs/img/badges.svg">

Registro institucional de pacientes delimitado por organización (tenant) para el ecosistema Velty. Provee identidad y datos de contacto; los diagnósticos y consultas pertenecen a [`clinical_encounter`](https://github.com/veltylabs/clinical_encounter).

## 1. ¿Por qué existe?

Gestiona la identidad de pacientes aislando cada clínica en su propio tenant. La persistencia estándar la provee [`webtyp.com/crud`](https://github.com/webtyp/crud) a partir de los permisos y reglas declarados en [`definition.go`](definition.go); el módulo implementa exclusivamente las reglas de negocio en los archivos `rule_patient_*.go`.

| Sin esto | Con esto |
|---|---|
| Cientos de líneas repetidas en cada módulo para transacciones, filtros por tenant, paginación y validaciones CRUD manuales. | Persistencia automática gobernada por `Allow: model.Create \| model.Read \| model.Update` en el esquema, reduciendo el módulo a sus reglas clínicas. |
| Riesgo de fuga de datos entre clínicas al olvidar condiciones de tenant en consultas manuales. | Aislamiento garantizado por diseño mediante [`orm.TenantStore`](https://github.com/webtyp/orm) en cada operación y lector. |

### Operaciones CRUD

Montadas por `MountOperations`, que `definitionc` genera en [`definition_gen.go`](definition_gen.go) a partir de [`Definition`](definition.go) (el módulo no escribe `ops.go`):

| Operación (`router.Operation`) | Cable REST | Permiso | Descripción |
|---|---|---|---|
| `patient_directory.patient.read` | `GET /api/patient_directory/patient` | Read | Consulta y búsqueda paginada por RUT, teléfono o nombre ([`rule_patient_search.go`](rule_patient_search.go)) |
| `patient_directory.patient.create` | `POST /api/patient_directory/patient` | Create | Registro de paciente con validación de documento y unicidad ([`rule_patient_create.go`](rule_patient_create.go)) |
| `patient_directory.patient.update` | `PATCH /api/patient_directory/patient` | Update | Modificación de campos permitidos o desactivación (`is_active = false`) ([`rule_patient_update.go`](rule_patient_update.go)) |

*Nota: no existe la operación de borrar. Los pacientes se desactivan, nunca se eliminan físicamente. `crud` publica `patient_directory.patient.created` y `patient_directory.patient.updated` con un `*crud.Change` tras cada commit; desactivar es un `updated` cuyo `Fields` incluye `is_active`.*

## 2. Antes de comenzar

- Base de datos compatible con [`webtyp.com/storage`](https://github.com/webtyp/storage) (SQLite, PostgreSQL o en memoria).
- Generador de identificadores único [`model.IDGenerator`](https://github.com/webtyp/model) (`webtyp.com/unixid`).
- Función validadora o normalizadora de documentos (por ejemplo [`input.NormalizeRUT`](https://github.com/webtyp/input)).

## 3. Guía paso a paso

### Paso 1: Configurar en una aplicación webtyp

**Meta:** Montar el módulo en el servidor y navegador mediante [`webtyp.com/appkit`](https://github.com/webtyp/appkit).

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

**Resultado:** Las operaciones `read`, `create` y `update` de `patient` quedan disponibles para el cliente.

### Paso 2: Interoperabilidad entre módulos (Regla de tipos planos)

**Meta:** Consultar o crear pacientes desde otro módulo sin acoplar dependencias de paquetes.

```go
package main

import (
	patientdirectory "github.com/veltylabs/patient_directory"
	"webtyp.com/fmt"
)

func consultar(m *patientdirectory.Module, tenantID, id string) {
	name, doc, phone, email, found, err := m.GetPatient(tenantID, id)
	if err != nil || !found {
		fmt.Println("paciente no encontrado")
		return
	}
	fmt.Println(name, doc, phone, email)
}
```

**Resultado:** Datos de identidad recuperados de forma segura respetando el aislamiento multitenant.

## 4. Mapa de documentación

| Orden | Documento | Qué pregunta responde y cuándo leerlo |
|---|---|---|
| 1 | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | ¿Cómo interactúan el modelo, el almacén multitenant y las reglas de dominio? |
| 2 | [docs/TESTS.md](docs/TESTS.md) | ¿Cuáles son los contratos y escenarios de prueba verificados en suite? |
