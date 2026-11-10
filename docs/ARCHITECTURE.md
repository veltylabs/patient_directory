# Arquitectura de `patient_directory`

## 1. Visión general en un minuto

`patient_directory` administra la identidad y datos de contacto de las personas atendidas en un establecimiento de salud. El aislamiento entre clínicas es estricto por tenant: una misma persona o documento puede existir en clínicas independientes sin interferir entre sí.

El diseño desacopla la persistencia de las reglas clínicas:
- La **persistencia** (transacciones atómicas, filtrado por tenant, lectura de campos, paginación e idempotencia) está delegada al framework mediante [`webtyp.com/crud`](https://github.com/webtyp/crud) y [`orm.TenantStore`](https://github.com/webtyp/orm), gobernados por `PatientModel.Allow` (`Create | Read | Update`).
- Las **reglas de negocio** residen en tres archivos `rule_patient_*.go` (los crea `definitionc` una vez): crear (`Create`: `checkDocument` y `sameDocument`), actualizar (`Update`) y buscar (`Search`). Los eventos no los escribe el módulo: `crud` los publica tras el commit.

Los datos clínicos (fichas, diagnósticos, atenciones) pertenecen exclusivamente a `clinical_encounter`.

## 2. Piezas y estructura

| Archivo / Componente | Responsabilidad | Propietario |
|---|---|---|
| `definition.go` | Declaración del módulo: `Definition`, `PatientModel` con `Allow` y `Rules`, y las constantes de errores de dominio. | Dominio |
| `definition_gen.go` | Generado por `definitionc`: tipos, esquema y `MountOperations`/`Definition()` del módulo. No se edita. | Generado |
| `rule_patient_create.go`, `rule_patient_update.go`, `rule_patient_search.go` | Reglas de crear (validación y unicidad de documento por clínica), actualizar y buscar. | Dominio |
| `store.go` | `StorePatient()`: el almacén de pacientes sellado por tenant. | Módulo |
| `module.go` | Constructor `New`, inyección de dependencias (`TenantID`, `IDs`, `Publisher`, `ValidateRUT`) y creación de `TenantStore`. | Módulo |
| `readers.go` | Lectores de interoperabilidad entre módulos con tipos planos (`GetPatient`, `FindPatient`, `CreatePatient`). | Integración |
| `import.go` | Carga e importación idempotente de registros históricos (`ImportPatient`). | Dominio |
| `view.go` | Configuración de vistas paginadas en el navegador mediante `view.NewCallerLister`. | UI Browser |
| `migrate/migrate.go` | Definición de pasos DDL y migración del esquema en la base de datos. | Almacenamiento |
| `seed/demo.go` | Datos iniciales para demostraciones locales y pruebas aisladas en memoria. | Herramientas |

## 3. Flujos paso a paso

### Registro de paciente nuevo

Narra la creación validada y su notificación atómica. Diagrama detallado en [docs/diagrams/patient_flow.md](diagrams/patient_flow.md).

- **Camino exitoso:**
  1. El cliente invoca `patient_directory.patient.create` (`POST`), o un módulo llama a `CreatePatient`.
  2. `crud` valida la conformidad del esquema y campos no nulos.
  3. La regla `Create` normaliza el RUT o valida pasaporte/cédula extranjera.
  4. `sameDocument` comprueba en la base de datos que el documento no esté tomado en la misma clínica.
  5. `crud` inserta el registro, asigna ID generado y marcas de tiempo del servidor.
  6. `crud` publica `patient_directory.patient.created` (cuerpo `*crud.Change`) tras el commit.
- **Camino de fallo:**
  - Si el documento tiene formato inválido, retorna inmediatamente `model.FieldError` (400).
  - Si el documento ya existe en el tenant, retorna `ErrRutAlreadyExists` (409) y no se almacena fila alguna.

### Modificación y desactivación de paciente

- **Camino exitoso:**
  1. El cliente invoca `patient_directory.patient.update` (`PATCH`) enviando los campos modificados (por ejemplo `["phone"]` o `["is_active"]`).
  2. La regla `Update` solo revalida el documento si la modificación altera campos de identidad (`rut`, `document_type`, `document_country`).
  3. `crud` publica siempre `patient_directory.patient.updated`; si se desactivó (`is_active = false`), el `Fields` del `*crud.Change` incluye `is_active`.
- **Camino de fallo:**
  - Si se intenta modificar un campo exclusivo del servidor (`ServerOnly`, como `tenant_id`), la llamada es rechazada antes de tocar la base de datos.

## 4. Contratos y límites de seguridad

1. **Aislamiento Multitenant Absoluto:**
   - Ninguna consulta ni mutación puede omitir el `tenant_id`. [`orm.TenantStore`](https://github.com/webtyp/orm) inyecta de forma forzosa la condición de tenant en cada sentencia generada.
   - Los campos `tenant_id` y `updated_at` están marcados como `ServerOnly: true`: cualquier valor enviado por el cliente es descartado o rechazado.

2. **Sin Eliminación Física:**
   - La operación de borrado físico (`model.Delete`) está expresamente deshabilitada en el esquema.
   - Los pacientes solo se desactivan mediante `is_active = false`, preservando la integridad referencial de atenciones pasadas.

3. **Interoperabilidad con Tipos Planos:**
   - Los lectores de integración (`readers.go`) intercambian tipos escalares de Go (`string`, `bool`, `error`), impidiendo acoplamientos rígidos entre módulos del ecosistema.
