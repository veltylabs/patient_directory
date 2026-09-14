---
PLAN: "feat: tenant-scoped patient registry with the ClientExists port"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 8885645866320096792
PR: https://github.com/veltylabs/patient_directory/pull/1
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# Plan — build the `patient_directory` module

You are an agent with **no prior context** and you have **only this repository**
(`github.com/veltylabs/patient_directory`). It currently contains a `gonew`
skeleton: `go.mod`, `LICENSE`, `README.md`, `.gitignore`, and a placeholder
`patient_directory.go` holding an empty `PatientDirectory` struct. **Delete
that placeholder file in stage 1** — nothing keeps its name or its type.

Everything you need is inline. Sibling repositories are referenced by GitHub
URL for optional reading only; every rule and every example you must follow is
reproduced here.

## 1. Why this module exists

A clinic application in this ecosystem books appointments with
`github.com/veltylabs/appointment_booking`. That module refuses to create a
reservation for a client it cannot verify, and declares the question as a
narrow port it does not import:

```go
// appointment_booking/service.go
// DirectoryReader verifica que un cliente existe y pertenece al tenant.
type DirectoryReader interface {
	ClientExists(tenantId, clientId string) (bool, error)
}
```

**Nothing in the ecosystem implements it.** A demo application stubs it to
always return `true`, and the one module that touches patients —
`github.com/veltylabs/clinical_encounter` — stores `patient_id` as free text
plus `patient_name_snapshot` / `patient_rut_snapshot` columns typed at the point
of use. There is no row anywhere that says "this person exists", so nothing
stops the same patient existing three times under three spellings, and nothing
gives a booking screen a person to book for.

This module is that missing row: the tenant-scoped registry of people the
establishment serves, and the implementation of `ClientExists`.

**Scope boundary — read this before writing any field.** This module owns
*identity and contact*: who a person is and how to reach them. It owns **no
clinical data**. Diagnoses, visits, prescriptions and history belong to
`clinical_encounter`; appointments belong to `appointment_booking`. A field
that is only meaningful to a doctor does not go here.

## 2. Design gate

### Prior art

- **FHIR `Patient` resource** (HL7) separates the demographic/administrative
  record — identifiers, name, birth date, contact, active flag — from every
  clinical resource that references it. That separation is exactly the scope
  boundary above, and it is why this is its own module rather than a table in
  `clinical_encounter`.
- **Django** `Model.objects.filter(...).exists()` and **Rails**
  `Model.exists?(id)` expose existence as a first-class boolean query, separate
  from fetching the row, so a caller that needs only the answer neither pays for
  nor handles a full record. **Ent** (Go) generates `Query().Exist(ctx)` for the
  same reason. `ClientExists` returns `(bool, error)` to match.
- **OpenEMR / OpenMRS** both keep a patient index keyed by a national
  identifier, with a lookup-by-identifier path distinct from lookup-by-id,
  because reception staff hold the identifier, never the internal id. That is
  why `find_patient_by_rut` exists alongside `get_patient`.

**Where we differ.** FHIR allows many identifiers per patient; we take exactly
one, the RUT, because a Chilean clinic has one national identifier and a
general identifier table would be unused generality. And unlike all of the
above, this module never renders human language — labels are the consuming
app's job.

### Novice-name test

- "Does this client exist in this tenant?" → `ClientExists(tenantId, clientId)`.
  The name is fixed by the port `appointment_booking` declares; choosing another
  would force every consumer to write an adapter.
- "Find the patient with this RUT" → `FindByRut(tenantId, rut)`.
- "The patient registry" → package `patientdirectory`, type `Patient`.
- "Deactivate this patient" → `DeactivatePatient(tenantId, id)`.

### Complexity ledger

| | change |
|---|---|
| Concepts | `+1` — "a patient is a record that exists independently of any visit". It is the concept the free-text `patient_id` was already an unnamed, unenforced instance of |
| Files | `+9` in a new repository |
| Call-site lines | `−1 stub per application` that currently fakes `DirectoryReader`; `clinical_encounter`'s hand-typed snapshots gain a source |
| Ways to do it | `−2` — "who is this patient" had two unsupported answers (free text on the clinical record, and a stub returning `true`) and zero supported ones. Now it has one |
| Net | negative |

