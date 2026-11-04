# patient_directory

Directorio de pacientes: datos personales, información de contacto e identificación.
Identidad de una persona; los datos clínicos (diagnósticos, atenciones) no van aquí.

## Reglas de documento

El campo `rut` almacena el número de documento de cualquier tipo (RUT, pasaporte, etc.)
Solo el RUT es validado mediante dígito verificador. El pasaporte y cédula extranjera se guardan tal como se ingresan, en mayúsculas.

## Uso

### Importar pacientes históricos

`ImportPatient` permite guardar un paciente de un sistema anterior con el mismo ID, o actualizarlo si ya existe:

```go
p, err := dir.ImportPatient(Patient{
    Id:           "1640011384447734400",
    TenantId:     t,
    DocumentType: kinds.DocumentRUT,
    Rut:          "12345678-5",
    Name:         "Ana",
})
```

## Arquitectura
Consulte `docs/ARCHITECTURE.md` para detalles de diseño y relaciones con otros módulos.
