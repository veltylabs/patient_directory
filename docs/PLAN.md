---
PLAN: "feat!: el id del paciente lo genera quien llama — CreatePatient exige Id, reintento idempotente, upsert por existencia; se elimina Deps.IDs"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 2342664417546434681
PR: https://github.com/veltylabs/patient_directory/pull/5
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `patient_directory`: IDs generados por quien llama (offline-first)

## 0. Contexto (leer primero)

Una ola offline-first (master plan:
<https://github.com/veltylabs/mjosefa-cms/blob/main/docs/OFFLINE_FIRST_MASTER_PLAN.md>) hace
que la recepción de una clínica pueda **registrar un paciente nuevo sin servidor**. El navegador
ejecuta este mismo módulo sobre su base local, guarda la operación en una cola (outbox) y la
reenvía al servidor cuando vuelve la conexión. Para que eso funcione:

- **El id lo genera quien llama** (decisión OF-5 del master). El paciente creado sin conexión
  tiene que llegar al servidor **con el mismo id**: la reserva hecha en el mismo momento ya lo
  referencia (`reservation.client_id`).
- **Reenviar la misma operación no puede duplicar ni fallar.** Si el servidor aplicó la
  operación pero la respuesta se perdió, la cola la reenvía. Mismo id = mismo paciente = éxito.
- **Sin retrocompatibilidad** (OF-6): las firmas cambian y no queda ningún camino viejo.

Estado actual (verificado en `main`, v0.0.54):

- `CreatePatient` usa `p.Id` si viene y, si no, genera uno con `m.ids.NewID()` (`module.go`).
  Son **dos caminos** para lo mismo.
- `opUpsertPatient` (`ops.go`) decide crear o actualizar según `p.Id == ""`. Con IDs generados
  por el cliente, un paciente nuevo **ya trae** id, entra por `UpdatePatient` y falla con
  `ErrNotFound`. El formulario de `crudview` ya asigna el id en el cliente al enviar.
- Reenviar `CreatePatient` con el mismo id falla con `ErrRutAlreadyExists` (el RUT ya está
  guardado, por el mismo paciente).

## Design gate

### 1. Prior art
- **Replicache / Zero**: las mutaciones crean entidades con ids generados en el cliente
  (`nanoid`); el servidor las acepta tal cual y la mutación es idempotente.
- **Stripe `Idempotency-Key`**: repetir la misma petición devuelve el mismo resultado sin crearlo
  dos veces.
- **HTTP `PUT /patients/{id}`** (RFC 9110, PUT es idempotente): el cliente nombra el recurso; si
  existe se reemplaza y si no se crea. Es la semántica del upsert de este plan.

### 2. Prueba del nombre para un novato
`CreatePatient(p)` con `p.Id` obligatorio: "crea este paciente con este id". `upsert_patient`:
"guarda este paciente: créalo si no existe, actualízalo si existe". Los nombres no cambian.

### 3. Libro de complejidad
```
Conceptos que aprender              −1 (Deps.IDs)
Archivos que tocar para hacer X      0
Líneas en el sitio de llamada        −1 (ya no se pasa IDs a New)
Formas de hacer lo mismo             −1 (CreatePatient ya no tiene el camino "genero yo el id")
```

### 4. Dónde va
En este módulo: es su regla de identidad. Generar el id es responsabilidad de quien llama: la UI
(`crudview` ya lo hace con el `IDGenerator` que recibe `ui.Browser`), el seed y la importación de
datos.

### 5. Qué borra
`Deps.IDs`, `Module.ids`, la comprobación `deps.IDs == nil` de `New`, la línea
`p.Id = m.ids.NewID()` y la decisión por `p.Id == ""` de `opUpsertPatient`.

## 1. Cambios (normativos)

### 1.1 `module.go`
- `Deps` pierde `IDs`; `Module` pierde `ids`; `New` deja de exigirlo. Actualiza el comentario de
  `Deps` (el módulo ya no genera ids: los recibe).
- Nuevo error de dominio en `model.go`, junto a los existentes:
  `ErrIdRequired domainError = "patient id is required"` y
  `ErrIdTaken domainError = "patient id belongs to another tenant"`.
- `CreatePatient(p Patient) (Patient, error)`, en este orden:
  1. `p.Id == ""` → `ValidationError{Err: ErrIdRequired}`.
  2. Validaciones existentes (tenant, rut, nombre, `validateRUT`).
  3. **Reintento**: si existe un paciente con ese `Id` (buscar **solo por id**, sin tenant):
     - del mismo tenant → devolverlo tal como está guardado, sin escribir ni publicar evento
       (es la misma operación repetida);
     - de otro tenant → `ErrIdTaken`.
  4. Regla del RUT existente (`ErrRutAlreadyExists` si otro paciente, con **otro** id, ya tiene
     ese RUT en el tenant).
  5. `UpdatedAt`, `db.Create`, evento `TopicPatientCreated`, como hoy.
- `UpdatePatient` no cambia.

### 1.2 `ops.go`
`opUpsertPatient`:
1. Decodificar. `p.Id == ""` → 400 con `ErrIdRequired`.
2. Si existe un paciente con ese id en `p.TenantId` → `UpdatePatient(p)`; si no → `CreatePatient(p)`.
   Para saber si existe usar `GetPatient(p.TenantId, p.Id)` y distinguir `ErrNotFound` por
   aserción de tipo (`err.(domainError)`), como hace el resto del archivo (nunca `==` entre
   interfaces: en TinyGo arrastra `reflectlite`).
3. `handleErr`: `ErrIdTaken` → 409; `ValidationError` ya da 400.

### 1.3 `seed/`
`Load` asigna ids **determinísticos** a cada paciente de demo (`"demo-patient-1"`, `"-2"`, …,
como constantes del paquete `seed`), porque el módulo ya no los genera. Los módulos posteriores
siguen leyendo `Data.Patients[i].Id`.

### 1.4 `web/client.go`
Quitar `IDs` de `patientdirectory.Deps`. El `ids` de `unixid.NewUnixID()` sigue pasándose a
`ui.Browser` (el formulario lo usa para asignar el id al guardar).

### 1.5 Documentación
`README.md` (ejemplo de `New`), `docs/ARCHITECTURE.md` (ejemplo de raíz de composición) y
`AGENTS.md` si menciona `Deps.IDs`: quitar `IDs` y agregar una línea: "El id del paciente lo
genera quien llama (UI, seed, importación); `CreatePatient` lo exige y repetir la misma creación
devuelve el paciente guardado."

## 2. Tests (`tests/`, paquete externo; `gotest`)

Actualizar los tests existentes a la nueva firma (todo `CreatePatient` pasa un `Id`; los
`setup` dejan de pasar `IDs`; `tests/idgen_test.go` se borra si queda sin uso). Agregar:

1. `CreatePatient` sin `Id` → `ValidationError` con `ErrIdRequired`.
2. Reintento: `CreatePatient(p)` dos veces con el mismo id y RUT → la segunda devuelve el mismo
   paciente, sin error; hay **una** fila; se publicó **un** evento (contar con el broker de test).
3. Mismo id en otro tenant → `ErrIdTaken`.
4. Otro id con el mismo RUT en el mismo tenant → `ErrRutAlreadyExists` (la regla se mantiene).
5. `opUpsertPatient` con un id nuevo → crea (antes fallaba con 404); con un id existente →
   actualiza; sin id → 400.
6. `seed.Load` dos veces sobre la misma base → sin error y sin duplicados (consecuencia del caso 2).

## 3. Reglas de código (no negociables)
- Paquete raíz compila a TinyGo WASM: `webtyp.com/fmt` para errores y formato; nada de `errors`,
  `strconv`, `strings` de la biblioteca estándar.
- Errores de dominio como constantes `domainError`; se comparan por aserción de tipo, nunca con `==`.
- Tests solo en `tests/`; nunca exportar un símbolo para un test.

## 4. Criterios de aceptación
- `gotest ./...` en verde (stdlib y wasm).
- `grep -rn "Deps.IDs\|m.ids\|IDs:" --include=*.go . | grep -v "ui/\|web/client.go"` → vacío
  (en `web/client.go` solo puede quedar el `ids` que se pasa a `ui.Browser`).
- `grep -n "NewID" module.go` → vacío.

| Etapa | Archivos | Listo cuando |
|---|---|---|
| 1 | `module.go`, `model.go` | `CreatePatient` exige id y es idempotente; sin `Deps.IDs` |
| 2 | `ops.go` | upsert por existencia |
| 3 | `seed/*.go`, `web/client.go` | demo funciona sin `IDs` en el módulo |
| 4 | `tests/*.go` | 6 casos nuevos + existentes en verde |
| 5 | `README.md`, `docs/ARCHITECTURE.md`, `AGENTS.md` | sin `IDs` en los ejemplos |

**Consumidores (no son trabajo de este plan):** `appointment_booking`, `clinical_encounter` y
`mjosefa-cms` pasan `IDs` a `patientdirectory.Deps`; se actualizan cuando suban de versión
(etapa I1 del master).
## Executor notes
Ejecuté todo el plan exitosamente. El único punto de atención es que en `web/client.go` el variable `ids` se sigue inicializando y pasándose a `ui.Browser`, lo cual es correcto de acuerdo al plan ya que UI lo necesita para asignar el id al guardar.
