---
PLAN: "feat: patient sex, notes and document type; ImportPatient for history"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 2065367097368748488
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — patients with sex, notes and document type, and a path to import history

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` before `ormc`.

## Why

The clinic's previous system (Pa100T) holds 46,330 patients. Three things it records are missing here: the
patient's **sex**, free **notes** (the old paper file number goes there: `Ficha antigua Pa100T: 2006-1`), and the
**kind of identity document**: 476 patients have a passport or a RUT typed wrong. Today `rut` is required and
validated, so those patients cannot be stored at all. The importer (`mjosefa-cms/migrate/pastdb`) also needs to store
each patient with the id it had in Pa100T, so running the import twice updates instead of duplicating.

## Ecosystem search

- `CreatePatient` already takes a given id and is idempotent by id (`module.go:67`) → **reuse** its checks.
- Closed option fields: `veltylabs/device_manager/kinds/kinds.go` (`input.Radio` + constants in package `kinds`,
  because ormc runs the constructor) → **copy that pattern** into a new `kinds/` package here.
- No other repo models a patient's document type → **new** fields.

## Design gate

1. Prior art: HL7 FHIR `Patient.gender` and `Patient.identifier.type` (https://hl7.org/fhir/patient.html) —
   identity document as *type + value*; followed.
2. Names: `document_type` with values `rut`, `foreign`, `unverified`; `sex` with `m`, `f`, `""` (not recorded).
3. Ledger: +3 fields, +1 method (`ImportPatient`), +1 package `kinds`. Ways to store a patient: `CreatePatient` for
   the app, `ImportPatient` only for history (they differ only as described below).
4. Belongs here: identity data of a patient (not clinical: the model comment forbids clinical data).
5. Deletes: `NotNull` on `is_active` (a bool with NotNull rejects `false`: an inactive patient could not be saved).

## Stage 1 — `kinds/kinds.go` (new; copy the structure of device_manager/kinds)

```go
package kinds

const (
	SexMale    = "m"
	SexFemale  = "f"
	SexUnknown = "" // not recorded

	DocumentRUT        = "rut"        // Chilean RUT, validated, unique per tenant
	DocumentForeign    = "foreign"    // passport or foreign id, stored as typed, not validated
	DocumentUnverified = "unverified" // typed but invalid; reception must fix it
)

// Sex: closed options for patient.sex. Example: in := kinds.Sex()
func Sex() input.Input { return input.Radio(fmt.KeyValue{Key: SexFemale, Value: "Female"}, fmt.KeyValue{Key: SexMale, Value: "Male"}) }

// DocumentType: closed options for patient.document_type. Example: in := kinds.DocumentType()
func DocumentType() input.Input {
	return input.Radio(
		fmt.KeyValue{Key: DocumentRUT, Value: "RUT"},
		fmt.KeyValue{Key: DocumentForeign, Value: "Passport or foreign document"},
		fmt.KeyValue{Key: DocumentUnverified, Value: "To verify"},
	)
}
```

## Stage 2 — `model.go`

In `PatientModel.Fields`:
- `rut`: change `Type: input.Rut()` → `Type: input.Text()`, `Label: "Document"`, `Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-', '.', 'K', 'k'}, Minimum: 1, Maximum: 30}`. The RUT check moves to the module (only for `document_type == rut`). Keep the column name `rut` (no migration).
- add after `rut`: `{Name: "document_type", Type: kinds.DocumentType(), Label: "Document type", NotNull: true, Item: model.ItemDescription}`
- add after `birthdate`: `{Name: "sex", Type: kinds.Sex(), Label: "Sex", OmitEmpty: true}`
- add after `address`: `{Name: "notes", Type: input.Textarea(), Label: "Notes", Help: "Anything reception must know about this patient, e.g. the old paper file number.", OmitEmpty: true}`
- `is_active`: remove `NotNull: true` (comment: a bool with NotNull rejects false).
Then `ormc` (regenerates `model_orm.go`; never edit it by hand).

## Stage 3 — `module.go`: document rules (CreatePatient, UpdatePatient)

Replace the unconditional `validateRUT` with:

