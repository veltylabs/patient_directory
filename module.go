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
	IDs       model.IDGenerator // requerido — el módulo nunca genera los suyos propios
	Publisher events.Publisher  // opcional — nil deshabilita la publicación silenciosamente
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
	ids         model.IDGenerator
	pub         events.Publisher
	validateRUT func(string) (string, error)
	tenantID    string
}

// New conecta el módulo a un *orm.DB ya conectado; se asume que el esquema
// ya existe —consulte el submódulo migrate.
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("patient_directory: Deps.IDs is required")
	}
	if deps.ValidateRUT == nil {
		return nil, fmt.Err("patient_directory: Deps.ValidateRUT is required")
	}
	if deps.TenantID == "" {
		return nil, fmt.Err("patient_directory: Deps.TenantID is required")
	}
	return &Module{
		db:          db,
		ids:         deps.IDs,
		pub:         deps.Publisher,
		validateRUT: deps.ValidateRUT,
		tenantID:    deps.TenantID,
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

	// Verificar si el RUT ya existe para este tenant
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

	// Verificar que el paciente existente pertenezca al tenant
	curr, err := m.GetPatient(p.TenantId, p.Id)
	if err != nil {
		return Patient{}, err
	}

	normRut, err := m.validateRUT(p.Rut)
	if err != nil {
		return Patient{}, ValidationError{Err: err}
	}
	p.Rut = normRut

	// Si el RUT cambió, verificar duplicados
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

// ClientExists informa si un paciente con este ID pertenece a este tenant.
// Satisface el puerto estrecho DirectoryReader que un módulo de agendamiento
// declara de su lado (ClientExists(tenantId, clientId) (bool, error)) —
// estructuralmente, sin adaptador y sin importar en ninguna dirección.
//
// Un registro faltante es (false, nil), NO un error: "este ID no es nuestro" es
// la respuesta que el solicitante pidió. Solo una falla real de almacenamiento
// retorna un error no nulo, para que un llamador nunca confunda una base de
// datos caída con una respuesta negativa limpia.
//
// Un paciente INACTIVO aún existe. La desactivación remueve a alguien de la
// lista de trabajo; no desaparece a la persona, y una reserva que lo
// referencia debe continuar resolviendo.
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
