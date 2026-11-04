# Arquitectura: patient_directory

El módulo `patient_directory` maneja el directorio de pacientes: datos personales y de contacto.

## Regla de documento único

Un paciente es identificado principalmente por su documento. El campo `document_type` puede ser:
- `rut` (por defecto): RUT Chileno, se valida su dígito verificador.
- `passport`: Pasaporte de cualquier país. Se valida que no esté vacío.
- `foreign_id`: Cédula de identidad extranjera.
- `none`: Paciente sin documento al momento del registro.
- `unverified`: Documento ingresado pero no verificado o inválido (debe corregirse en recepción).

La base de datos restringe los documentos para que no haya duplicados (RUT/pasaporte/cédula extranjera) en un mismo tenant (a menos que el tipo sea `unverified` o el paciente no tenga documento `none`).

### ¿Por qué el teléfono no es un identificador?

Las familias a menudo comparten un solo número de teléfono para distintos miembros de la familia (ej. niños o ancianos dependientes). Usar el teléfono como identificador único causaría conflictos.
