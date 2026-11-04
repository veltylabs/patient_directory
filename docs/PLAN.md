---
PLAN: "feat: patient identity for foreigners and patients without documents; sex, notes, estimated birth date; ImportPatient"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 10738169610157971371
PR: https://github.com/veltylabs/patient_directory/pull/13
REVIEWED_COMMIT: 448d48dc354dca19124b0738826a728b5a48c625
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — a patient is a person, the document is a fact that may be missing

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` before `ormc`.

## Why

The clinic sees many foreign patients. Some have no Chilean RUT, only their country's passport or id card, and some
have nothing at hand. Today `rut` is required and validated (`module.go:67`), so the front desk cannot register them:
the RUT (a tax id) blocks the care. Also missing: the patient's **sex**, free **notes** (the old paper file number goes
there: `Ficha antigua Pa100T: 2006-1`), and whether the birth date is **estimated** (typed as an age). Finally the
importer (`mjosefa-cms/migrate/pastdb`) must store each patient of the previous system (46,330) with the id it had
there, so importing twice updates instead of duplicating.

## Ecosystem search

- `input.Gender()` (`webtyp.com/input/gender.go`, options `m`/`f`) → **reuse** for `sex`; do NOT create another.
- Closed options of a module field: `veltylabs/device_manager/kinds/kinds.go` (`input.Radio` + constants in package
  `kinds`, because ormc runs the constructor) → **copy the pattern** for `document_type`.
- `CreatePatient` takes a given id and is idempotent by id (`module.go:67`) → keep.

## Design gate

1. Prior art: HL7 FHIR `Patient.identifier` = {type, issuer, value}, types `NN` (national) and `PPN` (passport) of
   HL7 table v2-0203 (https://hl7.org/fhir/patient.html#identifiers); passports per ICAO Doc 9303 (up to 9 letters and
   digits + issuing country); countries as ISO 3166-1 alpha-2. Followed: type + number + country; only the Chilean
   RUT has a check we can compute.
2. Names: `document_type` (`rut`, `passport`, `foreign_id`, `none`, `unverified`), `document_country` (`VE`, `PE`…),
   `sex`, `notes`, `birthdate_estimated`.
3. Ledger: +4 fields, +1 method `ImportPatient`, +1 package `kinds`. The column `rut` keeps its name and now holds
   the document number of any type (no rename: other modules call `FindByRut`); its label becomes "Document number".
4. Belongs here: identity of a person (no clinical data: the model comment forbids it).
5. Deletes: `NotNull` on `is_active` (a bool with NotNull rejects `false`: an inactive patient could not be saved).

## Stage 1 — `kinds/kinds.go` (new)

```go
package kinds

const (
	DocumentRUT        = "rut"        // Chilean RUT: check digit validated; unique per tenant
	DocumentPassport   = "passport"   // any country's passport, stored as typed
	DocumentForeignID  = "foreign_id" // another country's id card, stored as typed
	DocumentNone       = "none"       // the person has no document at hand
	DocumentUnverified = "unverified" // typed but invalid; the front desk must fix it
)

// DocumentType: closed options for patient.document_type. Example: in := kinds.DocumentType()
func DocumentType() input.Input {
	return input.Radio(
		fmt.KeyValue{Key: DocumentRUT, Value: "RUT"},
		fmt.KeyValue{Key: DocumentPassport, Value: "Passport"},
		fmt.KeyValue{Key: DocumentForeignID, Value: "Foreign id card"},
		fmt.KeyValue{Key: DocumentNone, Value: "No document"},
		fmt.KeyValue{Key: DocumentUnverified, Value: "To verify"},
	)
}
```

## Stage 2 — `model.go` (then `ormc`; never edit `model_orm.go` by hand)

In `PatientModel.Fields`:
- `rut`: `Type: input.Text()`, `Label: "Document number"`, **drop `NotNull`** and add `OmitEmpty: true` (empty for `none`),
  `Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-', '.'}, Maximum: 30}`.
- after `rut`: `{Name: "document_type", Type: kinds.DocumentType(), Label: "Document type", NotNull: true, Item: model.ItemDescription}`
- after `document_type`: `{Name: "document_country", Type: input.Text(), Label: "Issuing country", Help: "Two-letter country code of the passport or id card, e.g. VE, PE, HT. Empty for a Chilean RUT.", OmitEmpty: true, Permitted: model.Permitted{Letters: true, Minimum: 2, Maximum: 2}}`
- after `birthdate`: `{Name: "birthdate_estimated", Type: input.Checkbox(), Label: "Estimated birth date", Help: "Mark it when the date comes from the age the patient said, not from a document."}`
- after `birthdate_estimated`: `{Name: "sex", Type: input.Gender(), Label: "Sex", OmitEmpty: true}`
- after `address`: `{Name: "notes", Type: input.Textarea(), Label: "Notes", Help: "Anything the front desk must know about this patient, e.g. the old paper file number.", OmitEmpty: true}`
- `is_active`: remove `NotNull: true` (comment: a bool with NotNull rejects false).

## Stage 3 — `module.go`: one document rule for create and update

