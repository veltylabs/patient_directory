---
PLAN: "fix: opListPatients has no tenant fallback, so every real list load (which sends no args) returns zero rows"
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# PLAN — `patient_directory`: add `Deps.TenantID` and fall back to it in `opListPatients`

You are an external agent with **zero prior context** about this project. Everything you need is
in this file. Read `AGENTS.md` at the repo root first, then this file fully before writing code.

## 0. Prerequisite — run this first

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

All tests run with `gotest`, never `go test` directly. `go.mod`/`go.sum` already carry
`webtyp.com/router/mock` (added for the failing test below) — no further dependency work needed.

## 1. The bug, proven by a failing test already in this repo

`tests/tenant_fallback_test.go` (already committed on this branch) reproduces a real bug found
while manually testing a downstream app's patient list screen: it showed "0 / 0" even though a
real patient existed for the app's tenant. Run it now and confirm it is RED — it does not even
compile yet, which is the expected starting point:

```bash
gotest
# tests/tenant_fallback_test.go:51:3: unknown field TenantID in struct literal of type patientdirectory.Deps
```

The acceptance criterion: **`gotest` must go fully green**, including this test, with no other
test regressing.

## 2. Root cause

`webtyp.com/view`'s `callerLister.list()` (the code every crudview-backed list runs on `Reload()`)
always calls the List op with **no args** — this is correct, by design: `router.Caller.Call`'s own
doc says args may be nil, and a List op is expected to tolerate an absent/empty args object (an
already-fixed `webtyp.com/mcp` bug, unrelated to this one, made sure "no args" now round-trips as
`{}` instead of corrupting the request). So on every real page load, `opListPatients` (`ops.go`)
receives `ListPatientsArgs{TenantId: ""}`.

`Module.ListPatients` (`module.go`) has an **explicit early return** for exactly this case:

```go
func (m *Module) ListPatients(tenantID string, filter PatientFilter) ([]Patient, error) {
	if tenantID == "" {
		return []Patient{}, nil
	}
	...
```

— silently returning nothing, no error, for every real page load. The list renders "0 / 0" even
for a tenant with real rows.

`github.com/veltylabs/staff_manager` already solves this exact problem, and this plan ports its
pattern exactly:

```go
// staff_manager/module.go
type Deps struct {
	IDs         model.IDGenerator
	Publisher   events.Publisher
	TenantID    string
	...
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("staff_manager: Deps.IDs is required")
	}
	if deps.TenantID == "" {
		return nil, fmt.Err("staff_manager: Deps.TenantID is required")
	}
	...
}

// staff_manager/ops.go
func (m *Module) opListStaff(ctx router.Context) {
	var args ListStaffArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	tenantID := args.TenantId
	if tenantID == "" {
		tenantID = m.tenantID
	}
	list, err := m.ListStaff(tenantID)
	...
}
```

`patient_directory` has neither the field nor the fallback.

## 3. The fix

### `module.go`

