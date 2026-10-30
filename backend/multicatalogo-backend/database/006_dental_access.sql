-- Perfiles clínicos separados de los roles heredados de Multicatálogo.
CREATE TABLE IF NOT EXISTS dental_staff_roles (
    user_id INTEGER PRIMARY KEY REFERENCES usuarios(id) ON DELETE CASCADE,
    role VARCHAR(24) NOT NULL CHECK (role IN ('admin','receptionist','dentist','assistant','auditor')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    assigned_by INTEGER REFERENCES usuarios(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO dental_staff_roles (user_id, role)
SELECT id, 'admin' FROM usuarios WHERE rol = 'admin'
ON CONFLICT (user_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_dental_staff_role ON dental_staff_roles(role, active);