### Where it belongs

Its own module. Not in `clinical_encounter`, because that would make
`appointment_booking` — which has **zero** `veltylabs` dependencies on purpose —
depend on a clinical module to book an appointment, and would give
`clinical_encounter` two reasons to change (identity and clinical record). Not
in `staff_manager`, which registers the people who *work* at the establishment,
a disjoint set governed by authentication concerns.

### What it deletes

Nothing yet in any repository; this module is new. What it makes deletable is
the `DirectoryReader` stub in every application, and eventually the
unsourced snapshot columns in `clinical_encounter` — both in those repos, not
here.

## 3. Decisions already taken — do not revisit

1. **The RUT is the identity, and it is unique per tenant.** Two rows with the
   same `(tenant_id, rut)` are the defect this module exists to prevent.
2. **Uniqueness is enforced at the application layer, not by a DB constraint.**
   `model.FieldDB` exposes only a single-column `Unique bool`, and a
   single-column unique on `rut` would wrongly collide across tenants. The
   sibling `device_manager` enforces its own IP uniqueness the same way, with an
   `ErrIPAlreadyExists` sentinel and a lookup before insert. Follow that
   pattern; do **not** set `Unique: true` on `rut`.
3. **There is no delete op — only deactivate.** A patient referenced by a
   medical record or a past reservation must never disappear; those references
   are by id and would dangle. `is_active` is how a patient leaves the working
   list.
4. **RUT validation is injected, never imported.** `Deps.ValidateRUT
   func(string) (string, error)` normalises and validates, exactly as
   `staff_manager` does. This module must not depend on any particular RUT
   library, and must not ship its own check-digit implementation.
5. **No clinical fields.** See the scope boundary in §1. If a field only makes
   sense to a doctor, it is not in this plan.
6. **No human language.** No Spanish label, no status word, no `switch` mapping
   a code to a phrase. The consuming app translates.

## 4. Stages

### Stage 1 — repository skeleton

1. **Delete `patient_directory.go`** (the `gonew` placeholder). Its type
   `PatientDirectory` and its `New()` are replaced by `Module` and
   `New(db, deps)` in stage 4.
2. `go.mod` already declares `module github.com/veltylabs/patient_directory`
   and `go 1.25.2`. Add the dependency set — the same one `device_manager`
   uses, which is the closest sibling in shape:

   ```
   webtyp.com/ddl
   webtyp.com/events
   webtyp.com/fmt
   webtyp.com/input
   webtyp.com/model
   webtyp.com/orm
   webtyp.com/router
   webtyp.com/storage
   webtyp.com/time
   webtyp.com/view
   ```

   Resolve each to its latest published version with `go get`. Do **not** add a
   `replace` directive, and do **not** add any `github.com/veltylabs/*`
   dependency — this module depends on no sibling module.
3. Create `AGENTS.md` at the root recording decisions 3.1–3.6 above, so they
   outlive this plan file.

**Anti-footgun — no standard library.** Use `webtyp.com/fmt` for errors and
string work, `webtyp.com/time` for time. Never `errors`, `strconv`, `strings`,
`time`, or `encoding/json`. This package is compiled into a WASM client through
its `view.go`.

### Stage 2 — `model.go`

Package `patientdirectory`. Model it on `device_manager/model.go`, which is the
house shape.

```go
// Helper vars satisfy ormc's AST static type parser when the same Kind
// constructor is reused across more than one Definition — the same defensive
// pattern device_manager and item_catalog use.
var (
	BaseBool_FieldBool = model.Bool()
	BaseInt_FieldInt   = model.Int()
)

// PatientModel is the establishment's registry of the people it serves:
// IDENTITY and CONTACT only. Clinical data — diagnoses, visits, prescriptions —
// belongs to clinical_encounter and must never be added here.
//
// rut carries NO Unique flag on purpose: model.FieldDB's Unique is
// single-column, and a RUT is unique per TENANT, not globally. The constraint
// is enforced in CreatePatient against ErrRutAlreadyExists — the same
// application-layer rule device_manager applies to a device IP.
var PatientModel = model.Definition{
	Name: "patient",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "rut", Type: input.Rut(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 12}},
		{Name: "name", Type: input.Text(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}},
		{Name: "birthdate", Type: input.Date(), OmitEmpty: true},
		{Name: "phone", Type: input.Phone(), OmitEmpty: true},
		{Name: "email", Type: input.Email(), OmitEmpty: true},
		{Name: "address", Type: input.Address(), OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), NotNull: true},
		{Name: "updated_at", Type: BaseInt_FieldInt, OmitEmpty: true},
	},
}
```

