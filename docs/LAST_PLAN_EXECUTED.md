---
PLAN: "refactor!: patient_directory on the crud-by-declaration framework (Definition + Rules, definitionc, rule files) — the base the 18 modules copy"
EXECUTOR: local
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> This plan runs **locally** and first: it is the reference module. When it is green and the owner approves, the other 18 modules are dispatched together, each with this module as the model. Do not tag, push or publish anything.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.

## Goal

The previous plan (`docs/LAST_PLAN_EXECUTED.md`, «template module on crud tables») left the working tree half way, written against a `crud` that no longer exists (`crud.Table`, `crud.Caller`, `MountPatient`, `PatientRules`, hand topics). The framework is now the one in `webtyp/docs/CRUD_BY_DECLARATION_DESIGN.md`: the module declares `Definition` + `Rules`, `definitionc` writes **one** `definition_gen.go`, and the only code the module writes is the domain rules in `rule_<entity>_<verb>.go` files. This plan migrates the tree to that, **keeping everything already done that still holds** (readers, import, `seed/demo.go`, the tests and the view).

## Prerequisites (stop and say so in the PR if one is missing)

All of these must be published; `go doc` each:

| Needs | Check |
|---|---|
| `model` with `Rules`, `AddMyBusinessRule`, `HaveNoBusinessRule`, `ModuleDefinition` | `go doc webtyp.com/model ModuleDefinition` |
| `crud` v0.0.10: `StoredEntity`, `CurrentUser`, `RuleAnswer`, `Topic`, `Loopback`, `LoopbackActingAs` | `go doc webtyp.com/crud CurrentUser` |
| `definitionc` ≥ `v0.1.41` (6 stages: single `definition_gen.go`, rule skeletons) | `definitionc` writes `definition_gen.go` containing `func (m *Module) MountOperations(` |
| `view` ≥ `v0.6.72`, `appkit` ≥ `v0.2.33`, `rpc` ≥ `v0.2.21` | `go doc webtyp.com/view Layout`; `go doc webtyp.com/appkit` |

Dependencies: `go get webtyp.com/model@v0.2.18 webtyp.com/crud@v0.0.10 webtyp.com/orm@v0.12.31 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.21 webtyp.com/definitionc@v0.1.41 webtyp.com/view@v0.6.72 webtyp.com/appkit@v0.2.33 webtyp.com/rpc@v0.2.21 && go mod tidy`. No `replace`.
Tools: `go install webtyp.com/devflow/cmd/gotest@latest webtyp.com/lang/cmd/langc@latest webtyp.com/definitionc/cmd/definitionc@v0.1.41`.

## APIs used

| symbol | exact signature | where |
|---|---|---|
| `model.ModuleDefinition` | `{Name string; Entities []model.Definition}` | `model/module_definition.go` |
| `model.Definition.Rules` | `model.Rules{Create, Read, Update, Delete model.RuleMode}` | `model/rules.go` |
| create rule | `func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` | skeleton of `definitionc` stage 4 |
| update rule | `func (next *Patient) Update(a crud.CurrentUser, tx crud.Tx, old *Patient, fields []string, m *Module) (crud.RuleAnswer, error)` | same |
| search rule | `func (p *Patient) Search(a crud.CurrentUser, filter *Patient, text string, m *Module) ([]storage.Condition, error)` (`filter` carries the fields the caller sent, `text` may be `""`: D52) | same |
| answers | `crud.ApplyAndPublish`, `crud.ApplyWithoutPublishing` (a zero `RuleAnswer` is refused: `ErrNoAnswer`) | `crud` |
| topic | `func Topic(module, entity string, a model.Action) string` | `crud/topic.go` |
| change body | `&crud.Change{Record: model.Model, Fields: []string}` | `crud` |
| store | `func (m *Module) StorePatient() crud.Store` (hand-written, one line) | design §2 |
| tests | `crud.Loopback(nil, mod)`, `crud.LoopbackActingAs(nil, "u1", mod)` | `crud` |

If a signature differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 0 — state of the tree

`git status` shows uncommitted work of the previous plan. Keep: `definition.go` (the model), `kinds/`, `module.go`, `readers.go`, `import.go`, `seed/`, `ui/`, `view.go`, `module/`, `tests/`, `docs/`. Replace: `definition_crud.go`, `rules.go`, `ops.go`, `definition_gen.go` (old generator). Commit nothing yet.

## Stage 1 — `definition.go` declares the module

