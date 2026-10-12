# `link_patient` — vincular o registrar sin exponer datos

Cada rombo lleva un id; [`docs/PLAN.md`](../PLAN.md) §4 asigna a cada rama su prueba.

```mermaid
flowchart TD
    A[LinkPatient args] --> L1{L1: ¿falta tenant, id,<br>rut o nombre?}
    L1 -- sí --> E0[ValidationError del campo]
    L1 -- no --> F[FindByRut tenant, rut]
    F --> L2{L2: ¿resultado?}
    L2 -- encontrado --> RET[devolver id guardado<br>sin tocar nada]
    L2 -- ErrNotFound --> C[CreatePatient con id y datos recibidos]
    L2 -- otro error --> E1[devolver el error]
    C --> L3{L3: ¿resultado?}
    L3 -- ok --> NEW[devolver id del creado]
    L3 -- ErrRutAlreadyExists<br>carrera --> F2[FindByRut otra vez<br>y devolver su resultado tal cual]
    L3 -- otro error --> E3[devolver el error]
```

Respuesta de la op: **siempre** `{"id": "..."}` y nada más, sea nuevo o existente.
