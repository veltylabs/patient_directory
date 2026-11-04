---
PLAN: "feat!: six-verb plain API (GetPatient, FindPatient, CreatePatient); struct methods unexported; LinkPatient and ClientExists removed; module/"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 434415614690636480
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — patient_directory exports three plain methods any module can declare

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
go get webtyp.com/time@v0.6.4 webtyp.com/appkit@v0.2.8
```

## Why

A module cannot hand its `Patient` type to another module, so each consumer asked for its own plain variant with an
invented name: today this module has 6 names to read and create a patient (`ClientExists`, `FindByRut`,
`GetPatient`, `CreatePatient`, `LinkPatient` + the ones `appointment_booking`/`clinical_encounter` ask for), and an
app with those modules does not start. New ecosystem rule (module-layout «Six verbs, one name per job"): every
exported method is `<Verb><Entity>` with `Get`, `Find`, `List`, `Create`, `Update`, `Delete` (+ `Import<Entity>`),
plain values in and out; forms with the module's own type are unexported and used only by its ops.

## The contract (exported, exactly these signatures)

```go
GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error)
FindPatient(tenantID, documentNumber string) (id, name, phone, email string, found bool, err error)
CreatePatient(tenantID, id, documentNumber, name, phone, email string) (patientID string, err error)
```
Absent → `found == false` and `err == nil`; only a storage failure is an error. `CreatePatient` with a document
already registered → `ErrRutAlreadyExists` (`model.Conflict`).

Consumers (updated in parallel plans): `appointment_booking` declares `{GetPatient, FindPatient, CreatePatient}`,
`clinical_encounter` declares `{GetPatient}`.

## Design gate

1. Prior art: CRUD verbs (REST, Rails resources, Google AIP-131/133/134 Get/Create/Update) — the verbs everyone knows.
2. Names: Get/Find/Create + Patient read aloud without docs.
3. Ledger: exported methods 9 → 4 (`GetPatient`, `FindPatient`, `CreatePatient`, `ImportPatient`); ways to read a
   patient 4 → 2 (by id, by document).
4. Belongs here: owner of patients.
5. Deletes: `LinkPatient`, `opLinkPatient` + its registration, `LinkPatientArgsModel`, `LinkPatientResultModel`
   (`model.go:43,55`), `ClientExists`, `tests/link_patient_test.go`, `tests/client_exists_test.go`; exported
   `FindByRut`, `UpdatePatient`, `ListPatients`, `DeactivatePatient` become unexported.

## Stage 1 — unexport the struct forms (`module.go`)

| Today (exported) | Becomes |
|---|---|
| `GetPatient(tenantID, id) (Patient, error)` | `getPatient` |
| `CreatePatient(p Patient) (Patient, error)` | `createPatient` |
| `UpdatePatient(p Patient) (Patient, error)` | `updatePatient` |
| `ListPatients(tenantID, filter) ([]Patient, error)` | `listPatients` |
| `DeactivatePatient(tenantID, id) error` | `deactivatePatient` |
| `FindByRut(tenantID, rut) (Patient, error)` | `findByDocument` |

Rename every call (`ops.go`, `module.go`, `import.go`). `ImportPatient(p Patient) (Patient, error)` stays exported
(mjosefa-cms `migrate/pastdb` uses it). `seed/seed.go:52` `m.CreatePatient(items[i])` → `m.ImportPatient(items[i])`.

## Stage 2 — the three plain methods (`readers.go`, new)

```go
// GetPatient lee un paciente por su id. Ausente u otro tenant ⇒ found=false, err=nil.
// Ejemplo: name, doc, phone, email, ok, err := m.GetPatient("t1", "p1")
func (m *Module) GetPatient(tenantID, id string) (name, documentNumber, phone, email string, found bool, err error) {
	if tenantID == "" || id == "" {
		return "", "", "", "", false, nil
	}
	p, err := m.getPatient(tenantID, id)
	if model.KindOf(err) == model.ErrorNotFound {
		return "", "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", "", false, err
	}
	return p.Name, p.Rut, p.Phone, p.Email, true, nil
}

// FindPatient busca un paciente por su número de documento (RUT). Ausente ⇒ found=false, err=nil.
// Ejemplo: id, name, phone, email, ok, err := m.FindPatient("t1", "12.345.678-5")
func (m *Module) FindPatient(tenantID, documentNumber string) (id, name, phone, email string, found bool, err error) {
	p, err := m.findByDocument(tenantID, documentNumber)
	if model.KindOf(err) == model.ErrorNotFound {
		return "", "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", "", false, err
	}
	return p.Id, p.Name, p.Phone, p.Email, true, nil
}