1. Add the module declaration and the rules of `PatientModel`:

```go
var Definition = model.ModuleDefinition{Name: "patient_directory", Entities: []model.Definition{PatientModel}}
```

`PatientModel` keeps `Allow: model.Create | model.Read | model.Update` and gains:

```go
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.AddMyBusinessRule, Update: model.AddMyBusinessRule},
```

(`Read` is `AddMyBusinessRule` because patients are searched by RUT, phone or name: that is the `Search` rule.) No `MFA`, no `Delete`: a patient is deactivated with an update.
2. Delete the three hand topics `TopicPatientCreated`, `TopicPatientUpdated`, `TopicPatientDeactivated` (`grep -rn "TopicPatient" /home/cesar/Dev/Project --include=*.go` shows no user outside this repo): `crud` publishes `patient_directory.patient.created|updated` with a `*crud.Change` body. Deactivation is an `updated` whose `Change.Fields` contains `is_active`.
3. Keep the `Err*` constants that the rules and readers use; delete the ones no code uses (`grep` each).

## Stage 2 — generate, then write the three rules

1. Delete `definition_crud.go`, `rules.go`, `ops.go`, `definition_gen.go`. Run `definitionc`: it writes one `definition_gen.go` (`Patient`, `PatientList`, `Patient_`, and on `Module`: `ModelName()`, `Definition()`, `MountOperations()`) and the skeletons `rule_patient_create.go`, `rule_patient_update.go`, `rule_patient_search.go`.
2. Add `store.go`: `// StorePatient is the tenant-sealed store of the patients.` `func (m *Module) StorePatient() crud.Store { return m.store }`.
3. Fill the skeletons with the bodies of the deleted `rules.go` (keep `touches` and `sameDocument` as helpers in `rule_patient_update.go` / `rule_patient_create.go` respectively):

| Old (`rules.go`) | New |
|---|---|
| `checkNewPatient(_ crud.Caller, tx, p) error` | `(p *Patient) Create(a, tx, m)`: `m.checkDocument(p)`, `m.sameDocument(tx, p)`, `return crud.ApplyAndPublish, nil` |
| `checkPatientChange(_ crud.Caller, tx, old, next, fields) error` | `(next *Patient) Update(a, tx, old, fields, m)`: the same body with `m.` before the helpers; end with `return crud.ApplyAndPublish, nil` |
| `searchPatients(_ crud.Caller, text) ([]storage.Condition, error)` | `(p *Patient) Search(a, filter, text, m)`: same body; with `text == ""` it returns no condition (the rule now runs on every read) |
| `patientCommitted` | deleted: `crud` publishes after the commit |

Every rejection returns `0, err` (an error never returns an answer). The rules query with `tx`, never with `m.db`.

## Stage 3 — `module.go`, readers and import

- `module.go`: delete `publish` and the `Topic*` use; keep `Deps.Publisher` **only** for `CreatePatient`, which is called by other modules outside `crud` (see below). `New` stays as it is (`IDs` required, `orm.TenantStore`).
- `readers.go` `CreatePatient`: the rule runs through the same code the framework runs: `m.store.Atomic(func(tx crud.Tx) error { if _, err := p.Create(crud.CurrentUser{}, tx, m); err != nil { return err }; return tx.Create(&p) })`, then publish **once**: `m.pub.Publish(events.Event{Topic: crud.Topic(ModelName, "patient", model.Create), Payload: &crud.Change{Record: &p}})` guarded by `m.pub != nil`. `GetPatient`, `FindPatient`: unchanged (they already read through `m.store`).
- `import.go`: replace the `checkNewPatient` call by `p.Create(crud.CurrentUser{}, tx, m)` ignoring the answer (an import publishes nothing).
- `grep -rn "crud.Caller\|crud.Table\|MountPatient\|PatientRules\|checkNewPatient\|checkPatientChange\|searchPatients\|patientCommitted" .` → empty.

## Stage 4 — view, browser module, demo

`view.go`, `module/browser.go`, `module/server.go`, `seed/demo.go`: keep what the previous plan did (`view.NewCallerLister`, `view.ModuleBrowser`/`view.BrowserDeps`, `seed.Demo() (view.Demo, error)` with `Modules: []crud.Module{mod}`), adapted to the published `view`/`appkit` (`Layout` contract of D50; the module imports neither `layout` nor `httpd`). `grep -rn "webtyp.com/layout\|httpd\|router/loopback" .` → empty, `go.mod` included.

