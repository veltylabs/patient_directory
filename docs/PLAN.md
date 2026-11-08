---
PLAN: "refactor!: patient_directory on the closed crud contract — server tenant, transactional batch, single declarative validation, view.go browser-only"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> Never edit `model_orm.go` or `model_crud.go`: a build error there means the hand-written code is wrong.

# Plan: `patient_directory` — template module on `crud.Registry`

Context: master `webtyp/docs/ARCHITECTURE_MASTER_PLAN.md` D2–D16 (phases A4.1 + L5). Runs after the view (L1), appkit (L2), layout (L3) and app (L4) plans of that master. Today `ops.go` registers five raw
operations; three read `args.TenantId` from the JSON (`opGetPatient`, `opFindPatientByRut`, `opDeactivatePatient`, lines 77-117):
any signed-in user can read or deactivate another clinic's patients. `createPatient`, `updatePatient` and `ImportPatient`
repeat by hand rules already in `PatientModel` (`Name`, `Sex`, `Id`). After this plan the module mounts three operations
through generated helpers, the tenant always comes from `Deps`, and validation of the schema lives in one place.

## Tools and local dependencies

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/ormc/cmd/ormc@v0.1.36
```
```bash
go get webtyp.com/model@v0.2.14 webtyp.com/router@v0.4.19 webtyp.com/crud@v0.0.3 webtyp.com/rpc@v0.2.17 webtyp.com/mcp@v0.3.23 \
  webtyp.com/dbsync@v0.3.47 webtyp.com/view@latest webtyp.com/appkit@latest
```
`view@latest` and `appkit@latest` must contain `view.Demo` and `appkit` without `layout` (plans L1, L2); if `go doc webtyp.com/view Demo`
fails, stop and say so in the PR. Remove every `replace` from `go.mod`; `go mod tidy`; run `ormc` (writes `model_crud.go`).
After the delivery check, `go.mod` does not list `webtyp.com/layout`.

## APIs used

| Symbol | Signature |
|---|---|
| generated `MountPatientRead` | `func MountPatientRead(reg crud.Registry, newFilter func() model.Model, read func(c crud.Caller, args crud.ReadArgs, filter model.Model) (PatientList, error)) crud.Route` |
| generated `MountPatientCreate` | `func MountPatientCreate(reg crud.Registry, create func(c crud.Caller, list PatientList) error) crud.Route` |
| generated `MountPatientUpdate` | `func MountPatientUpdate(reg crud.Registry, update func(c crud.Caller, ids, fields []string, rec Patient) error) crud.Route` |
| `crud.ReadArgs` | `struct{ ID, Search, After string; Limit int64; Filter model.Model }` — limit already capped, oversized args already rejected |
| `crud.Loopback` / `crud.LoopbackActingAs` | `func Loopback(mods ...crud.Module) router.Caller` / `func LoopbackActingAs(userID string, mods ...crud.Module) router.Caller` |
| `crud.Op` | `func Op(m model.ModuleNaming, action model.Action) string` → `patient_r`, `patient_c`, `patient_u` |
| `model.ValidateFields` | `func ValidateFields(action byte, f Fielder) error` |
| `orm.DB.Tx` | `func (d *DB) Tx(fn func(tx *DB) error) error` (orm v0.12.26 `tx.go:12`) |
| `orm.DB.UpdateFields` | `func (d *DB) UpdateFields(m model.Model, fields []string, cond storage.Condition, rest ...storage.Condition) error` |

If a signature differs, stop and name it.

## Explicit deletions (grep must be empty)

- `ops.go`: `OpListPatients`, `OpGetPatient`, `OpFindPatientByRut`, `OpUpsertPatient`, `OpDeactivatePatient`, every `op*` handler,
  `var _ router.OperationModule`, imports `webtyp.com/router`, `webtyp.com/rpc`, `webtyp.com/view`.
  `grep -rn "reg.Operation\|router.Context\|rpc\.\|args.TenantId" --include=*.go . | grep -v _test` → empty.
- `model.go`: `GetPatientArgsModel`, `FindPatientByRutArgsModel`, `DeactivatePatientArgsModel` (and their generated types after `ormc`);
  errors `ErrNameRequired`, `ErrTenantRequired`, `ErrIdRequired`, `ErrSex`, `ErrCannotRead` (the schema and crud cover them).
  `grep -rn "ErrNameRequired\|ErrTenantRequired\|ErrIdRequired\|ErrSex\b\|ErrCannotRead\|GetPatientArgs\|FindPatientByRutArgs\|DeactivatePatientArgs" .` → empty.
- `module.go`: `deactivatePatient` (becomes part of `UpdatePatients`), `listPatients`/`getPatient` move to `queries.go` (renamed, below).

## Stage 1 — `model.go` and `errors.go`

- `tenant_id` and `updated_at` get `ServerOnly: true` (the client may read them, never write them).
- Move the remaining `Err*` constants to a new `errors.go` (`ErrNotFound`, `ErrRutAlreadyExists`, `ErrRutRequired`, `ErrIdTaken`,
  `ErrDocumentType`, `ErrDocumentNumber`), Spanish comments with a usage example each. Run `ormc`.

## Stage 2 — `crud.go` (new: writes) and `queries.go` (new: reads); `module.go` keeps `Deps`, `Module`, `New`, `checkDocument`, `findSameDocument`, `publish`

Every function that touches the database takes the `*orm.DB` it must use, so the same code runs inside or outside a transaction.

```go
// crud.go

