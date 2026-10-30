// Package database aplica las migraciones SQL embebidas y prepara el administrador inicial.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"strings"
)

//go:embed *.sql
var migrationFiles embed.FS

// Orden de aplicación; todos los scripts son idempotentes.
var migrationOrder = []string{
	"init.sql",
	"003_seguridad.sql",
	"004_backoffice.sql",
	"005_dental.sql",
	"006_dental_access.sql",
	"007_dental_notifications.sql",
}

const migrationLockID = 74210531

// Migrate aplica una sola vez cada script pendiente. Un bloqueo consultivo evita que dos instancias migren a la vez.
func Migrate(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("no se pudo obtener el bloqueo de migraciones: %w", err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID)
	if _, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dental_schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return err
	}
	for _, name := range migrationOrder {
		var applied bool
		if err = conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dental_schema_migrations WHERE name=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		script, err := migrationFiles.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err = conn.ExecContext(ctx, string(script)); err != nil {
			return fmt.Errorf("migración %s: %w", name, err)
		}
		if _, err = conn.ExecContext(ctx, `INSERT INTO dental_schema_migrations(name) VALUES($1)`, name); err != nil {
			return err
		}
		log.Printf("Migración aplicada: %s", name)
	}
	return nil
}

const legacyAdminEmail = "admin@upse.edu.ec"

// BootstrapAdmin crea o actualiza la cuenta definida en DENTAL_ADMIN_EMAIL/DENTAL_ADMIN_PASSWORD con perfil clínico admin.
// En producción desactiva la cuenta heredada si conserva la contraseña de ejemplo y exige otro administrador activo.
func BootstrapAdmin(ctx context.Context, db *sql.DB) error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("DENTAL_ADMIN_EMAIL")))
	password := os.Getenv("DENTAL_ADMIN_PASSWORD")
	production := os.Getenv("APP_ENV") == "production"
	if email != "" || password != "" {
		if email == "" || len(password) < 12 {
			return fmt.Errorf("DENTAL_ADMIN_EMAIL y DENTAL_ADMIN_PASSWORD (12+ caracteres) deben definirse juntos")
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var id int
		err = tx.QueryRowContext(ctx, `INSERT INTO usuarios(email,password,rol,nombre,activo) VALUES($1,hash_password($2),'admin','Administrador clínico',true)
			ON CONFLICT (email) DO UPDATE SET password=CASE WHEN verify_password($2,usuarios.password) THEN usuarios.password ELSE hash_password($2) END, rol='admin', activo=true
			RETURNING id`, email, password).Scan(&id)
		if err != nil {
			return fmt.Errorf("no se pudo preparar el administrador inicial: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dental_staff_roles(user_id,role,active) VALUES($1,'admin',true) ON CONFLICT (user_id) DO UPDATE SET role='admin',active=true,updated_at=NOW()`, id); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		log.Printf("Administrador clínico inicial listo: %s", email)
	}
	if !production || email == legacyAdminEmail {
		return nil
	}
	res, err := db.ExecContext(ctx, `UPDATE usuarios SET activo=false WHERE email=$1 AND activo AND verify_password('123456',password)`, legacyAdminEmail)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("Cuenta heredada %s desactivada: conserva la contraseña de ejemplo", legacyAdminEmail)
	}
	var admins int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dental_staff_roles r JOIN usuarios u ON u.id=r.user_id WHERE r.role='admin' AND r.active AND u.activo`).Scan(&admins); err != nil {
		return err
	}
	if admins == 0 {
		return fmt.Errorf("no hay administrador clínico activo: define DENTAL_ADMIN_EMAIL y DENTAL_ADMIN_PASSWORD")
	}
	return nil
}
