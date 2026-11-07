package patientdirectory

import (
	"github.com/veltylabs/patient_directory/kinds"
	"strings"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage"
	"webtyp.com/time"
)

type Deps struct {
	Publisher events.Publisher // opcional — nil deshabilita la publicación silenciosamente
	// TenantID identifica esta instalación — el valor por defecto que opListPatients usa
	// cuando quien llama no envía tenant_id (todo listado respaldado por
	// crudview lo hace así).
	TenantID string
	// ValidateRUT normaliza y valida un RUT, retornando la forma canónica.
	// Requerido. Inyectado, nunca importado: este módulo no debe depender de ninguna
	// implementación particular de RUT, ni incluir su propio cálculo de dígito verificador.
	ValidateRUT func(string) (string, error)
}

type Module struct {
	db          *orm.DB
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
	return &Module{
		db:          db,
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

func (m *Module) findSameDocument(p Patient) error {
	if p.Rut == "" || p.DocumentType == kinds.DocumentUnverified {
		return nil
	}
	var existing Patient
	qb := m.db.Query(&existing).
		Where(Patient_.TenantId).Eq(p.TenantId).
		Where(Patient_.DocumentType).Eq(p.DocumentType).
		Where(Patient_.Rut).Eq(p.Rut)

	results, err := ReadAllPatient(qb)
	if err == nil {
		for _, r := range results {
			if r.Id != p.Id && r.DocumentCountry == p.DocumentCountry {
				return ErrRutAlreadyExists
			}
		}
	}
	if err != nil && !orm.IsNotFound(err) {
		return err
	}
	return nil
}

func (m *Module) publish(topic string, payload model.Model) {
	if m.pub != nil {
		m.pub.Publish(events.Event{
			Topic:   topic,
			Payload: payload,
		})
	}
}

func (m *Module) createPatient(p Patient) (Patient, error) {
	if p.Id == "" {
		return Patient{}, &model.FieldError{Field: "id", Err: ErrIdRequired}
	}
	if p.TenantId == "" {
		return Patient{}, &model.FieldError{Field: "tenant_id", Err: ErrTenantRequired}
	}
	if p.Name == "" {
		return Patient{}, &model.FieldError{Field: "name", Err: ErrNameRequired}
	}

	if err := m.checkDocument(&p); err != nil {
		return Patient{}, err
	}

	if p.Sex != "" && p.Sex != "m" && p.Sex != "f" {
		return Patient{}, &model.FieldError{Field: "sex", Err: ErrSex}
	}

	// Reintento idempotente: buscar por Id sin importar el tenant
	var existingById Patient
	qbId := m.db.Query(&existingById).Where(Patient_.Id).Eq(p.Id)
	_, errId := ReadOnePatient(qbId, &existingById)
	if errId == nil {
		if existingById.TenantId == p.TenantId {
			// Es la misma operación repetida, devolverla tal como está guardada
			return existingById, nil
		}
		// Si el paciente existe con otro tenant
		return Patient{}, ErrIdTaken
	} else if !orm.IsNotFound(errId) {
		return Patient{}, errId
	}

	// Verificar duplicados de documento
	if err := m.findSameDocument(p); err != nil {
		return Patient{}, err
	}

	p.UpdatedAt = time.NowNano()

	if err := m.db.Create(&p); err != nil {
		return Patient{}, err
	}

	m.publish(TopicPatientCreated, &p)
	return p, nil
}

func (m *Module) updatePatient(p Patient) (Patient, error) {
	if p.Id == "" {
		return Patient{}, ErrNotFound
	}
	if p.TenantId == "" {
		return Patient{}, &model.FieldError{Field: "tenant_id", Err: ErrTenantRequired}
	}
	if p.Name == "" {
		return Patient{}, &model.FieldError{Field: "name", Err: ErrNameRequired}
	}

	// Verificar que el paciente existente pertenezca al tenant
	curr, err := m.getPatient(p.TenantId, p.Id)
	if err != nil {
		return Patient{}, err
	}
	_ = curr

	if err := m.checkDocument(&p); err != nil {
		return Patient{}, err
	}

	if p.Sex != "" && p.Sex != "m" && p.Sex != "f" {
		return Patient{}, &model.FieldError{Field: "sex", Err: ErrSex}
	}

	// Verificar duplicados de documento
	if err := m.findSameDocument(p); err != nil {
		return Patient{}, err
	}

	p.UpdatedAt = time.NowNano()

	if err := m.db.Update(&p, storage.Eq(Patient_.Id, p.Id), storage.Eq(Patient_.TenantId, p.TenantId)); err != nil {
		return Patient{}, err
	}

	m.publish(TopicPatientUpdated, &p)
	return p, nil
}

func (m *Module) getPatient(tenantID, id string) (Patient, error) {
	if tenantID == "" || id == "" {
		return Patient{}, ErrNotFound
	}
	var p Patient
	qb := m.db.Query(&p).
		Where(Patient_.TenantId).Eq(tenantID).
		Where(Patient_.Id).Eq(id)
	_, err := ReadOnePatient(qb, &p)
	if orm.IsNotFound(err) {
		return Patient{}, ErrNotFound
	}
	if err != nil {
		return Patient{}, err
	}
	return p, nil
}

func (m *Module) findByDocument(tenantID, rut string) (Patient, error) {
	if tenantID == "" || rut == "" {
		return Patient{}, ErrNotFound
	}
	normRut, err := m.validateRUT(rut)
	if err != nil {
		return Patient{}, ErrNotFound
	}
	var p Patient
	qb := m.db.Query(&p).
		Where(Patient_.TenantId).Eq(tenantID).
		Where(Patient_.DocumentType).Eq(kinds.DocumentRUT).
		Where(Patient_.Rut).Eq(normRut)
	results, err := ReadAllPatient(qb)
	if err == nil {
		for _, r := range results {
			if r.DocumentCountry == "" {
				return *r, nil
			}
		}
	}
	if err != nil && !orm.IsNotFound(err) {
		return Patient{}, err
	}
	return Patient{}, ErrNotFound
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

func (m *Module) listPatients(tenantID, search, after string, limit int64) ([]Patient, error) {
	if tenantID == "" {
		return []Patient{}, nil
	}
	var p Patient
	qb := m.db.Query(&p).Where(Patient_.TenantId).Eq(tenantID)
	switch norm, err := input.NormalizeRUT(search); {
	case search == "":
	case err == nil:
		qb = qb.Where(Patient_.DocumentType).Eq(kinds.DocumentRUT).Where(Patient_.Rut).In([]string{norm, strings.ReplaceAll(norm, "-", "")})
	case isPhone(search):
		qb = qb.Where(Patient_.Phone).Like("%" + digitsOnly(search) + "%")
	default:
		qb = qb.Where(Patient_.Name).Like("%" + strings.ToLower(search) + "%")
	}
	if after != "" {
		qb = qb.Where(Patient_.Id).Gt(after)
	}
	qb = qb.OrderBy(Patient_.Id).Asc()
	if limit > 0 {
		qb = qb.Limit(int(limit))
	}

	results, err := ReadAllPatient(qb)
	if err != nil {
		return nil, err
	}
	out := make([]Patient, len(results))
	for i, r := range results {
		out[i] = *r
	}
	return out, nil
}

func (m *Module) deactivatePatient(tenantID, id string) error {
	p, err := m.getPatient(tenantID, id)
	if err != nil {
		return err
	}
	p.IsActive = false
	p.UpdatedAt = time.NowNano()

	if err := m.db.Update(&p, storage.Eq(Patient_.Id, p.Id), storage.Eq(Patient_.TenantId, p.TenantId)); err != nil {
		return err
	}
	m.publish(TopicPatientDeactivated, &p)
	return nil
}
