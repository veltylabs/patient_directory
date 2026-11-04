package kinds

import (
	"webtyp.com/fmt"
	"webtyp.com/input"
)

const (
	DocumentRUT        = "rut"        // Chilean RUT: check digit validated; unique per tenant
	DocumentPassport   = "passport"   // any country's passport, stored as typed
	DocumentForeignID  = "foreign_id" // another country's id card, stored as typed
	DocumentNone       = "none"       // the person has no document at hand
	DocumentUnverified = "unverified" // typed but invalid; the front desk must fix it
)

// DocumentType: closed options for patient.document_type. Example: in := kinds.DocumentType()
func DocumentType() input.Input {
	return input.Radio(
		fmt.KeyValue{Key: DocumentRUT, Value: "RUT"},
		fmt.KeyValue{Key: DocumentPassport, Value: "Passport"},
		fmt.KeyValue{Key: DocumentForeignID, Value: "Foreign id card"},
		fmt.KeyValue{Key: DocumentNone, Value: "No document"},
		fmt.KeyValue{Key: DocumentUnverified, Value: "To verify"},
	)
}
