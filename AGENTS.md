# Directorio de Pacientes (`patient_directory`)

Este repositorio posee los datos del registro de pacientes por tenant: únicamente identidad y detalles de contacto.

## Decisiones Arquitectónicas

1. **Identidad y alcance por Tenant:** El RUT es la identidad nacional primaria y es único por tenant.
2. **Unicidad en la Capa de Aplicación:** La unicidad del RUT por tenant se exige en el código de la aplicación (vía `ErrRutAlreadyExists`), no mediante una restricción de unicidad de columna única en la base de datos. NO establecer `Unique: true` en la definición del campo `rut` en los modelos.
3. **Sin eliminación física:** Los pacientes se desactivan (`is_active = false`), nunca se eliminan, para preservar la integridad referencial de atenciones clínicas e historial de reservas.
4. **Validación de RUT Inyectada:** La función de validación y normalización de RUT `ValidateRUT` se inyecta en `Deps`. No importar ni implementar lógica de dígito verificador de RUT en este repositorio.
5. **Sin campos clínicos:** No almacenar diagnósticos, consultas, recetas ni notas clínicas en este módulo. Los datos clínicos pertenecen a `clinical_encounter`.
6. **Sin lenguaje humano:** Este módulo no incluye cadenas de interfaz de usuario ni traducciones de lenguaje humano (por ejemplo, sin etiquetas ni frases descriptivas en la lógica central). La traducción y formateo de etiquetas es responsabilidad de las aplicaciones consumidoras.