## Stage 5 — tests (`tests/`, `docs/TESTS.md`)

Fixtures follow `crud` v0.0.10: `crud.Loopback(nil, mod)`, `crud.LoopbackActingAs(nil, "u1", mod)`; operations are `router.Operation{Module: "patient_directory", Entity: "patient", Action: model.Create}` (never a string). `crud.Mount` must not return `ErrGeneratedOutOfDate`. Keep every test of `LAST_PLAN_EXECUTED.md` stage 6 and add:

| Test | Setup → call → expected |
|---|---|
| `TestMount_MatchesDefinition` | `crud.Mount(reg, nil, nil, mod)` → nil |
| `TestCreate_PublishesChange` | create with a `events` recorder → one event, topic `crud.Topic("patient_directory","patient",model.Create)`, body `*crud.Change` |
| `TestUpdate_DeactivateIsAnUpdatedEvent` | update `["is_active"]` false → one `updated` event whose `Fields` contain `is_active`; no other topic |
| `TestRule_NoAnswerIsRefused` | a rule returning the zero `RuleAnswer` is refused (`crud.ErrNoAnswer`) — proves the framework, with a throwaway module in the test |

`TestDelete_DoesNotExist` (calling a delete on `patient` answers «unknown operation») stays as it is.

## Stage 6 — translations and documentation

`langc sync .` (zero pending, zero reuse warnings). `README.md` (Spanish, ≤ 150 lines, 4 mandatory sections; «Sin / Con»: hand-written `ops.go` + rules vs `Definition` + three rule files) and `docs/ARCHITECTURE.md` (declaration → `definitionc` → rules; the events are `crud`'s). `docs/TESTS.md` rows for every test above.

## Delivery check

1. `gotest` (vet, style, race, tests, wasm, browser) all ✅; `langc sync .` clean.
2. Every `grep` above empty; `git status` shows no `*.orig`, no scratch.
3. `git diff --stat` lists only the files named in the stages.
4. Report in the PR what in this plan was wrong or missing: the 18 modules are dispatched from this experience.

| Stage | Files |
|---|---|
| 1 | `definition.go` |
| 2 | `definition_gen.go`, `rule_patient_create.go`, `rule_patient_update.go`, `rule_patient_search.go`, `store.go`; delete `definition_crud.go`, `rules.go`, `ops.go` |
| 3 | `module.go`, `readers.go`, `import.go` |
| 4 | `view.go`, `module/*.go`, `seed/demo.go` |
| 5 | `tests/*.go`, `docs/TESTS.md` |
| 6 | `lang.json`, `README.md`, `docs/*` |

## Executed locally — what the plan got wrong or missed (for the other modules)

1. **`crud` v0.0.10 validated the filter as a new record** (`id required` on every read of a table with a rule): a filter has no required field. Fixed in `crud` (local, **unpublished**, to be v0.0.11: `validateFilter`); until it is published `go.mod` carries a temporary `replace webtyp.com/crud => ../../../webtyp/crud`, which must be removed after `go get webtyp.com/crud@v0.0.11`.
2. **`definitionc` v0.1.41 gives `Search` the filter as a parameter** (`Search(a, filter *Patient, text, m)`), not as the receiver; the plans now say so. It also returns no condition for an empty text in this module's rule (the rule runs on every read).
3. **Running `definitionc` rewrote `go.mod` to older versions.** After running it, run the `go get` list again, then `go mod tidy`. Indirect dependencies were stale too (`rbac` needed ≥ v0.0.37, `mcp` ≥ v0.3.25, `appkit` v0.2.33, not v0.2.32).
4. **Events in tests**: `crud` publishes on the publisher given to `crud.LoopbackActingAs(pub, "u1", mod)`, not on `Deps.Publisher`; a test that counts events passes its recorder there. `Deps.Publisher` stays only for `CreatePatient`, which is called outside `crud`.
5. **A caller double** (`mockCaller`) takes `router.Operation` in `Call` and `Dispatch`; a dummy `ddl.Execer` returns `(storage.Result, error)`.
6. **`ErrNoAnswer`** reaches the client as `500 internal error` (the text is not shown): the test asserts a refusal and that nothing was stored.
7. `langc sync .` reports 25 keys untranslated in `zh, hi, ar, pt, fr, de, ru`: the repo's `lang.json` lists 8 languages and carries no translations; not caused by this plan.