`tenant_id` is `model.Text()`, not `input.Text()`: it is machine-supplied and
must never render as a form input. Every other user-facing field carries an
`input.*` widget. This is the widget-policy-by-role rule the sibling modules
state explicitly — a base kind on machine-supplied and output-only fields, an
`input.*` on what a person types.

Transport-only Definitions (every field `DB: nil`):

```go
var ListPatientsArgsModel = model.Definition{
	Name: "list_patients_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "active_only", Type: BaseBool_FieldBool},
		{Name: "limit", Type: BaseInt_FieldInt},
		{Name: "offset", Type: BaseInt_FieldInt},
	},
}

var GetPatientArgsModel = model.Definition{
	Name: "get_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var FindPatientByRutArgsModel = model.Definition{
	Name: "find_patient_by_rut_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "rut", Type: input.Rut()},
	},
}

var DeactivatePatientArgsModel = model.Definition{
	Name: "deactivate_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}
```

Sentinels and topics:

```go
var (
	ErrNotFound         = fmt.Err("patient not found")
	ErrRutAlreadyExists = fmt.Err("patient rut already exists in this tenant")
	ErrRutRequired      = fmt.Err("patient rut is required")
	ErrNameRequired     = fmt.Err("patient name is required")
	ErrTenantRequired   = fmt.Err("patient tenant_id is required")
)

type ValidationError struct{ Err error }

func (v ValidationError) Error() string { return v.Err.Error() }

// <module>.<entity>.<past-tense-verb> — tenant_id goes in the payload, never
// in the topic name.
const (
	TopicPatientCreated     = "patient_directory.patient.created"
	TopicPatientUpdated     = "patient_directory.patient.updated"
	TopicPatientDeactivated = "patient_directory.patient.deactivated"
)
```

### Stage 3 — `model_orm.go`

Generate with `ormc`. **Never hand-edit it.** It produces `Patient`,
`PatientList`, `Patient_`, the args structs and their list twins.

### Stage 4 — `module.go`

```go
type Deps struct {
	IDs       model.IDGenerator // required — the module never builds its own
	Publisher events.Publisher  // optional — nil disables publishing silently
	// ValidateRUT normalises and validates a RUT, returning the canonical form.
	// Required. Injected, never imported: this module must not depend on any
	// particular RUT implementation, and must not ship its own check digit.
	ValidateRUT func(string) (string, error)
}

type Module struct {
	db          *orm.DB
	ids         model.IDGenerator
	pub         events.Publisher
	validateRUT func(string) (string, error)
}

// New connects the module to an already-connected *orm.DB; the schema is
// assumed to already exist — see the migrate subpackage.
func New(db *orm.DB, deps Deps) (*Module, error)
```

`New` returns an error when `deps.IDs` is nil (`"patient_directory: Deps.IDs is
required"`) or `deps.ValidateRUT` is nil (`"patient_directory:
Deps.ValidateRUT is required"`). It performs **no DDL** — see stage 7.

Methods:

