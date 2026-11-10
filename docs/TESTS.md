# Pruebas

| Prueba | Regla |
|---|---|
| [TestTenant_ReadOtherTenantIsNotFound](../tests/crud_test.go) | El módulo del tenant A crea `p1`; el módulo del tenant B (misma base de datos) intenta leer `{id:"p1"}` y recibe `model.ErrorNotFound`. |
| [TestTenant_ClientTenantRejected](../tests/crud_test.go) | Intento de creación enviando `tenant_id:"b"` desde el cliente es rechazado con error de campo exclusivo del servidor (`ServerOnly`). |
| [TestCreate_DuplicateDocumentRejected](../tests/crud_test.go) | Creación de dos pacientes con el mismo RUT: el segundo es rechazado con error de campo en `rut` (`ErrRutAlreadyExists`). |
| [TestCreate_RetryIsIdempotent](../tests/crud_test.go) | Creación idéntica reintentada dos veces con el mismo ID: genera una sola fila en la base de datos y un solo evento `created`. |
| [TestUpdate_DeactivateIsAnUpdatedEvent](../tests/crud_test.go) | Desactivar (`is_active: false`) publica un solo evento `updated` cuyo `*crud.Change` trae `Fields` con `is_active`, y ningún otro. |
| [TestMount_MatchesDefinition](../tests/crud_test.go) | `crud.Mount` acepta el módulo: lo que monta `MountOperations` coincide con `Definition()`. |
| [TestCreate_PublishesChange](../tests/crud_test.go) | Crear un paciente publica un evento en el tema `crud.Topic("patient_directory","patient",Create)` con cuerpo `*crud.Change`. |
| [TestRule_NoAnswerIsRefused](../tests/crud_test.go) | Una regla que responde el valor cero (`crud.ErrNoAnswer`) es rechazada y no se guarda nada. |
| [TestUpdate_OnlyListedFields](../tests/crud_test.go) | Actualización enviando únicamente `["phone"]`: el campo `name` y el resto de los campos permanecen intactos. |
| [TestUpdate_DocumentRecheckedOnlyWhenTouched](../tests/crud_test.go) | Actualización de `["phone"]` en un paciente cuyo RUT almacenado no pasaría validaciones nuevas: la operación tiene éxito al no modificarse el documento. |
| [TestUpdate_DocumentTypeOnlyKeepsStoredRut](../tests/crud_test.go) | Paciente con pasaporte; se actualiza `["document_type"]` a RUT con el número almacenado válido: tiene éxito manteniendo el RUT previo. |
| [TestSearch_ByRutPhoneName](../tests/crud_test.go) | Tres pacientes registrados; búsqueda por RUT, por dígitos de teléfono y por fragmento de nombre retornan exactamente una coincidencia cada una. |
| [TestDelete_DoesNotExist](../tests/crud_test.go) | Invocación de la operación eliminada `patient_d` falla con operación desconocida al no permitirse `model.Delete`. |
| [TestImport_Idempotent](../tests/crud_test.go) | `ImportPatient` ejecutado dos veces con el mismo ID mantiene una sola fila con los campos de la segunda llamada. |
| [TestDemo_Valid](../tests/demo_test.go) | `seed.Demo()` se ejecuta sin error en WASM, su validación es limpia y expone al menos una pantalla. |
| [TestMigrate](../migrate/migrate_test.go) | Aplica migraciones DDL de esquemas de paciente sobre base de datos relacional. |
| [TestCreatePatientDefaultsToRUT](../tests/document_test.go) | La creación de paciente sin especificar tipo de documento asume `DocumentRUT` por defecto. |
| [TestInactivePatientSaves](../tests/document_test.go) | Un paciente inactivo (`is_active: false`) se almacena correctamente. |
| [TestNoDocument](../tests/document_test.go) | Paciente con tipo de documento `none` se registra sin requerir número de documento. |
| [TestPassportNeedsNumber](../tests/document_test.go) | Paciente con pasaporte requiere número de documento no vacío. |
| [TestPassportStoredAsTyped](../tests/document_test.go) | El número de pasaporte se preserva en mayúsculas sin guiones. |
| [TestSameDocumentTwice](../tests/document_test.go) | Reintento de creación de documento idéntico es rechazado dentro del mismo tenant. |
| [TestSamePassportOtherCountry](../tests/document_test.go) | Dos pasaportes con el mismo número pero distinto país emisor pueden coexistir en el mismo tenant. |
| [TestSexValues](../tests/document_test.go) | Validación de valores permitidos para el sexo del paciente (`m`, `f` o vacío). |
| [TestTwoWithoutDocument](../tests/document_test.go) | Dos pacientes distintos sin documento pueden coexistir en la misma clínica. |
| [TestFindPatient_AnyWayTheRutIsWritten](../tests/find_written_test.go) | `FindPatient` encuentra al paciente independientemente del formato con o sin puntos o guión del RUT. |
| [TestImportPatientCreatesThenUpdates](../tests/import_test.go) | `ImportPatient` crea si el ID no existe o actualiza si ya existe en el tenant. |
| [TestImportPatientUnverified](../tests/import_test.go) | `ImportPatient` permite registrar pacientes con documento por verificar. |
| [TestCreatePatient_AssignsIDAndNormalisesRut](../tests/patient_test.go) | Creación de paciente asigna identificador y normaliza el formato del RUT. |
| [TestCreatePatient_DuplicateRutSameTenant](../tests/patient_test.go) | Duplicado de RUT en el mismo tenant es rechazado. |
| [TestCreatePatient_IdempotentRetry](../tests/patient_test.go) | Reintento idempotente de creación retorna el mismo registro. |
| [TestCreatePatient_MissingIdRutOrName](../tests/patient_test.go) | Rechazo cuando faltan campos mandatorios como el nombre. |
| [TestCreatePatient_SameIdDifferentTenant](../tests/patient_test.go) | Mismo ID en diferente tenant genera conflicto de pertenencia. |
| [TestCreatePatient_SameRutDifferentTenant](../tests/patient_test.go) | Mismo RUT en distintos tenants coexiste de forma aislada. |
| [TestFindPatient_NormalisesInput](../tests/patient_test.go) | Búsqueda de paciente normaliza el texto de entrada. |
| [TestItemsDeclaredOnFields](../tests/patient_test.go) | Verificación de que `ItemLabel` y `ItemDescription` están configurados en el modelo. |
| [TestUpdatePatient_KeepingOwnRut](../tests/patient_test.go) | Actualización manteniendo el propio RUT no dispara conflicto de unicidad. |
| [TestUpdatePatient_RutTakenByAnother](../tests/patient_test.go) | Actualización cambiando a un RUT de otro paciente es rechazada. |
| [TestCreatePatient](../tests/readers_test.go) | Lector `CreatePatient` crea registro y retorna su ID asignado. |
| [TestCreatePatientDuplicate](../tests/readers_test.go) | Lector `CreatePatient` rechaza documento duplicado. |
| [TestFindPatient](../tests/readers_test.go) | Lector `FindPatient` retorna datos planos del paciente encontrado. |
| [TestFindPatientMissing](../tests/readers_test.go) | Lector `FindPatient` retorna `found=false` si el paciente no existe. |
| [TestGetPatient](../tests/readers_test.go) | Lector `GetPatient` recupera paciente por ID retornando tipos planos. |
| [TestGetPatientMissing](../tests/readers_test.go) | Lector `GetPatient` retorna `found=false` si no existe. |
| [TestScreenSave](../tests/screen_save_test.go) | Guardado desde pantalla en el navegador persiste correctamente vía bus. |
| [TestSeed_Load](../tests/seed_test.go) | Semillado carga pacientes de demostración sin error. |
| [TestUI_Screens](../tests/ui_view_browser_test.go) | Componente UI genera pantallas válidas con el llamador conectado. |