// CreatePatient crea un paciente activo con RUT y devuelve su id. Documento ya registrado ⇒ ErrRutAlreadyExists.
// Ejemplo: id, err := m.CreatePatient("t1", newID, "12.345.678-5", "Ana", "+56911223344", "ana@correo.cl")
func (m *Module) CreatePatient(tenantID, id, documentNumber, name, phone, email string) (string, error) {
	p, err := m.createPatient(Patient{
		Id: id, TenantId: tenantID, DocumentType: kinds.DocumentRUT, Rut: documentNumber,
		Name: name, Phone: phone, Email: email, IsActive: true,
	})
	if err != nil {
		return "", err
	}
	return p.Id, nil
}
```

`findByDocument` returns `ErrNotFound` (`model.NotFound`) when absent, as `FindByRut` does today (`module.go:231`).

## Stage 3 — deletions

Delete everything in gate 5. Run `ormc` after editing `model.go`; `langc sync .`, remove keys it reports unused.

## Stage 4 — module/ (copy `github.com/veltylabs/staff_manager/module`, rename)

`module/module.go`: `var Module = appkit.Module{Name: "patient_directory", Server: server, Browser: browser}`.
`module/server.go` (`//go:build !browser`):
```go
func build(d appkit.Deps) (model.ModuleNaming, error) {
	return patientdirectory.New(d.DB, patientdirectory.Deps{
		Publisher: d.Events, TenantID: d.TenantID, ValidateRUT: trustedip.ValidateRUT,
	})
}
```
(`trustedip "webtyp.com/auth/trusted_ip"`, as staff_manager.) `module/browser.go` (`//go:build browser`):
`Screens: func(d appkit.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) }`.

## Stage 5 — tests

Existing tests in `tests/` (external package) that call the now-unexported methods switch to `ImportPatient` to set
up data, the plain methods to read, or the ops (as `ops_test.go`). Never export a method for a test. New
`tests/readers_test.go`:

| Test | Setup → call | Expected |
|---|---|---|
| `TestGetPatient` | `ImportPatient` p1 "Ana", RUT "12.345.678-5" → `GetPatient("t1","p1")` | "Ana", stored RUT, phone, email, `true`, nil |
| `TestGetPatientMissing` | → `GetPatient("t1","zz")`; p1 → `GetPatient("t2","p1")` | `found == false`, `err == nil` both |
| `TestFindPatient` | p1 → `FindPatient("t1","12345678-5")` | `"p1"`, `true` |
| `TestFindPatientMissing` | → `FindPatient("t1","11.111.111-1")` | `found == false`, `err == nil` |
| `TestCreatePatient` | `CreatePatient("t1","p2","12.345.678-5","Ana","","")` → `FindPatient` | `"p2"` |
| `TestCreatePatientDuplicate` | p1 has that RUT → `CreatePatient(...,"p3",...)` | `errors.Is(err, ErrRutAlreadyExists)` |

## Stage 6 — docs (Spanish)

`README.md` Step-by-Step: `Modules: []appkit.Module{patientdirectory.Module}` and the three methods; `docs/ARCHITECTURE.md`:
the six-verb rule, plain values, unexported struct forms; no `LinkPatient`/`ClientExists`.

## Delivery check

1. `gotest` passes; `langc sync .` → 0 untranslated, 0 reuse warnings.
2. `grep -rn "LinkPatient\|ClientExists\|link_patient\|FindByRut" --include=*.go --include=*.md --include=*.json .` → empty.
3. `grep -n "^func (m \*Module) [A-Z]" *.go | grep -v "op\|ModelName\|MountOperations"` lists only `GetPatient`,
   `FindPatient`, `CreatePatient`, `ImportPatient`.

Commit after every stage. Never edit `model_orm.go` by hand; a build error there means the hand-written code is wrong.
Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work. Stuck 30 minutes: open
the PR anyway listing what fails. A PR that does not compile, or whose tests could not run, is a failed delivery.
Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` before `ormc`.
If a signature in the published tag differs from what this plan states, stop and name the symbol in the PR; never adapt it by guessing.

## Stages

| Stage | Files | Done when |
|---|---|---|
| 1 | `module.go`, `ops.go`, `import.go`, `seed/seed.go` | compiles |
| 2 | `readers.go` | compiles |
| 3 | `module.go`, `ops.go`, `model.go`, `model_orm.go` (ormc), `lang.json`, 2 test files deleted | compiles |
| 4 | `module/module.go`, `module/server.go`, `module/browser.go` | compiles for browser |
| 5 | `tests/*` | tests pass |
| 6 | `README.md`, `docs/ARCHITECTURE.md` | done |