| Method | Behaviour |
|---|---|
| `CreatePatient(p Patient) (Patient, error)` | validates; normalises the RUT through `validateRUT`; rejects a duplicate `(tenant_id, rut)` with `ErrRutAlreadyExists`; assigns `Id` from `ids.NewID()` when empty; sets `UpdatedAt = time.Now()`; publishes `TopicPatientCreated` |
| `UpdatePatient(p Patient) (Patient, error)` | requires `Id`; `ErrNotFound` when absent; normalises the RUT; rejects a RUT that belongs to a **different** id in the same tenant with `ErrRutAlreadyExists`; sets `UpdatedAt`; publishes `TopicPatientUpdated` |
| `GetPatient(tenantID, id string) (Patient, error)` | `orm.ErrNotFound` → `ErrNotFound`; any other error surfaces unchanged |
| `FindByRut(tenantID, rut string) (Patient, error)` | normalises the RUT first, then looks up; `ErrNotFound` when absent |
| `ListPatients(tenantID string, filter PatientFilter) ([]Patient, error)` | tenant-scoped; honours `ActiveOnly`, `Limit`, `Offset` |
| `DeactivatePatient(tenantID, id string) error` | sets `is_active = false`; publishes `TopicPatientDeactivated` |
| `ClientExists(tenantID, clientID string) (bool, error)` | see below |

```go
// PatientFilter narrows ListPatients — a plain internal type, never
// ormc-generated (only its transport twin ListPatientsArgs is).
type PatientFilter struct {
	ActiveOnly bool
	Limit      int64
	Offset     int64
}
```

```go
// ClientExists reports whether a patient with this id belongs to this tenant.
// It satisfies the narrow DirectoryReader port that a scheduling module
// declares on its own side (ClientExists(tenantId, clientId) (bool, error)) —
// structurally, with no adapter and no import in either direction.
//
// A missing row is (false, nil), NOT an error: "this id is not one of ours" is
// the answer the caller asked for. Only a real storage failure returns a
// non-nil error, so a caller can never mistake a dead database for a clean
// "no".
//
// An INACTIVE patient still exists. Deactivation removes someone from the
// working list; it does not unmake the person, and a reservation that
// references them must keep resolving.
func (m *Module) ClientExists(tenantID, clientID string) (bool, error)
```

**Anti-footgun — do not filter `ClientExists` by `is_active`.** The comment
above is the contract; a deactivated patient's past and pending reservations
must keep resolving. Reuse `GetPatient` and map `ErrNotFound` to `(false, nil)`
rather than writing a second copy of the not-found mapping.

**Anti-footgun — never hide a storage failure as "not found".** Only
`orm.ErrNotFound` maps to the domain sentinel; every other error surfaces as
the internal error it is. This rule is written into `device_manager`'s
`GetDevice` for the same reason.

A `publish` helper funnels every event so the `nil`-publisher check lives in
one place.

### Stage 5 — `ops.go`

```go
const (
	OpListPatients       = "list_patients"
	OpGetPatient         = "get_patient"
	OpFindPatientByRut   = "find_patient_by_rut"
	OpUpsertPatient      = "upsert_patient"
	OpDeactivatePatient  = "deactivate_patient"
)

func (m *Module) ModelName() string { return "patient_directory" }

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListPatients, m.opListPatients).Requires("patient", model.Read).Accepts(&ListPatientsArgs{})
	reg.Operation(OpGetPatient, m.opGetPatient).Requires("patient", model.Read).Accepts(&GetPatientArgs{})
	reg.Operation(OpFindPatientByRut, m.opFindPatientByRut).Requires("patient", model.Read).Accepts(&FindPatientByRutArgs{})
	reg.Operation(OpUpsertPatient, m.opUpsertPatient).Requires("patient", model.Create|model.Update).Accepts(&Patient{})
	reg.Operation(OpDeactivatePatient, m.opDeactivatePatient).Requires("patient", model.Update).Accepts(&DeactivatePatientArgs{})
}

var _ router.OperationModule = (*Module)(nil)
```

**Every op declares a `Resource` and an `Action`.** There is no op without a
policy — a consuming application denies by default and grants explicitly, so an
op that declares nothing is unreachable at best and a hole at worst.

**There is no `delete_patient` op** (decision 3.3). Do not add one.

Handler status codes, following `device_manager/ops.go` exactly:

| Condition | Status |
|---|---|
| `ctx.Decode` fails | `400` |
| `ValidationError` | `400` |
| `ErrRutAlreadyExists` | `409` |
| `ErrNotFound` | `404` |
| anything else | `500` |
| success with no body | `200` |

