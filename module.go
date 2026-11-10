package patientdirectory

import (
	"strings"

	"github.com/veltylabs/patient_directory/kinds"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
)

type Deps struct {
	Publisher   events.Publisher // opcional — nil deshabilita la publicación silenciosamente
	TenantID    string
	IDs         model.IDGenerator // required: ids for patients created without one
	ValidateRUT func(string) (string, error)
}

type Module struct {
	db          *orm.DB
	store       *orm.Store
	pub         events.Publisher
	validateRUT func(string) (string, error)
	tenantID    string
}

func depErr(name string) error {
	return fmt.Err("patient_directory: Deps."+name, "is required")
}

// New conecta el módulo a un *orm.DB ya conectado; se asume que el esquema
// ya existe —consulte el submódulo migrate.
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.ValidateRUT == nil {
		return nil, depErr("ValidateRUT")
	}
	if deps.TenantID == "" {
		return nil, depErr("TenantID")
	}
	if deps.IDs == nil {
		return nil, depErr("IDs")
	}
	return &Module{
		db:          db,
		store:       orm.TenantStore(db, deps.TenantID, deps.IDs),
		pub:         deps.Publisher,
		validateRUT: deps.ValidateRUT,
		tenantID:    deps.TenantID,
	}, nil
}

// checkDocument normalizes and validates p's document by its type. An empty type means RUT (what the form sends
// today). Only a RUT is validated; other documents are stored as typed, upper-cased and trimmed.
func (m *Module) checkDocument(p *Patient) error {
	if p.DocumentType == "" {
		p.DocumentType = kinds.DocumentRUT
	}
	p.Rut = strings.TrimSpace(p.Rut)
	p.DocumentCountry = strings.ToUpper(strings.TrimSpace(p.DocumentCountry))
	switch p.DocumentType {
	case kinds.DocumentRUT:
		norm, err := m.validateRUT(p.Rut)
		if err != nil {
			return &model.FieldError{Field: "rut", Err: err}
		}
		p.Rut, p.DocumentCountry = norm, ""
	case kinds.DocumentPassport, kinds.DocumentForeignID:
		if p.Rut == "" {
			return &model.FieldError{Field: "rut", Err: ErrDocumentNumber}
		}
		p.Rut = strings.ToUpper(p.Rut)
	case kinds.DocumentNone:
		p.Rut, p.DocumentCountry = "", ""
	case kinds.DocumentUnverified:
		// stored as typed: the front desk fixes it
	default:
		return &model.FieldError{Field: "document_type", Err: ErrDocumentType}
	}
	return nil
}

func isPhone(s string) bool {
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		} else if r != '+' && r != ' ' && r != '-' && r != '(' && r != ')' {
			return false
		}
	}
	return digits >= 6
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
