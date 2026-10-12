---
PLAN: "feat: link_patient — vincula un paciente por RUT sin mostrar ni modificar sus datos (reserva online de invitado)"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 6559570370499122425
PR: https://github.com/veltylabs/patient_directory/pull/6
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.

> **Pieza D** de
> [`mjosefa-cms/docs/RESERVA_Y_PAGOS_MASTER_PLAN.md`](https://github.com/veltylabs/mjosefa-cms/blob/main/docs/RESERVA_Y_PAGOS_MASTER_PLAN.md)
> (decisión RP-7). Compuerta de la pieza B (`appointment_booking`).

# Plan — `patient_directory`: op `link_patient`

## 0. Contexto (leer primero)

La reserva online la hace un **invitado**: escribe RUT, nombre, correo y teléfono, y paga con
Webpay. Para guardar la reserva hace falta el id del paciente. Las ops que existen no sirven
para un invitado:

- `find_patient_by_rut` devuelve la ficha completa: un desconocido vería nombre, teléfono y
  correo de quien tenga ese RUT.
- `upsert_patient` sobrescribe: un desconocido podría cambiar el teléfono o el correo de otra
  persona.

`link_patient` resuelve las dos cosas: **devuelve solo un id**. Si el RUT no existe, crea el
paciente con los datos recibidos; si existe, devuelve el id guardado **sin tocar ni mostrar
nada**. La respuesta tiene la misma forma en los dos casos.

## Design gate

1. **Prior art:** "guest checkout" de Reservo y AgendaPro (Chile) y de Zocdoc: el invitado
   escribe sus datos y el sistema lo empareja por documento sin exponer la ficha. FHIR
   `Patient/$match` devuelve candidatos solo a sistemas autorizados, nunca al público; aquí la
   respuesta al público es únicamente un id opaco.
2. **Prueba del nombre:** `LinkPatient(cmd)` / op `link_patient`: "vincula (o registra) al
   paciente de este RUT y dame su id". `LinkPatientCmd`, `LinkPatientArgs`,
   `LinkPatientResult{Id}`.
3. **Libro de complejidad:** +1 método exportado, +1 op, +2 modelos (`LinkPatientArgs`,
   `LinkPatientResult`), +1 constante de recurso. No cambia nada existente.
4. **Dónde vive:** aquí, porque el dueño del paciente y de la regla "RUT único por tenant" es
   este módulo (OF-20).
5. **Qué borra:** nada. Evita que `appointment_booking` tenga que exponer `upsert_patient` al
   público.

## 1. Reglas del repositorio (aplicar sin excepción)

- Código wasm/TinyGo: nada de `strings`, `strconv`, `errors` ni `fmt` de la stdlib → usar
  `webtyp.com/fmt`. **Sin `map`** en ningún archivo, pruebas incluidas. Sin `reflect`.
- Modelos nuevos se declaran como `model.Definition` en `model.go` y se regenera
  `model_orm.go` con `go install webtyp.com/ormc/cmd/ormc@latest && ormc` en la raíz del
  repo. **Nunca editar `model_orm.go` a mano.**
- Op: `ctx.Decode(&args)` (error ⇒ 400) → `args.Validate(model.Create)` (error ⇒ 400) →
  servicio → `ctx.Encode`. Errores con el `handleErr` existente.
- **Todas** las pruebas en `tests/` (paquete `tests`, solo API exportada), con
  `orm.New(mem.New())`. Correr todo con `gotest`. No exportar nada solo para una prueba.

## 1.1 Método: diagrama → pruebas en rojo → código (TDD), 100% del código nuevo

- Prerrequisitos: `go install webtyp.com/devflow/cmd/gotest@latest` y
  `go install webtyp.com/ormc/cmd/ormc@latest`.
- Fuente de verdad: [`docs/diagrams/link_patient.md`](diagrams/link_patient.md) (ya escrito).
  Cada rama tiene su prueba en §4 con nombre exacto.
- Orden: etapa 1 con `LinkPatient` y `opLinkPatient` en `panic("sin implementar")` → escribir
  todas las pruebas de §4 → `gotest` → **confirmar que fallan** → commit
  `test: link_patient en rojo` → implementar → verde → commit `feat: link_patient`.
- Cobertura: `LinkPatient`, `opLinkPatient` y los métodos generados de `LinkPatientArgs` y
  `LinkPatientResult` en 100.0%:

  ```bash
  go test -coverpkg=github.com/veltylabs/patient_directory -coverprofile=/tmp/cover.out ./tests/... \
    && go tool cover -func=/tmp/cover.out | grep -E "LinkPatient|linkPatient"
  # cada línea debe mostrar 100.0%
  ```

  La rama de carrera (`L3-ErrRutAlreadyExists`) se provoca sin exportar nada, con dos
  `LinkPatient` concurrentes del mismo RUT nuevo sobre la
  misma base (`sync.WaitGroup`, 50 repeticiones): ambas deben devolver **el mismo id** y quedar
  un solo paciente. Si la cobertura de esa rama no aparece en la corrida, repetir hasta 500;
  nunca agregar un gancho de prueba al paquete.
- Comentarios y documentación en español.

## 2. Etapa 1 — modelo y servicio

### 2.1 `model.go`

Agregar:

```go
var LinkPatientArgsModel = model.Definition{
	Name: "link_patient_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		// id lo genera quien llama (OF-5). Se usa solo si el RUT no existe.
		{Name: "id", Type: model.Text(), NotNull: true},
		{Name: "rut", Type: input.Rut(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 12}},
		{Name: "name", Type: input.Text(), NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}},
		{Name: "phone", Type: input.Phone(), NotNull: true},
		{Name: "email", Type: input.Email(), NotNull: true},
	},
}

var LinkPatientResultModel = model.Definition{
	Name: "link_patient_result",
	Fields: model.Fields{
		{Name: "id", Type: model.Text()},
	},
}
```

Regenerar `model_orm.go` con `ormc`.

### 2.2 `module.go` — método `LinkPatient`

```go
// LinkPatient devuelve el id del paciente de ese RUT. Si no existe, lo crea con
// los datos recibidos y el id de cmd. Si existe, devuelve su id guardado y NO
// modifica ni devuelve ningún otro dato: lo llama un invitado anónimo.
func (m *Module) LinkPatient(a LinkPatientArgs) (string, error)
```

Pasos exactos:

1. Si `a.TenantId`, `a.Id`, `a.Rut` o `a.Name` están vacíos → `ValidationError` con el error
   existente correspondiente (`ErrTenantRequired`, `ErrIdRequired`, `ErrRutRequired`,
   `ErrNameRequired`).
2. `existing, err := m.FindByRut(a.TenantId, a.Rut)`:
   - `err == nil` → `return existing.Id, nil` (aunque el paciente esté inactivo: el id sigue
     siendo el suyo; la reserva lo referencia igual que hoy).
   - `err` es `ErrNotFound` → seguir.
   - otro error → devolverlo.
3. `p, err := m.CreatePatient(Patient{Id: a.Id, TenantId: a.TenantId, Rut: a.Rut, Name: a.Name,
   Phone: a.Phone, Email: a.Email, IsActive: true})` → `return p.Id, err`. `CreatePatient` ya
   normaliza el RUT, publica `TopicPatientCreated` y es idempotente por id.
4. Carrera: si `CreatePatient` devuelve `ErrRutAlreadyExists` (otra pestaña creó el mismo RUT
   entre el paso 2 y el 3), repetir el paso 2 una vez y devolver ese id.

## 3. Etapa 2 — op

En `ops.go`:

```go
const OpLinkPatient = "link_patient"
```

Registrar en `MountOperations`:

```go
reg.Operation(OpLinkPatient, m.opLinkPatient).Requires("patient_link", model.Create).Accepts(&LinkPatientArgs{})
```

- Recurso propio `"patient_link"` (constante exportada `ResourcePatientLink = "patient_link"`,
  usada en la línea anterior): la app lo concede a la identidad anónima para la reserva
  online **sin** conceder `patient`. No marcar la op `.Public()` aquí: decidir quién la usa es
  de la app (seguridad por defecto).
- Handler: decode → `Validate(model.Create)` → `LinkPatient` → `ctx.Encode(&LinkPatientResult{Id: id})`.

## 4. Pruebas — trazabilidad diagrama → prueba (`tests/link_patient_test.go`)

| Rama | Prueba |
|---|---|
| `L1-sí` (cada campo) | `TestLinkPatient_MissingFields` (tabla: tenant, id, rut, nombre) |
| `L2-encontrado` | `TestLinkPatient_ExistingRutReturnsStoredIdUntouched` (RUT escrito con otro formato; teléfono y correo guardados no cambian; devuelve el id guardado, no el enviado) |
| `L2-encontrado` inactivo | `TestLinkPatient_InactivePatientStillLinks` |
| `L2-ErrNotFound` + `L3-ok` | `TestLinkPatient_NewRutCreatesWithCallerId` |
| `L2-otro error` | `TestLinkPatient_StorageErrorPropagates` (un `storage.Conn` falso declarado en `tests/` que envuelve `mem` y falla en las consultas; el error llega intacto) |
| `L3-ErrRutAlreadyExists` | `TestLinkPatient_ConcurrentSameRutYieldsOneId` |
| `L3-otro error` | `TestLinkPatient_IdTakenByOtherTenant` (`ErrIdTaken`), `TestLinkPatient_InvalidRutIsRejected` (`FindByRut` responde `ErrNotFound` y `CreatePatient` rechaza el RUT; nada creado) |
| reintento | `TestLinkPatient_RetryIsIdempotent` |
| aislamiento | `TestLinkPatient_TenantIsolation` |
| op: solo id | `TestOpLinkPatient_ResponseCarriesOnlyId` (sin claves `name`, `phone`, `email`, `rut`; mismo forma para nuevo y existente) |
| op: decode/validate | `TestOpLinkPatient_BadBodyAndInvalidArgsAre400` |
| op: permiso | `TestOpLinkPatient_RequiresPatientLinkCreate` |
| modelos generados | `TestLinkPatientModels_RoundTripAndValidate` |

Tras `ErrRutAlreadyExists`, la segunda búsqueda devuelve su resultado tal cual
(`return existing.Id, err`), sin una rama propia: no existe operación que borre pacientes.

## 5. Etapa 4 — documentación

- `README.md` y `docs/ARCHITECTURE.md`: agregar `link_patient` a la tabla de ops, con la regla
  "devuelve solo el id; nunca modifica un paciente existente".

## 6. Criterios de aceptación

- `gotest` en verde (vet, race, tests, wasm).
- Cada prueba de §4 existe con su nombre exacto; el criterio de cobertura de §1.1 se cumple.
- El historial del PR muestra el commit `test:` en rojo antes del `feat:`.
- `grep -n "OpLinkPatient" ops.go` muestra la constante y el registro con `ResourcePatientLink`.
- `grep -rn "map\[" --include=*.go .` → vacío.
- Antes de abrir el PR: `git diff --stat main` lista todos los archivos de la tabla de etapas,
  y no se borra ningún archivo que este plan no ordene borrar.

## 7. Tabla de etapas

| Etapa | Archivos |
|---|---|
| 1 | `model.go`, `model_orm.go` (regenerado), `module.go` |
| 2 | `ops.go` |
| 3 | `tests/link_patient_test.go` |
| 4 | `README.md`, `docs/ARCHITECTURE.md` |