// CreatePatients guarda un lote de pacientes en una sola transacción. El tenant lo pone
// siempre el servidor. Repetir la misma creación devuelve sin error (idempotente por id).
//
// Ejemplo: err := dir.CreatePatients(crud.Caller{UserID: "u1"}, PatientList{&Patient{Id: "1", Name: "Ana", DocumentType: kinds.DocumentRUT, Rut: "12345678-5"}})
func (m *Module) CreatePatients(_ crud.Caller, list PatientList) error {
	created := make(PatientList, 0, len(list))
	err := m.db.Tx(func(tx *orm.DB) error {
		for _, p := range list {
			p.TenantId = m.tenantID
			out, isNew, err := m.createPatient(tx, *p)
			if err != nil {
				return err
			}
			*p = out
			if isNew {
				created = append(created, p)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, p := range created {
		m.publish(TopicPatientCreated, p)
	}
	return nil
}

// createPatient valida y guarda un paciente; isNew es false cuando el mismo id ya estaba
// guardado en este tenant (reintento). No publica: quien llama publica tras confirmar.
func (m *Module) createPatient(db *orm.DB, p Patient) (out Patient, isNew bool, err error) {
	if err := model.ValidateFields(model.ActionCreate, &p); err != nil {
		return Patient{}, false, err
	}
	if err := m.checkDocument(&p); err != nil {
		return Patient{}, false, err
	}
	var existing Patient
	_, errID := ReadOnePatient(db.Query(&existing).Where(Patient_.Id).Eq(p.Id), &existing)
	switch {
	case errID == nil && existing.TenantId == p.TenantId:
		return existing, false, nil
	case errID == nil:
		return Patient{}, false, ErrIdTaken
	case !orm.IsNotFound(errID):
		return Patient{}, false, errID
	}
	if err := m.findSameDocument(db, p); err != nil {
		return Patient{}, false, err
	}
	p.UpdatedAt = time.NowNano()
	if err := db.Create(&p); err != nil {
		return Patient{}, false, err
	}
	return p, true, nil
}

// UpdatePatients cambia los campos indicados de cada paciente del tenant, en una transacción.
// Desactivar es fields = ["is_active"] con IsActive = false (un paciente nunca se elimina).
//
// Ejemplo: err := dir.UpdatePatients(c, []string{"1"}, []string{"is_active"}, Patient{IsActive: false})
func (m *Module) UpdatePatients(_ crud.Caller, ids, fields []string, rec Patient) error {
	var events []patientEvent
	err := m.db.Tx(func(tx *orm.DB) error {
		for _, id := range ids {
			curr, err := m.readPatient(tx, id)
			if err != nil {
				return err
			}
			next := curr
			applyFields(&next, rec, fields)
			if touches(fields, Patient_.Rut, Patient_.DocumentType, Patient_.DocumentCountry) {
				if err := m.checkDocument(&next); err != nil {
					return err
				}
				if err := m.findSameDocument(tx, next); err != nil {
					return err
				}
			}
			next.UpdatedAt = time.NowNano()
			cols := append(append([]string{}, fields...), Patient_.UpdatedAt)
			if err := tx.UpdateFields(&next, cols, storage.Eq(Patient_.Id, id), storage.Eq(Patient_.TenantId, m.tenantID)); err != nil {
				return err
			}
			topic := TopicPatientUpdated
			if curr.IsActive && !next.IsActive {
				topic = TopicPatientDeactivated
			}
			events = append(events, patientEvent{topic: topic, p: next})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := range events {
		m.publish(events[i].topic, &events[i].p)
	}
	return nil
}

type patientEvent struct {
	topic string
	p     Patient
}
```
`applyFields(dst *Patient, src Patient, fields []string)` copies, for each name in `fields`, the matching pointer of
`src.Pointers()` into `dst.Pointers()` by index in `dst.Schema()` (no reflection, no map; a `switch` on the pointer type
`*string`, `*int64`, `*bool`, `*time.Nano`, `*time.Unix` — check the exact types in `model_orm.go`). `touches(fields []string, names ...string) bool`.
`Patient_.UpdatedAt` etc. are the generated column-name constants (check their exact names in `model_orm.go`).
crud already rejected unknown, PK, `tenant_id` and `ServerOnly` fields, so `fields` is safe.

```go
// queries.go

// ReadPatients lee pacientes del tenant: uno por id (args.ID) o una página buscada por
// RUT, teléfono o nombre (args.Search, args.After, args.Limit ya acotado por crud).
//
// Ejemplo: list, err := dir.ReadPatients(c, crud.ReadArgs{Search: "12.345", Limit: 20}, nil)
func (m *Module) ReadPatients(_ crud.Caller, args crud.ReadArgs, _ model.Model) (PatientList, error) {
	if args.ID != "" {
		p, err := m.readPatient(m.db, args.ID)
		if err != nil {
			return nil, err
		}
		return PatientList{&p}, nil
	}
	rows, err := m.searchPatients(m.db, args.Search, args.After, args.Limit)
	if err != nil {
		return nil, err
	}
	list := make(PatientList, 0, len(rows))
	for i := range rows {
		list = append(list, &rows[i])
	}
	return list, nil
}
```
`readPatient(db *orm.DB, id string) (Patient, error)` = today's `getPatient` body with `m.tenantID` (never a parameter);
`searchPatients(db, search, after string, limit int64)` = today's `listPatients` body with `m.tenantID`; `findByDocument` stays
for `readers.go` and also takes `db`. `findSameDocument(db *orm.DB, p Patient)` uses `db` instead of `m.db`.
`ImportPatient` (`import.go`): delete its hand-written `Id`/`TenantId`/`Name`/`Sex` checks; it starts with
`model.ValidateFields(model.ActionCreate, &p)` and keeps `checkDocument` + `findSameDocument(m.db, p)` + its upsert.
`readers.go` (`CreatePatient`) calls `createPatient(m.db, …)` and publishes `TopicPatientCreated` when `isNew`.

## Stage 3 — `ops.go`

```go
package patientdirectory

import "webtyp.com/crud"

// ModelName es la identidad del módulo: las operaciones viajan como "patient_directory.patient_r|c|u".
const ModelName = "patient_directory"

func (m *Module) ModelName() string { return ModelName }

var _ crud.Module = (*Module)(nil)

// MountOperations monta leer, crear y actualizar pacientes. No hay eliminar: un paciente se desactiva.
func (m *Module) MountOperations(reg crud.Registry) {
	MountPatientRead(reg, nil, m.ReadPatients)
	MountPatientCreate(reg, m.CreatePatients)
	MountPatientUpdate(reg, m.UpdatePatients)
}
```

## Stage 4 — `view.go`, demo and module glue

- `view.go`: first line `//go:build browser`; `view.Ops{Module: ModelName, Allow: model.Create | model.Update, Page: view.Page{First: 0, Size: 20, More: true}}`
  (no `List`/`Save` strings). Keep the Spanish comment: no delete because a patient is deactivated.
- `module/browser.go` and `module/server.go`: `appkit.ModuleBrowser` → `view.ModuleBrowser`, `appkit.BrowserDeps` → `view.BrowserDeps`
  (`appkit.ModuleServer` stays). The module imports only contracts (`view`, `crud`, `appkit` for `ModuleServer`), never `webtyp.com/layout`.
- **The demo becomes `seed/demo.go` (D15)** and `web/browser.go` is deleted (`test ! -e web/browser.go`; remove `web/` if empty). Today's
  `web/browser.go` shows only a title («Demo render sin webtyp.com/layout»); the daemon (`webtyp/app`) now draws the full menu from:

```go
//go:build wasm

package seed

// Demo es la demo ejecutable del módulo: base en memoria, datos de ejemplo y sus pantallas.
// El demonio de webtyp la dibuja con el chasis; este módulo nunca importa webtyp.com/layout.
//
// Ejemplo: abrir la raíz del repo con webtyp y elegir «Pacientes».
func Demo() (view.Demo, error) {
	ids, err := unixid.NewUnixID()
	if err != nil {
		return view.Demo{}, err
	}
	db := orm.New(mem.New())
	mod, err := patientdirectory.New(db, patientdirectory.Deps{
		IDs: ids, Publisher: &mock.Broker{}, TenantID: demoTenantID, ValidateRUT: trustedip.ValidateRUT,
	})
	if err != nil {
		return view.Demo{}, err
	}
	if _, err := Load(mod, demoTenantID); err != nil {
		return view.Demo{}, err
	}
	return view.Demo{
		Label: ui.DefaultLabel, Timezone: "America/Santiago", ActingAs: "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: patientdirectory.ModelName, Run: migrate.Migrate}},
		Screens:    func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) },
	}, nil
}

const demoTenantID = "demo"
```
  Use the exact `patientdirectory.Deps` fields that exist after stage 2 and the exact `ui` label constant (`grep -n "Label" ui/module.go`);
  `migrate` is `github.com/veltylabs/patient_directory/migrate`. Do not run migrations here if today's demo did not.
- After `go mod tidy`: `grep -rn "webtyp.com/layout\|appkit.BrowserDeps\|appkit.ModuleBrowser\|router/loopback" .` → empty, `go.mod` included.
- `module/server.go`: `Build` still returns the `*Module` (a `crud.Module`); nothing else changes there.

## Stage 5 — tests (`tests/`, through `crud.LoopbackActingAs("u1", mod)`; operation names from `crud.Op(&patientdirectory.Patient{}, …)` qualified with `patientdirectory.ModelName + "."`)

Migrate `ops_test.go`, `patient_list_search_test.go`, `patient_test.go`, `screen_save_test.go`, `tenant_fallback_test.go`, `tenant_test.go`
to `crud.CreateArgs`/`ReadArgs`/`UpdateArgs`. Delete tests of removed behaviour (client-supplied tenant fallback). Add `TestDemo_Valid` (`tests/demo_test.go`, `//go:build wasm`): `seed.Demo()` → no error and `Validate()` → `nil`; its `Screens` returns at least one screen. Required tests
(setup → call → expected; rows in `docs/TESTS.md`):

| Test (file) | Setup | Call | Expected |
|---|---|---|---|
| `TestTenant_ReadOtherTenantByIDIsNotFound` (`tenant_test.go`) | module A (tenant `a`) creates patient `p1`; module B (tenant `b`) on the same db | B: `patient_r` with `{id:"p1"}` | error kind `model.ErrorNotFound` |
| `TestTenant_UpdateOtherTenantIsNotFound` | same | B: `patient_u` ids `["p1"]`, fields `["is_active"]` | `model.ErrorNotFound`; A still reads `p1` active |
| `TestTenant_ClientTenantRejected` | module A | `patient_c` with a record carrying `tenant_id:"b"` | error text `crud: field written only by the server`; nothing stored |
| `TestCreate_BatchIsAtomic` (`ops_test.go`) | — | `patient_c` with 2 records, the second with an invalid RUT | error on `rut`; reading returns 0 patients |
| `TestCreate_RetryIsIdempotent` | — | `patient_c` with `p1` twice | second call succeeds; one row; one `TopicPatientCreated` |
| `TestUpdate_DeactivatePublishesDeactivated` | `p1` active | `patient_u` `["is_active"]`, `IsActive:false` | `TopicPatientDeactivated` published once; `TopicPatientUpdated` not |
| `TestValidation_NameFromSchema` | — | `patient_c` record without `name` | field error on `name` (from the schema, not a module constant) |
| `TestValidation_SexFromKind` | — | record with `sex:"x"` | field error on `sex` |
| `TestImport_ValidatesWithSchema` (`import_test.go`) | — | `ImportPatient(Patient{Id:"1", TenantId:"a"})` (no name) | field error on `name` |

## Stage 6 — translations and documentation (Spanish, veltylabs)

`langc sync .`: zero untranslated keys, zero reuse warnings. `README.md` (≤150 lines, 4 mandatory sections: `Por qué existe` with
a Sin/Con comparison — `reg.Operation` + `args.TenantId` vs `MountPatientRead/Create/Update` —, `Antes de empezar`, `Guía paso a paso`
with a complete compilable example: `New`, mounting in an app, a test with `crud.LoopbackActingAs`, `Mapa de documentación`).
Operations table: `patient_r` (lee uno o busca), `patient_c` (crea en lote), `patient_u` (actualiza o desactiva).
`docs/ARCHITECTURE.md` (4 sections) and `docs/diagrams/database.md` refreshed. Translate to Spanish every README, `docs/*` file,
diagram and code comment still in English; exported symbols without a usage example get one.

## Delivery check

`gotest` passes; `langc sync .` clean; every grep above empty. A tree that does not compile or whose tests did not run is a failed delivery.

| Stage | Files |
|---|---|
| 1 | `model.go`, `errors.go`, `model_orm.go`/`model_crud.go` (generated) |
| 2 | `crud.go`, `queries.go`, `module.go`, `import.go`, `readers.go` |
| 3 | `ops.go` |
| 4 | `view.go`, `module/*.go`, `seed/demo.go`, delete `web/browser.go`, `go.mod` |
| 5 | `tests/*.go`, `docs/TESTS.md` |
| 6 | `lang.json`, `README.md`, `docs/*` |