```go
// checkDocument normalizes and validates p's document by its type. An empty type means RUT (the app's form).
func (m *Module) checkDocument(p *Patient) error {
	if p.DocumentType == "" {
		p.DocumentType = kinds.DocumentRUT
	}
	switch p.DocumentType {
	case kinds.DocumentRUT:
		norm, err := m.validateRUT(p.Rut)
		if err != nil {
			return &model.FieldError{Field: "rut", Err: err}
		}
		p.Rut = norm
	case kinds.DocumentForeign, kinds.DocumentUnverified:
		p.Rut = strings.TrimSpace(p.Rut) // stored as typed
	default:
		return &model.FieldError{Field: "document_type", Err: ErrDocumentType}
	}
	return nil
}
```

Use it in `CreatePatient` and `UpdatePatient` where `validateRUT` is called today. The uniqueness check
(`FindByRut` → `ErrRutAlreadyExists`) runs **only when `DocumentType == DocumentRUT`**. `sex` must be `""`, `m` or
`f`, else `&model.FieldError{Field: "sex", Err: ErrSex}`. New errors in `errors.go`, same style as the existing ones:
`ErrDocumentType` = "the document type must be RUT, foreign or to verify"; `ErrSex` = "sex must be female, male or not recorded".

## Stage 4 — `import.go` (new): `ImportPatient`

```go
// ImportPatient stores a patient from a previous system with the id it had there: it creates it, or updates every
// field when that id already exists, so importing twice leaves the same row. It validates exactly like CreatePatient
// (required fields, document rules, unique RUT) and publishes no event: importing history is not a business action.
//
// Example: p, err := dir.ImportPatient(Patient{Id: "1640011384447734400", TenantId: t, Rut: "12345678-5", DocumentType: kinds.DocumentRUT, Name: "Ana"})
func (m *Module) ImportPatient(p Patient) (Patient, error)
```

Body: required `Id`, `TenantId`, `Name` (same errors as `CreatePatient`); `checkDocument`; sex check; if a RUT-type
patient with the same normalized RUT exists **with another id** → `ErrRutAlreadyExists`; read by id: found (same
tenant) → `m.db.Update` all fields with `orm.Eq(Patient_.Id, p.Id), orm.Eq(Patient_.TenantId, p.TenantId)`; not found →
`m.db.Create`. Set `UpdatedAt = time.NowNano()`. **Never** call `m.db.Tx` (the importer already runs inside one) and
never `m.publish`.

## Stage 5 — tests (`tests/import_test.go`, `tests/document_test.go`)

| Test | Call | Expected |
|---|---|---|
| `TestImportPatientCreatesThenUpdates` | ImportPatient twice, second with new name | 1 row, name updated, same id |
| `TestImportPatientForeign` | `DocumentType: foreign`, `Rut: "P1234567"` | stored as typed |
| `TestImportPatientUnverified` | `unverified`, `Rut: "12345678-0"` (bad check digit) | stored as typed |
| `TestImportPatientRutTakenByOtherId` | two ids, same valid RUT | second → `ErrRutAlreadyExists` |
| `TestCreatePatientDefaultsToRUT` | CreatePatient without DocumentType, bad RUT | rut FieldError; good RUT → type `rut` |
| `TestSexValues` | `x` | FieldError on `sex` |
| `TestInactivePatientSaves` | `IsActive: false` | saved |
| `TestImportPublishesNothing` | counting fake Publisher | 0 events |

## Stage 6 — translations and docs

`langc sync .`, fill every Spanish value (e.g. "Document type" → "Tipo de documento", "To verify" → "Por verificar").
`README.md` and `docs/ARCHITECTURE.md`: the document-type rule and `ImportPatient`, **in Spanish** (veltylabs rule),
each exported symbol's comment with a usage example.

## Delivery check

1. `gotest` passes (a PR that does not compile, or whose tests could not run, is a failed delivery).
2. `langc sync .` → 0 untranslated `es`, 0 reuse warnings.
3. `grep -n "m.db.Tx\|publish" import.go` → empty.

Commit after every stage. Never edit `model_orm.go` by hand; a build error there means the hand-written code is
wrong. Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work. Stuck 30
minutes: open the PR anyway listing what fails.

## Stages

| Stage | Files | Done when |
|---|---|---|
| 1 | `kinds/kinds.go` | compiles |
| 2 | `model.go`, `model_orm.go` (ormc) | ormc ran clean |
| 3 | `module.go`, `errors.go` | compiles |
| 4 | `import.go` | compiles |
| 5 | `tests/*` | tests pass |
| 6 | `lang.json`, `README.md`, `docs/ARCHITECTURE.md` | done |
