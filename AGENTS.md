# Patient Directory (`patient_directory`)

This repository owns tenant-scoped patient registry data: identity and contact details only.

## Architectural Decisions

1. **Identity & Tenant Scope:** The RUT is the primary national identity, and it is unique per tenant.
2. **Application-Layer Uniqueness:** Uniqueness of RUT per tenant is enforced in application code (against `ErrRutAlreadyExists`), not via single-column DB unique constraint. Do NOT set `Unique: true` on `rut` in model definitions.
3. **No Hard Delete:** Patients are deactivated (`is_active = false`), never deleted, to preserve integrity of historical clinical encounters and appointment bookings.
4. **Injected RUT Validation:** RUT validation and normalization function `ValidateRUT` is injected in `Deps`. Do not import or implement RUT check-digit logic in this repository.
5. **No Clinical Fields:** Do not store diagnoses, visits, prescriptions, or clinical notes in this module. Clinical data belongs to `clinical_encounter`.
6. **No Human Language:** This module ships no UI strings or human language translations (e.g. no Spanish labels or status words). Translating and formatting labels is the responsibility of consuming applications.