1. Add `TenantID string` to `Deps`, next to `Publisher`:
   ```go
   type Deps struct {
   	IDs       model.IDGenerator // requerido — el módulo nunca genera los suyos propios
   	Publisher events.Publisher  // opcional — nil deshabilita la publicación silenciosamente
   	// TenantID identifica esta instalación — el fallback que opListPatients usa
   	// cuando quien llama no envía tenant_id (todo listado respaldado por
   	// crudview lo hace así).
   	TenantID string
   	ValidateRUT func(string) (string, error)
   }
   ```
   (keep every existing field and its comment exactly as-is; only `TenantID` is new — place it
   wherever reads best next to `Publisher`, `ValidateRUT`'s position does not need to move).
2. Add `tenantID string` to `Module`, next to `pub`.
3. In `New`, require it exactly like `ValidateRUT`:
   ```go
   func New(db *orm.DB, deps Deps) (*Module, error) {
   	if deps.IDs == nil {
   		return nil, fmt.Err("patient_directory: Deps.IDs is required")
   	}
   	if deps.ValidateRUT == nil {
   		return nil, fmt.Err("patient_directory: Deps.ValidateRUT is required")
   	}
   	if deps.TenantID == "" {
   		return nil, fmt.Err("patient_directory: Deps.TenantID is required")
   	}
   	return &Module{
   		db:          db,
   		ids:         deps.IDs,
   		pub:         deps.Publisher,
   		validateRUT: deps.ValidateRUT,
   		tenantID:    deps.TenantID,
   	}, nil
   }
   ```

### `ops.go`

In `opListPatients`, between decoding args and calling `m.ListPatients`, add the fallback:

```go
func (m *Module) opListPatients(ctx router.Context) {
	var args ListPatientsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}
	tenantID := args.TenantId
	if tenantID == "" {
		tenantID = m.tenantID
	}
	patients, err := m.ListPatients(tenantID, PatientFilter{
		ActiveOnly: args.ActiveOnly,
		Limit:      args.Limit,
		Offset:     args.Offset,
	})
	...
```

(the rest of the function body is unchanged — only the `tenantID` variable is introduced and used
in place of `args.TenantId` on the `m.ListPatients(...)` call). Leave `Module.ListPatients`'s own
`if tenantID == "" { return []Patient{}, nil }` guard exactly as it is — it is a legitimate
defense at the service layer; the bug is that nothing upstream of it ever supplies a real tenant
when the caller sent none.

### What NOT to do

- **Do not add this fallback to `opGetPatient`, `opFindPatientByRut`, or `opDeactivatePatient`.**
  Those ops always receive an explicit `id`/`rut` from a real UI action — never through
  `callerLister.list()`'s args-less path — so they are not affected by this bug. `staff_manager`
  draws the identical line (only its List op falls back). Adding it everywhere "for consistency"
  would be scope creep past what this bug requires.
- **Do not touch `opUpsertPatient`.** It takes a full `Patient` record from the caller, including
  its own `tenant_id` — an upsert is never called with an empty tenant by design.
- **Do not touch `webtyp.com/view`.** `callerLister.list()` passing no args is correct per
  `router.Caller`'s own documented contract — the fix is this module adopting the fallback
  `staff_manager` already has, not changing the generic caller.
- **Do not make `Deps.TenantID` optional with a silent empty-string default.** Require it in `New`,
  exactly like `ValidateRUT` — a module silently running with an empty tenant fallback would scope
  every fallback query to `tenant_id = ''`, which is a worse, harder-to-diagnose version of today's
  bug.
- **Do not remove or weaken `Module.ListPatients`'s own `tenantID == ""` guard.** It stays as a
  legitimate defense-in-depth check; the fix is making sure a real tenant reaches it, not removing
  the check.

## 4. Verification

```bash
gotest
# vet ✅, race ✅, tests ✅ — TestOpListPatients_FallsBackToModuleTenant now PASSES,
# and every other existing test (tenant_test.go, patient_test.go, ops_test.go, etc.) still passes.
```

## 5. Downstream consumers (informational — not part of this plan's scope)

Once this ships as a new tagged version, `github.com/veltylabs/mjosefa-cms` needs: a version bump
(`go get github.com/veltylabs/patient_directory@<new-version>`), AND its own
`modules/patient_directory/server.go` wrapper updated to actually pass `tenantID` through to
`patientdirectory.Deps{TenantID: tenantID}` (today that parameter is received and documented as
unused, matching `device_manager`'s and `clinical_encounter`'s wrapper comments). Both are the
consuming app's own job, tracked there, not here.

## Stages

| # | Stage | File(s) | Acceptance |
|---|---|---|---|
| 1 | Add `Deps.TenantID`, require it in `New` | `module.go` | Matches the exact shape shown above |
| 2 | Fall back to it in `opListPatients` only | `ops.go` | No other op touched |
| 3 | Verify | — | `gotest` green, `TestOpListPatients_FallsBackToModuleTenant` passes |