```go
// checkDocument normalizes and validates p's document by its type. An empty type means RUT (what the form sends
// today). Only a RUT is validated; other documents are stored as typed, upper-cased and trimmed.
func (m *Module) checkDocument(p *Patient) error {
	if p.DocumentType == "" {
		p.DocumentType = kinds.DocumentRUT
	}
	p.Rut = strings.TrimSpace(p.Rut)
	p.DocumentCountry = strings.ToUpper(strings.TrimSpace(p.DocumentCountry))
	switch p.DocumentType {
	case kinds.DocumentRUT:
		norm, err := m.validateRUT(p.Rut)
		if err != nil {
			return &model.FieldError{Field: "rut", Err: err}
		}
		p.Rut, p.DocumentCountry = norm, ""
	case kinds.DocumentPassport, kinds.DocumentForeignID:
		if p.Rut == "" {
			return &model.FieldError{Field: "rut", Err: ErrDocumentNumber}
		}
		p.Rut = strings.ToUpper(p.Rut)
	case kinds.DocumentNone:
		p.Rut, p.DocumentCountry = "", ""
	case kinds.DocumentUnverified:
		// stored as typed: the front desk fixes it
	default:
		return &model.FieldError{Field: "document_type", Err: ErrDocumentType}
	}
	return nil
}
```

Use `strings` from `webtyp.com/fmt` helpers if the repo avoids stdlib `strings` (check the existing imports of
`module.go` and follow them). In `CreatePatient` and `UpdatePatient` replace the `rut` required check and the
`validateRUT` call with `checkDocument`. Uniqueness: replace the `FindByRut` → `ErrRutAlreadyExists` check with
`m.findSameDocument(p)` — another patient of the tenant with the same `document_type`, `document_country` and `rut`,
**only when `rut != ""`** and the type is not `unverified` → `ErrRutAlreadyExists` (keep the name: other code uses it).
`sex` must be `""`, `m` or `f` → else `&model.FieldError{Field: "sex", Err: ErrSex}`.

New errors in `errors.go`, same style as the existing ones:
- `ErrDocumentType` = "the document type must be RUT, passport, foreign id card, no document or to verify"
- `ErrDocumentNumber` = "a passport or foreign id card needs its number"
- `ErrSex` = "sex must be female, male or not recorded"

## Stage 4 — `import.go` (new): `ImportPatient`

```go
// ImportPatient stores a patient from a previous system with the id it had there: it creates it, or updates every
// field when that id already exists, so importing twice leaves the same row. It validates exactly like
// CreatePatient (name, document rules, sex, unique document) and publishes no event.
//
// Example: p, err := dir.ImportPatient(Patient{Id: "1640011384447734400", TenantId: t, DocumentType: kinds.DocumentRUT, Rut: "12345678-5", Name: "Ana"})
func (m *Module) ImportPatient(p Patient) (Patient, error)
```

Body: `Id`, `TenantId`, `Name` required (same errors as `CreatePatient`); `checkDocument`; sex check; same document
on **another id** → `ErrRutAlreadyExists`; read by id: found (same tenant) → `m.db.Update(&p, orm.Eq(Patient_.Id, p.Id),
orm.Eq(Patient_.TenantId, p.TenantId))`; another tenant → `ErrIdTaken`; not found → `m.db.Create(&p)`;
`UpdatedAt = time.NowNano()`. Never `m.db.Tx` (the importer runs inside one), never `m.publish`.

## Stage 5 — tests (`tests/document_test.go`, `tests/import_test.go`)

| Test | Call | Expected |
|---|---|---|
| `TestCreatePatientDefaultsToRUT` | no type, bad check digit | FieldError `rut`; good RUT → stored type `rut`, normalized |
| `TestPassportNeedsNumber` | `passport`, empty number | `ErrDocumentNumber` |
| `TestPassportStoredAsTyped` | `passport`, `"  ab123456 "`, country `"ve"` | `"AB123456"`, `"VE"` |
| `TestNoDocument` | `none`, number `"x"` | stored with empty number |
| `TestSameDocumentTwice` | two patients, same passport and country | second → `ErrRutAlreadyExists` |
| `TestSamePassportOtherCountry` | same number, `VE` and `PE` | both stored |
| `TestTwoWithoutDocument` | two `none` | both stored |
| `TestSexValues` | `x` | FieldError `sex` |
| `TestInactivePatientSaves` | `IsActive: false` | saved |
| `TestImportPatientCreatesThenUpdates` | twice, new name | 1 row, name updated |
| `TestImportPatientUnverified` | `unverified`, `"12345678-0"` | stored as typed |
| `TestImportPublishesNothing` | counting fake Publisher | 0 events |

## Stage 6 — translations and docs

`langc sync .`, fill every Spanish value ("Document number" → "Número de documento", "Issuing country" → "País
emisor", "To verify" → "Por verificar", "Estimated birth date" → "Fecha de nacimiento aproximada"). `README.md` and
`docs/ARCHITECTURE.md` **in Spanish**: the document rule (only RUT is validated; why the phone is not an identifier:
families share it), `ImportPatient`, each exported symbol with a usage example.

## Delivery check

1. `gotest` passes (a PR that does not compile, or whose tests could not run, is a failed delivery).
2. `langc sync .` → 0 untranslated `es`, 0 reuse warnings.
3. `grep -rn "Sex()" kinds/` → empty (sex uses `input.Gender()`); `grep -n "m.db.Tx\|publish" import.go` → empty.

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
