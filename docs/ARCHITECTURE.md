# Architecture & Design Decisions

## Domain Scope Boundary

`patient_directory` owns patient **identity** and **contact details**:
- Identity: internal ID, national identifier (RUT), full name, birthdate.
- Contact: phone number, email address, physical address.
- Status: active flag (`is_active`), last updated timestamp (`updated_at`).

It explicitly does **not** own:
- Clinical encounters, diagnoses, notes, or prescriptions (owned by `clinical_encounter`).
- Appointments or scheduling state (owned by `appointment_booking`).
- Staff or practitioner user accounts (owned by `staff_manager`).

## Core Architectural Decisions

1. **RUT Identity & Tenant Scope:**
   - Chilean clinics identify patients by national RUT.
   - Uniqueness is scoped per tenant — the same RUT may exist in different clinic tenants independently.

2. **Application-Layer Uniqueness Enforcement:**
   - Single-column unique constraints on `rut` would fail across multi-tenant databases.
   - `patient_directory` checks RUT existence in `CreatePatient` and `UpdatePatient` and returns `ErrRutAlreadyExists`. `PatientModel` carries no `Unique: true` flag.

3. **Deactivation over Deletion:**
   - Patients are never deleted. `DeactivatePatient` sets `is_active = false`.
   - Historical records and bookings referencing a patient ID must always resolve.

4. **Injected RUT Validation:**
   - `Deps.ValidateRUT` function is injected into `patientdirectory.New`.
   - The module depends on no specific RUT library and implements no check-digit calculations.

5. **No UI Strings / No Spanish Translations:**
   - Handlers and models return standard domain constants and errors.
   - Translating labels for end users is the responsibility of consuming web/mobile interfaces.

## Composition Root Wiring Example

Wiring `patient_directory` into `appointment_booking`:

```go
package main

import (
	"github.com/veltylabs/patient_directory"
	// "github.com/veltylabs/appointment_booking"
)

func main() {
	// 1. Initialize patient_directory module
	patientMod, err := patientdirectory.New(db, patientdirectory.Deps{
		IDs: idGen,
		ValidateRUT: myRutValidator,
		Publisher: eventPub,
	})
	if err != nil {
		log.Fatalf("failed to init patient_directory: %v", err)
	}

	// 2. Wire directly into appointment_booking without adapter (structurally satisfies ClientExists)
	/*
	bookingMod, err := appointmentbooking.New(db, appointmentbooking.Deps{
		Directory: patientMod, // satisfies DirectoryReader interface
	})
	*/
	_ = patientMod
}
```