`opUpsertPatient` branches on `p.Id == ""` → `CreatePatient`, else
`UpdatePatient` — the same shape as `opUpsertDevice`.

### Stage 6 — `view.go`

```go
// Item implements view.Itemizer. Description is the RUT because that is what a
// person at the counter is holding and reads back to confirm they picked the
// right patient.
func (p *Patient) Item() view.Item {
	return view.Item{ID: p.Id, Label: p.Name, Description: p.Rut}
}

const titlePatients = "Patients"

// NewView builds the patient Presenter — the renderer-agnostic engine a
// renderer (crudview, or any other) wraps. This module builds it (view + model
// + router only); the app decides which renderer draws it.
//
// Ops carries NO Delete: view.NewCallerLister implements exactly the write
// capabilities whose op names are non-empty, so a renderer paints no delete
// control. A patient is deactivated, never deleted (see AGENTS.md).
func NewView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{List: OpListPatients, Save: OpUpsertPatient},
		func() model.ModelSlice { return &PatientList{} })
	return view.New(b, &Patient{}, view.WithTitle(titlePatients))
}
```

`titlePatients` is English: this module ships no Spanish. The consuming app
translates through `webtyp.com/fmt/lang`.

### Stage 7 — `migrate/migrate.go`

Separate package `migrate`, one exported function:

```go
package migrate

import (
	"webtyp.com/ddl"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// Migrate reconciles the database schema patient_directory owns: Patient.
//
// It is deliberately NOT called by New, and deliberately lives in its own
// package rather than a file in the root package: nothing on a consuming app's
// WASM build path (its view.go, which imports the root package for NewView)
// ever imports ".../patient_directory/migrate" — so webtyp.com/ddl never
// enters that build graph, regardless of build tags on the consumer's side.
//
// Sync, not CreateTable: CreateTable compiles to CREATE TABLE IF NOT EXISTS and
// is a no-op against a table that already exists, so a column added to the
// model later would never reach a deployed database. Sync creates the table
// when it is absent and adds the missing columns when it is not — additive
// only, never dropping or narrowing, so it is safe to run repeatedly.
//
// conn is a ddl.Execer, not an *orm.DB, so a deploy-time transport that can
// only execute DDL satisfies it:
//
//	conn, _ := postgres.Open(dsn)
//	compiler, _ := conn.(ddl.Compiler)
//	err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).Sync(&patientdirectory.Patient{})
}
```

**Anti-footgun.** Some sibling modules' `migrate` packages still call
`CreateTable`. You have only this repository; do not treat a remembered
`CreateTable` example as the pattern. `Sync` is what this module uses.

Also add `migrate/migrate_test.go` (`package migrate_test`) with a
`dummyExecer` recording queries and a `dummyCompiler` returning `stmt.Table`,
asserting `Migrate` runs without error and touches the `patient` table.

### Stage 8 — `tests/`

**Every test lives in `tests/`**, except `migrate/migrate_test.go`, which is the
`migrate` package's own unit test.

`tests/setup_test.go` — builds a `*Module` over `webtyp.com/storage/mem` with a
deterministic fake `model.IDGenerator`, a recording fake `events.Publisher`, and
a fake `ValidateRUT` that uppercases and strips dots/dashes. **Do not use a real
RUT library**; the fake is the point of the injection.

`tests/patient_test.go`:

| Test | Asserts |
|---|---|
| `TestCreatePatient_AssignsIDAndNormalisesRut` | id assigned, RUT stored in canonical form, `TopicPatientCreated` published |
| `TestCreatePatient_DuplicateRutSameTenant` | `ErrRutAlreadyExists` |
| `TestCreatePatient_SameRutDifferentTenant` | **succeeds** — the RUT is unique per tenant, not globally |
| `TestCreatePatient_MissingRutOrName` | `ValidationError` |
| `TestUpdatePatient_RutTakenByAnother` | `ErrRutAlreadyExists` |
| `TestUpdatePatient_KeepingOwnRut` | succeeds — a patient may be saved without changing their RUT |
| `TestFindByRut_NormalisesInput` | an un-normalised RUT finds the stored patient |
| `TestDeactivatePatient` | `is_active` false, `TopicPatientDeactivated` published |

