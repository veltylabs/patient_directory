# Directorio de Pacientes (`patient_directory`)

Este repositorio posee los datos del registro de pacientes por tenant: únicamente identidad y detalles de contacto.

## Decisiones Arquitectónicas

1. **Identidad y alcance por Tenant:** El RUT es la identidad nacional primaria y es único por tenant.
2. **Unicidad en la Capa de Aplicación:** La unicidad del RUT por tenant se exige en el código de la aplicación (vía `ErrRutAlreadyExists`), no mediante una restricción de unicidad de columna única en la base de datos. NO establecer `Unique: true` en la definición del campo `rut` en los modelos.
3. **Sin eliminación física:** Los pacientes se desactivan (`is_active = false`), nunca se eliminan, para preservar la integridad referencial de atenciones clínicas e historial de reservas.
4. **Validación de RUT Inyectada:** La función de validación y normalización de RUT `ValidateRUT` se inyecta en `Deps`. No importar ni implementar lógica de dígito verificador de RUT en este repositorio.
5. **Sin campos clínicos:** No almacenar diagnósticos, consultas, recetas ni notas clínicas en este módulo. Los datos clínicos pertenecen a `clinical_encounter`.
6. **Sin lenguaje humano:** Este módulo no incluye cadenas de interfaz de usuario ni traducciones de lenguaje humano (por ejemplo, sin etiquetas ni frases descriptivas en la lógica central). La traducción y formateo de etiquetas es responsabilidad de las aplicaciones consumidoras.

## ui/, seed/ and web/ — the module's own view and demo

The whitelist and blacklist above apply to the **root domain package**. Three sub-packages are exempt, and only them:

- `ui/` (package `ui`, no build tag; `css.go`/`svg.go` tagged `!wasm`) may import `webtyp.com/layout/*`, `webtyp.com/components/*`, `dom`, `html`, `css`, `svg`, `widget`. It is the module's screen: `const ID`, `const Label` and `Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)`.
- `seed/` (package `seed`, no build tag) holds demo data: `Load(...) (Data, error)`, which writes through the module's own methods so every row is validated, and returns the rows it created so downstream demos can reference them.
- `web/` (package `main`, `web/client.go` tagged `wasm`) is the runnable demo and may additionally import the concrete `webtyp.com/storage/mem`, `webtyp.com/router/loopback`, `webtyp.com/events/mock`, `webtyp.com/unixid` and `webtyp.com/auth/trusted_ip` (for `ValidateRUT`).

**Dependencies between modules form a DAG.** The root domain package still imports no sibling module. `ui/`, `seed/` and `web/` may import the domain, `ui/` and `seed/` packages of **upstream** modules only. Current graph: `device_manager`, `item_catalog`, `patient_directory`, `business_calendar` (leaves) ← `staff_manager` ← `appointment_booking` ← `clinical_encounter` (`appointment_booking` also depends on `item_catalog`, `patient_directory`, `business_calendar`; `clinical_encounter` also on `patient_directory` and `staff_manager`). A screen that mixes modules lives in the most-downstream module it touches.
