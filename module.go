package patientdirectory

import (
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage"
	"webtyp.com/time"
)

type Deps struct {
	IDs       model.IDGenerator // required — the module never builds its own
	Publisher events.Publisher  // optional — nil disables publishing silently
	// ValidateRUT normalises and validates a RUT, returning the canonical form.
	// Required. Injected, never imported: this module must not depend on any
	// particular RUT implementation, and must not ship its own check digit.
	ValidateRUT func(string) (string, error)
}

type Module struct {
	db          *orm.DB
	ids         model.IDGenerator
	pub         events.Publisher
	validateRUT func(string) (string, error)
}

// New connects the module to an already-connected *orm.DB; the schema is
// assumed to already exist — see the migrate subpackage.
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("patient_directory: Deps.IDs is required")
	}
	if deps.ValidateRUT == nil {
		return nil, fmt.Err("patient_directory: Deps.ValidateRUT is required")
	}
	return &Module{
		db:          db,
		ids:         deps.IDs,
		pub:         deps.Publisher,
		validateRUT: deps.ValidateRUT,
	}, nil
}

type PatientFilter struct {
	ActiveOnly bool
	Limit      int64
	Offset     int64
}

func (m *Module) publish(topic string, payload model.Model) {
	if m.pub != nil {
		m.pub.Publish(events.Event{
			Topic:   topic,
			Payload: payload,
		})
	}
}

func (m *Module) CreatePatient(p Patient) (Patient, error) {
	if p.TenantId == "" {
		return Patient{}, ValidationError{Err: ErrTenantRequired}
	}
	if p.Rut == "" {
		return Patient{}, ValidationError{Err: ErrRutRequired}
	}
	if p.Name == "" {
		return Patient{}, ValidationError{Err: ErrNameRequired}
	}

	normRut, err := m.validateRUT(p.Rut)
	if err != nil {
		return Patient{}, ValidationError{Err: err}
	}
	p.Rut = normRut

	// Check if RUT already exists for this tenant
	existing, err := m.FindByRut(p.TenantId, p.Rut)
	if err == nil && existing.Id != "" {
		return Patient{}, ErrRutAlreadyExists
	} else if err != nil && err != ErrNotFound {
		return Patient{}, err
	}

	if p.Id == "" {
		p.Id = m.ids.NewID()
	}
	p.UpdatedAt = time.Now()

	if err := m.db.Create(&p); err != nil {
		return Patient{}, err
	}

	m.publish(TopicPatientCreated, &p)
	return p, nil
}

func (m *Module) UpdatePatient(p Patient) (Patient, error) {
	if p.Id == "" {
		return Patient{}, ErrNotFound
	}
	if p.TenantId == "" {
		return Patient{}, ValidationError{Err: ErrTenantRequired}
	}
	if p.Rut == "" {
		return Patient{}, ValidationError{Err: ErrRutRequired}
	}
	if p.Name == "" {
		return Patient{}, ValidationError{Err: ErrNameRequired}
	}

	// Verify existing patient exists in tenant
	curr, err := m.GetPatient(p.TenantId, p.Id)
	if err != nil {
		return Patient{}, err
	}

	normRut, err := m.validateRUT(p.Rut)
	if err != nil {
		return Patient{}, ValidationError{Err: err}
	}
	p.Rut = normRut

	// If RUT changed, check duplicate
	if normRut != curr.Rut {
		existing, err := m.FindByRut(p.TenantId, normRut)
		if err == nil && existing.Id != "" && existing.Id != p.Id {
			return Patient{}, ErrRutAlreadyExists
		} else if err != nil && err != ErrNotFound {
			return Patient{}, err
		}
	}

	p.UpdatedAt = time.Now()

	if err := m.db.Update(&p, storage.Eq(Patient_.Id, p.Id), storage.Eq(Patient_.TenantId, p.TenantId)); err != nil {
		return Patient{}, err
	}

	m.publish(TopicPatientUpdated, &p)
	return p, nil
}

func (m *Module) GetPatient(tenantID, id string) (Patient, error) {
	if tenantID == "" || id == "" {
		return Patient{}, ErrNotFound
	}
	var p Patient
	qb := m.db.Query(&p).
		Where(Patient_.TenantId).Eq(tenantID).
		Where(Patient_.Id).Eq(id)
	_, err := ReadOnePatient(qb, &p)
	if err == orm.ErrNotFound {
		return Patient{}, ErrNotFound
	}
	if err != nil {
		return Patient{}, err
	}
	return p, nil
}

func (m *Module) FindByRut(tenantID, rut string) (Patient, error) {
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
		Where(Patient_.Rut).Eq(normRut)
	_, err = ReadOnePatient(qb, &p)
	if err == orm.ErrNotFound {
		return Patient{}, ErrNotFound
	}
	if err != nil {
		return Patient{}, err
	}
	return p, nil
}

func (m *Module) ListPatients(tenantID string, filter PatientFilter) ([]Patient, error) {
	if tenantID == "" {
		return []Patient{}, nil
	}
	var p Patient
	qb := m.db.Query(&p).Where(Patient_.TenantId).Eq(tenantID)
	if filter.ActiveOnly {
		qb = qb.Where(Patient_.IsActive).Eq(true)
	}
	if filter.Limit > 0 {
		qb = qb.Limit(int(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(int(filter.Offset))
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

func (m *Module) DeactivatePatient(tenantID, id string) error {
	p, err := m.GetPatient(tenantID, id)
	if err != nil {
		return err
	}
	p.IsActive = false
	p.UpdatedAt = time.Now()

	if err := m.db.Update(&p, storage.Eq(Patient_.Id, p.Id), storage.Eq(Patient_.TenantId, p.TenantId)); err != nil {
		return err
	}
	m.publish(TopicPatientDeactivated, &p)
	return nil
}

// ClientExists reports whether a patient with this id belongs to this tenant.
// It satisfies the narrow DirectoryReader port that a scheduling module
// declares on its own side (ClientExists(tenantId, clientId) (bool, error)) —
// structurally, with no adapter and no import in either direction.
//
// A missing row is (false, nil), NOT an error: "this id is not one of ours" is
// the answer the caller asked for. Only a real storage failure returns a
// non-nil error, so a caller can never mistake a dead database for a clean
// "no".
//
// An INACTIVE patient still exists. Deactivation removes someone from the
// working list; it does not unmake the person, and a reservation that
// references them must keep resolving.
func (m *Module) ClientExists(tenantID, clientID string) (bool, error) {
	if tenantID == "" || clientID == "" {
		return false, nil
	}
	_, err := m.GetPatient(tenantID, clientID)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
