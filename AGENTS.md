# Directorio de Pacientes (`patient_directory`)

<!-- webtyp:base — copied from devskills skill agents-md; do not edit here -->
## Ecosystem rules — read before any change

`docs/PLAN.md` is the spec: no design judgment; anything unclear is a question in the PR, not a guess.

- Public API: typed (no `any`/`interface{}`), explicit, minimal; never export a symbol only a test uses.
- One intent, one path: the old path is deleted in the same change. No silent fallback, `TODO`, commented code or stub.
- Fail at compile time, else a loud diagnostic — never silently. Zero values deny.
- A defect in a dependency is reported, never wrapped or patched here.
- No string literals in logic: named constants. The same logic never lives in two repos: import the package the plan names; if you find an existing one the plan does not name that already does what you are about to write, say so in the PR instead of writing it.
- `cmd/*/main.go`: parse flags, inject, print/exit only. stdout = data, stderr = logs, exit codes are the contract.
- WASM code: `webtyp/fmt`, `webtyp/time`, `webtyp/json` instead of stdlib; no `map`; `dom.Element` embedded by value.
- `css.go`, `svg.go`, `js.go`, `html.go` carry `//go:build !wasm`; no `ssr.go`, no `front.go`.
- Build tags: `wasm`/`!wasm` only when the API does not exist on that platform; `browser`/`!browser` when the file belongs to another binary One main per folder, no destination tag: `web/browser/main.go` → `web/public/browser.wasm`, `web/server/main.go` → `web/server/server`, `web/edge/main.go` → `web/edge/edge.wasm`.
- Code and keys in English; UI text is `lang.Text`, translations live in `lang.json`. Comments and docs: English in `webtyp/*`; in `veltylabs/*` **every README, `docs/*` file, diagram and code comment is in Spanish**, written for a reader with little technical background, and each exported symbol's comment shows a short usage example.
- A translation key is reused, never duplicated as a near-identical phrase (`"not a valid integer"` + `"not a valid decimal number"` repeats `"not a valid"` three ways instead of composing it once) — every library ships to WASM/TinyGo, so each unreused phrase is dictionary rows AND binary bytes, in every consumer, forever. A token that doesn't need translating (a kind/type name like `int`) is passed as data (a variable), never as a string literal — a literal becomes a dictionary key whether or not it needed one.
- Tests cover behaviour and domain rules, never the look (CSS, colours, markup order, class strings). In `tests/` (external package), real stack with fakes only at the edge; run `gotest`.
- Docs (`README.md`, `docs/*`) updated with the API; permanent docs never link `docs/PLAN.md`.
- Before the PR: `git diff --stat main` lists every file of the plan's stages, and nothing the plan did not order is deleted.
<!-- /webtyp:base -->

## This repo

Este repositorio posee los datos del registro de pacientes por tenant: únicamente identidad y detalles de contacto.

## Decisiones Arquitectónicas

1. **Identidad y alcance por Tenant:** El RUT es la identidad nacional primaria y es único por tenant. El id del paciente lo genera quien llama (UI, seed, importación); `CreatePatient` lo exige y repetir la misma creación devuelve el paciente guardado.
2. **Unicidad en la Capa de Aplicación:** La unicidad del RUT por tenant se exige en el código de la aplicación (vía `ErrRutAlreadyExists`), no mediante una restricción de unicidad de columna única en la base de datos. NO establecer `Unique: true` en la definición del campo `rut` en los modelos.
3. **Sin eliminación física:** Los pacientes se desactivan (`is_active = false`), nunca se eliminan, para preservar la integridad referencial de atenciones clínicas e historial de reservas.
4. **Validación de RUT Inyectada:** La función de validación y normalización de RUT `ValidateRUT` se inyecta en `Deps`. No importar ni implementar lógica de dígito verificador de RUT en este repositorio.
5. **Sin campos clínicos:** No almacenar diagnósticos, consultas, recetas ni notas clínicas en este módulo. Los datos clínicos pertenecen a `clinical_encounter`.
6. **Sin lenguaje humano:** Este módulo no incluye cadenas de interfaz de usuario ni traducciones de lenguaje humano (por ejemplo, sin etiquetas ni frases descriptivas en la lógica central). La traducción y formateo de etiquetas es responsabilidad de las aplicaciones consumidoras.

## ui/, seed/ and web/ — the module's own view and demo

The whitelist and blacklist above apply to the **root domain package**. Three sub-packages are exempt, and only them:

- `ui/` (package `ui`, no build tag) declares the module's screens: `const ID`, `const Label` and `Screens(caller router.Caller) []view.Screen`. It imports `webtyp.com/view`, `webtyp.com/router` and the module itself; never `webtyp.com/layout`.
- `seed/` (package `seed`, no build tag) holds demo data: `Load(...) (Data, error)`, which writes through the module's own methods so every row is validated, and returns the rows it created so downstream demos can reference them.
- `web/` (package `main`, `web/browser.go` tagged `browser`) is the runnable demo and may additionally import the concrete `webtyp.com/storage/mem`, `webtyp.com/router/loopback`, `webtyp.com/events/mock`, `webtyp.com/unixid` and `webtyp.com/auth/trusted_ip` (for `ValidateRUT`).

**Dependencies between modules form a DAG.** The root domain package still imports no sibling module. `ui/`, `seed/` and `web/` may import the domain, `ui/` and `seed/` packages of **upstream** modules only. Current graph: `device_manager`, `item_catalog`, `patient_directory`, `business_calendar`, `payment_manager` (leaves) ← `staff_manager` ← `appointment_booking` ← `clinical_encounter` (`appointment_booking` also depends on `item_catalog`, `patient_directory`, `business_calendar`, `payment_manager`; `clinical_encounter` also on `patient_directory` and `staff_manager`). A screen that mixes modules lives in the most-downstream module it touches.
