-- Núcleo SIGO-Dental. Los datos demo permanecen en el frontend; esta migración no copia datos de Multicatálogo.
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS dental_patients (
    id SERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    document_number VARCHAR(32) NOT NULL UNIQUE,
    phone VARCHAR(40) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    birth_date DATE,
    emergency_contact VARCHAR(255) NOT NULL DEFAULT '',
    guardian_name VARCHAR(255) NOT NULL DEFAULT '',
    insurance VARCHAR(255) NOT NULL DEFAULT '',
    medical_alert TEXT NOT NULL DEFAULT '',
    medical_history TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE dental_patients ADD COLUMN IF NOT EXISTS medical_history TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS dental_chairs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT TRUE
);
INSERT INTO dental_chairs (name) VALUES ('Sillón 1'), ('Sillón 2'), ('Sillón 3') ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS dental_appointments (
    id BIGSERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    chair_id INTEGER NOT NULL REFERENCES dental_chairs(id),
    professional_user_id INTEGER REFERENCES usuarios(id),
    professional_name VARCHAR(255) NOT NULL DEFAULT '',
    procedure_name VARCHAR(255) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 30 CHECK (duration_minutes BETWEEN 5 AND 480),
    status VARCHAR(32) NOT NULL DEFAULT 'Confirmada'
        CHECK (status IN ('Programada','Confirmada','En espera','En atención','Atendida','Cancelada','No asistió')),
    created_by INTEGER REFERENCES usuarios(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    EXCLUDE USING gist (
        chair_id WITH =,
        tstzrange(starts_at, starts_at + duration_minutes * INTERVAL '1 minute', '[)') WITH &&
    ) WHERE (status NOT IN ('Cancelada','No asistió'))
);
CREATE INDEX IF NOT EXISTS idx_dental_appointments_start ON dental_appointments(starts_at);
CREATE INDEX IF NOT EXISTS idx_dental_appointments_patient ON dental_appointments(patient_id, starts_at DESC);

CREATE TABLE IF NOT EXISTS dental_odontogram_findings (
    id BIGSERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    tooth_number VARCHAR(3) NOT NULL,
    surface VARCHAR(24) NOT NULL DEFAULT 'general',
    finding_type VARCHAR(40) NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_by INTEGER REFERENCES usuarios(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (patient_id, tooth_number, surface)
);

CREATE TABLE IF NOT EXISTS dental_clinical_notes (
    id BIGSERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    professional_user_id INTEGER NOT NULL REFERENCES usuarios(id),
    subjective TEXT NOT NULL DEFAULT '',
    objective TEXT NOT NULL DEFAULT '',
    assessment TEXT NOT NULL DEFAULT '',
    plan TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    signature_reference TEXT,
    amended_note_id BIGINT REFERENCES dental_clinical_notes(id)
);
CREATE INDEX IF NOT EXISTS idx_dental_notes_patient ON dental_clinical_notes(patient_id, created_at DESC);

CREATE OR REPLACE FUNCTION dental_notes_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'dental_clinical_notes is append-only; add a signed amendment instead';
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_dental_notes_immutable ON dental_clinical_notes;
CREATE TRIGGER trg_dental_notes_immutable
BEFORE UPDATE OR DELETE ON dental_clinical_notes
FOR EACH ROW EXECUTE FUNCTION dental_notes_immutable();

CREATE TABLE IF NOT EXISTS dental_periodontal_measurements (
    id BIGSERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    tooth_number VARCHAR(3) NOT NULL,
    site VARCHAR(3) NOT NULL CHECK (site IN ('MB','B','DB','ML','L','DL')),
    probing_depth SMALLINT NOT NULL CHECK (probing_depth BETWEEN 0 AND 15),
    gingival_margin SMALLINT NOT NULL DEFAULT 0 CHECK (gingival_margin BETWEEN -10 AND 15),
    attachment_level SMALLINT GENERATED ALWAYS AS (probing_depth + gingival_margin) STORED,
    furcation SMALLINT NOT NULL DEFAULT 0 CHECK (furcation BETWEEN 0 AND 3),
    mobility SMALLINT NOT NULL DEFAULT 0 CHECK (mobility BETWEEN 0 AND 3),
    bleeding BOOLEAN NOT NULL DEFAULT FALSE,
    recorded_by INTEGER REFERENCES usuarios(id),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dental_perio_latest ON dental_periodontal_measurements(patient_id,tooth_number,site,recorded_at DESC);

CREATE TABLE IF NOT EXISTS dental_inventory_items (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(32) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL DEFAULT 'Materiales',
    unit VARCHAR(40) NOT NULL DEFAULT 'unidades',
    quantity NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    minimum_quantity NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (minimum_quantity >= 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dental_inventory_movements (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES dental_inventory_items(id),
    quantity_delta NUMERIC(12,2) NOT NULL CHECK (quantity_delta <> 0),
    reason VARCHAR(255) NOT NULL,
    performed_by INTEGER REFERENCES usuarios(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dental_treatment_plans (
    id BIGSERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES dental_patients(id),
    title VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'Pendiente' CHECK (status IN ('Pendiente','Aprobado','En progreso','Completado','Cancelado')),
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    created_by INTEGER REFERENCES usuarios(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS dental_treatment_items (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES dental_treatment_plans(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    tooth_number VARCHAR(3) NOT NULL DEFAULT '',
    phase VARCHAR(100) NOT NULL DEFAULT 'Fase 1',
    amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'Pendiente' CHECK (status IN ('Pendiente','Aprobado','Agendado','Realizado'))
);

CREATE TABLE IF NOT EXISTS dental_audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES usuarios(id),
    action VARCHAR(24) NOT NULL,
    entity VARCHAR(80) NOT NULL,
    entity_id VARCHAR(64) NOT NULL,
    source_ip INET,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dental_audit_entity ON dental_audit_log(entity, entity_id, occurred_at DESC);

CREATE OR REPLACE FUNCTION dental_audit_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'dental_audit_log is append-only';
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_dental_audit_immutable ON dental_audit_log;
CREATE TRIGGER trg_dental_audit_immutable
BEFORE UPDATE OR DELETE ON dental_audit_log
FOR EACH ROW EXECUTE FUNCTION dental_audit_immutable();