`tests/client_exists_test.go`:

| Test | Asserts |
|---|---|
| `TestClientExists_True` | `(true, nil)` |
| `TestClientExists_UnknownID` | `(false, nil)` — **not** an error |
| `TestClientExists_WrongTenant` | `(false, nil)` |
| `TestClientExists_InactivePatientStillExists` | `(true, nil)` — decision 3.3's contract |
| `TestClientExists_EmptyArgs` | `(false, nil)` |

Plus the compile-time port assertion, in `tests/client_exists_test.go`:

```go
// The port appointment_booking declares on its own side. Redeclared locally on
// purpose: this repository must not depend on a scheduling module to prove it
// satisfies a structural interface.
type directoryReader interface {
	ClientExists(tenantId, clientId string) (bool, error)
}

var _ directoryReader = (*patientdirectory.Module)(nil)
```

`tests/tenant_test.go` — every read and write is tenant-scoped: a patient of
tenant A is invisible to `GetPatient`, `FindByRut` and `ListPatients` for
tenant B.

`tests/ops_test.go` — mount the ops on a `router/mock` or loopback registry and
assert the status-code table from stage 5, including `409` on a duplicate RUT
and the **absence** of a `delete_patient` op.

Run `gotest ./...` (never `go test`) — everything green.

### Stage 9 — documentation

- `README.md` — replace the two-line `gonew` stub with: what the module owns,
  the scope boundary from §1, the Ops table (op, resource, action, args), a
  Quick Start showing `migrate.Migrate` then `New`, a **"Ports this module
  satisfies"** section naming `ClientExists`, and a Key files table.
- `docs/ARCHITECTURE.md` — domain scope, the six decisions from §3 with their
  reasoning, and the composition-root example wiring this module into
  `appointment_booking`'s `Deps.Directory`.
- `AGENTS.md` — created in stage 1; verify it still matches what was built.
- Do **not** link any permanent document to `docs/PLAN.md` — it is deleted when
  this lands.

## 5. Stages table

| # | Stage | Files | Acceptance |
|---|---|---|---|
| 1 | Skeleton | `go.mod`, `AGENTS.md`; delete `patient_directory.go` | no `PatientDirectory` type anywhere |
| 2 | Definitions | `model.go` | `rut` has no `Unique`; `tenant_id` has no widget |
| 3 | Generated | `model_orm.go` (ormc) | `Patient`, `PatientList`, `Patient_` exist |
| 4 | Service | `module.go` | `New` does no DDL; `ClientExists` ignores `is_active` |
| 5 | Transport | `ops.go` | 5 ops, each with Resource+Action; no delete |
| 6 | Presenter | `view.go` | `view.Ops` carries no `Delete` |
| 7 | Migration | `migrate/migrate.go`, `migrate/migrate_test.go` | uses `Sync`, not `CreateTable` |
| 8 | Tests | `tests/*.go` | all cases green, port assertion compiles |
| 9 | Docs | `README.md`, `docs/ARCHITECTURE.md` | ports and scope documented |

## 6. Acceptance criteria

- `var _ directoryReader = (*patientdirectory.Module)(nil)` compiles.
- `ClientExists` returns `(false, nil)` — never an error — for an unknown id, a
  foreign tenant, and empty arguments; and `(true, nil)` for an inactive
  patient.
- `grep -rn "delete_patient\|DeletePatient" .` → **empty**.
- `grep -rn "CreateTable" migrate/` → **empty**; `Sync` is the only DDL call.
- `grep -rn "webtyp.com/ddl" *.go` at the repository root → **empty**.
- `grep -rn "github.com/veltylabs" go.mod` → **empty**. This module depends on
  no sibling module.
- `grep -rnE '"(errors|strconv|strings|encoding/json)"' .` → **empty**.
- `grep -rn "Unique" model.go` → **empty**.
- `grep -rn "PatientDirectory" .` → **empty** (the `gonew` placeholder is gone).
- No Spanish string anywhere in the repository outside `docs/`.
- `gotest ./...` green.
