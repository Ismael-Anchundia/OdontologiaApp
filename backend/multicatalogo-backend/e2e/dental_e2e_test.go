// Pruebas de extremo a extremo de la API dental contra PostgreSQL real.
// Se omiten si DENTAL_E2E_DATABASE_URL no está definido, para no afectar `go test ./...`.
package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/jackc/pgx/v5/stdlib"

	"multicatalogo-backend/auth"
	"multicatalogo-backend/cache"
	"multicatalogo-backend/config"
	"multicatalogo-backend/controllers"
	"multicatalogo-backend/repository"
	"multicatalogo-backend/routes"
)

type env struct {
	app    *fiber.App
	db     *sql.DB
	nonce  string
	tokens map[string]string
}

func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("DENTAL_E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("DENTAL_E2E_DATABASE_URL no definido; se omiten las pruebas E2E con PostgreSQL")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "e2e-only-secret-0123456789-abcdefghijkl")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("PostgreSQL no disponible: %v", err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous; db.Close() })

	repo := repository.NewPostgresRepository()
	redisCache := cache.NewRedisCache()
	app := fiber.New()
	routes.SetupRoutes(app,
		controllers.NewAuthController(repo), controllers.NewProdController(repo), controllers.NewRedController(repo),
		controllers.NewCheckoutController(repo, redisCache), controllers.NewRegisterController(repo),
		controllers.NewAdminProductosController(repo), controllers.NewPedidosController(repo),
		controllers.NewComisionesController(repo), controllers.NewLiquidacionesController(repo),
		controllers.NewReportesController(repo), controllers.NewUsuariosController(repo), controllers.NewDentalController())

	e := &env{app: app, db: db, nonce: fmt.Sprintf("%d", time.Now().UnixNano()), tokens: map[string]string{}}
	for _, role := range []string{"admin", "receptionist", "dentist", "assistant", "auditor", "none"} {
		email := fmt.Sprintf("e2e-%s-%s@dentalia.test", role, e.nonce)
		var id int
		if err := db.QueryRow(`INSERT INTO usuarios(email,password,rol,nombre,activo) VALUES($1,'not-a-login','cliente',$2,true) RETURNING id`, email, "E2E "+role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if role != "none" {
			if _, err := db.Exec(`INSERT INTO dental_staff_roles(user_id,role,active) VALUES($1,$2,true)`, id, role); err != nil {
				t.Fatal(err)
			}
		}
		token, err := auth.GenerateToken(id, email, "cliente", nil)
		if err != nil {
			t.Fatal(err)
		}
		e.tokens[role] = token
	}
	return e
}

func (e *env) call(t *testing.T, role, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token := e.tokens[role]; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req, 10000)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, out
}

func (e *env) expect(t *testing.T, want int, role, method, path string, body any) []byte {
	t.Helper()
	status, out := e.call(t, role, method, path, body)
	if status != want {
		t.Fatalf("%s %s como %s: status %d, se esperaba %d. Respuesta: %s", method, path, role, status, want, out)
	}
	return out
}

func (e *env) patient(t *testing.T, suffix string) string {
	t.Helper()
	out := e.expect(t, 201, "receptionist", "POST", "/api/dental/patients", map[string]any{
		"name": "Paciente E2E " + suffix, "document": "E2E" + suffix + e.nonce[len(e.nonce)-9:], "phone": "+593990000000",
		"birth": "1990-05-10", "medicalHistory": "Sin antecedentes relevantes",
	})
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &p); err != nil || !strings.HasPrefix(p.ID, "P-") {
		t.Fatalf("código de paciente inválido: %s", out)
	}
	return p.ID
}

func TestPatients(t *testing.T) {
	e := setup(t)
	doc := "PAT" + e.nonce[len(e.nonce)-12:]
	payload := map[string]any{"name": "Ana E2E", "document": doc, "phone": "0990000001", "medicalHistory": "Alergia a penicilina", "alert": "Penicilina"}
	out := e.expect(t, 201, "receptionist", "POST", "/api/dental/patients", payload)
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(out, &created)
	e.expect(t, 409, "receptionist", "POST", "/api/dental/patients", payload)
	e.expect(t, 400, "receptionist", "POST", "/api/dental/patients", map[string]any{"name": "Sin anamnesis", "document": doc + "X"})
	e.expect(t, 400, "receptionist", "POST", "/api/dental/patients", map[string]any{"name": "Fecha mala", "document": doc + "Y", "medicalHistory": "x", "birth": "10/05/1990"})
	payload["name"] = "Ana María E2E"
	e.expect(t, 200, "dentist", "PUT", "/api/dental/patients/"+created.ID, payload)
	e.expect(t, 404, "dentist", "PUT", "/api/dental/patients/P-999999", payload)
	out = e.expect(t, 200, "admin", "GET", "/api/dental/patients?q="+doc, nil)
	var list []struct {
		ID, Name, Alert string
	}
	json.Unmarshal(out, &list)
	if len(list) != 1 || list[0].Name != "Ana María E2E" || list[0].Alert != "Penicilina" {
		t.Fatalf("búsqueda de pacientes inesperada: %s", out)
	}
	var reads int
	e.db.QueryRow(`SELECT COUNT(*) FROM dental_audit_log a JOIN usuarios u ON u.id=a.user_id WHERE u.email LIKE $1 AND a.action='READ' AND a.entity='patients'`, "e2e-admin-"+e.nonce+"%").Scan(&reads)
	if reads != 1 {
		t.Fatalf("la consulta de pacientes debe quedar auditada, lecturas=%d", reads)
	}
}

func TestAppointmentsAndOverlap(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "CITA")
	// Fecha futura distinta por ejecución para no chocar con datos de corridas anteriores.
	day := time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(time.Now().UnixNano()%3000)).Format("2006-01-02")
	base := map[string]any{"patientId": patient, "date": day, "time": "10:00", "chair": "Sillón 3", "procedure": "Profilaxis", "doctor": "Dra. E2E", "durationMinutes": 30}
	with := func(changes map[string]any) map[string]any {
		m := map[string]any{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range changes {
			m[k] = v
		}
		return m
	}
	out := e.expect(t, 201, "receptionist", "POST", "/api/dental/appointments", base)
	var first struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(out, &first)
	e.expect(t, 409, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"time": "10:15"}))
	e.expect(t, 409, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"time": "09:45", "durationMinutes": 20}))
	e.expect(t, 201, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"time": "10:30"}))
	e.expect(t, 201, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"chair": "Sillón 2"}))
	e.expect(t, 400, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"chair": "Sillón 99"}))
	e.expect(t, 404, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"patientId": "P-999999"}))
	e.expect(t, 400, "receptionist", "POST", "/api/dental/appointments", with(map[string]any{"durationMinutes": 600}))

	var queued int
	e.db.QueryRow(`SELECT COUNT(*) FROM dental_notification_outbox WHERE appointment_id=$1`, first.ID).Scan(&queued)
	if queued != 3 {
		t.Fatalf("se esperaban confirmación + recordatorios 48h/24h en cola, hay %d", queued)
	}
	e.expect(t, 400, "assistant", "PATCH", fmt.Sprintf("/api/dental/appointments/%d/status", first.ID), map[string]any{"status": "Inventado"})
	e.expect(t, 200, "assistant", "PATCH", fmt.Sprintf("/api/dental/appointments/%d/status", first.ID), map[string]any{"status": "Cancelada"})
	var cancelled int
	e.db.QueryRow(`SELECT COUNT(*) FROM dental_notification_outbox WHERE appointment_id=$1 AND status='cancelled'`, first.ID).Scan(&cancelled)
	if cancelled != 3 {
		t.Fatalf("cancelar la cita debe cancelar sus notificaciones, canceladas=%d", cancelled)
	}
	e.expect(t, 201, "receptionist", "POST", "/api/dental/appointments", base)
	e.expect(t, 409, "assistant", "PATCH", fmt.Sprintf("/api/dental/appointments/%d/status", first.ID), map[string]any{"status": "Confirmada"})

	out = e.expect(t, 200, "assistant", "GET", "/api/dental/appointments?date="+day, nil)
	var list []struct{ PatientID, Status string }
	json.Unmarshal(out, &list)
	if len(list) != 4 {
		t.Fatalf("se esperaban 4 citas en el día, hay %d: %s", len(list), out)
	}
	e.expect(t, 400, "assistant", "GET", "/api/dental/appointments?from="+day, nil)
}

func TestImmutableClinicalNotes(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "NOTA")
	path := "/api/dental/patients/" + patient + "/notes"
	out := e.expect(t, 201, "dentist", "POST", path, map[string]any{"subjective": "Dolor al frío", "objective": "Caries 16", "assessment": "Caries dentina", "plan": "Restauración"})
	var note struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(out, &note)
	e.expect(t, 400, "dentist", "POST", path, map[string]any{"subjective": "  "})
	e.expect(t, 404, "dentist", "POST", "/api/dental/patients/P-999999/notes", map[string]any{"plan": "x"})
	out = e.expect(t, 200, "assistant", "GET", path, nil)
	if !strings.Contains(string(out), "Dolor al frío") {
		t.Fatalf("la nota no aparece en el historial: %s", out)
	}
	if status, _ := e.call(t, "dentist", "PUT", fmt.Sprintf("%s/%d", path, note.ID), map[string]any{"plan": "cambio"}); status < 404 || status > 405 {
		t.Fatalf("no debe existir ruta para editar notas, status=%d", status)
	}
	if _, err := e.db.Exec(`UPDATE dental_clinical_notes SET plan='alterado' WHERE id=$1`, note.ID); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE directo debe fallar por inmutabilidad, err=%v", err)
	}
	if _, err := e.db.Exec(`DELETE FROM dental_clinical_notes WHERE id=$1`, note.ID); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE directo debe fallar por inmutabilidad, err=%v", err)
	}
	if _, err := e.db.Exec(`UPDATE dental_audit_log SET action='X' WHERE entity='clinical_note' AND entity_id=$1`, fmt.Sprint(note.ID)); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("la bitácora de auditoría debe ser inmutable, err=%v", err)
	}
}

func TestOdontogram(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "ODON")
	path := "/api/dental/patients/" + patient + "/odontogram"
	e.expect(t, 400, "dentist", "PUT", path, map[string]any{"tooth": "19", "surface": "oclusal", "type": "Caries"})
	e.expect(t, 400, "dentist", "PUT", path, map[string]any{"tooth": "16", "surface": "techo", "type": "Caries"})
	e.expect(t, 400, "dentist", "PUT", path, map[string]any{"tooth": "16", "surface": "oclusal", "type": "Desconocido"})
	e.expect(t, 200, "dentist", "PUT", path, map[string]any{"tooth": "16", "surface": "oclusal", "type": "Caries", "notes": "Profunda"})
	e.expect(t, 200, "dentist", "PUT", path, map[string]any{"tooth": "16", "surface": "oclusal", "type": "Restauración"})
	e.expect(t, 200, "assistant", "PUT", path, map[string]any{"tooth": "55", "type": "Sellante"})
	out := e.expect(t, 200, "assistant", "GET", path, nil)
	var findings []struct{ Tooth, Surface, Type string }
	json.Unmarshal(out, &findings)
	if len(findings) != 2 || findings[0].Tooth != "16" || findings[0].Type != "Restauración" || findings[1].Surface != "general" {
		t.Fatalf("odontograma inesperado: %s", out)
	}
	e.expect(t, 200, "dentist", "PUT", path, map[string]any{"tooth": "16", "surface": "oclusal", "type": "Sano"})
	out = e.expect(t, 200, "dentist", "GET", path, nil)
	json.Unmarshal(out, &findings)
	if len(findings) != 1 {
		t.Fatalf("marcar Sano debe eliminar el hallazgo actual: %s", out)
	}
}

func TestPeriodontogram(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "PERIO")
	path := "/api/dental/patients/" + patient + "/periodontogram"
	out := e.expect(t, 201, "dentist", "POST", path, map[string]any{"tooth": "16", "site": "mb", "probingDepth": 5, "gingivalMargin": 2, "bleeding": true})
	var single struct {
		AttachmentLevel int `json:"attachmentLevel"`
	}
	json.Unmarshal(out, &single)
	if single.AttachmentLevel != 7 {
		t.Fatalf("NIC debe ser profundidad + margen (7), fue %d", single.AttachmentLevel)
	}
	e.expect(t, 400, "dentist", "POST", path, map[string]any{"tooth": "16", "site": "XX", "probingDepth": 3})
	e.expect(t, 400, "dentist", "POST", path, map[string]any{"tooth": "16", "site": "B", "probingDepth": 20})
	sites := []string{"MB", "B", "DB", "ML", "L", "DL"}
	chart := []map[string]any{}
	for i, s := range sites {
		chart = append(chart, map[string]any{"tooth": "16", "site": s, "probingDepth": 3, "gingivalMargin": i - 1, "mobility": 1})
	}
	e.expect(t, 201, "assistant", "POST", path+"/bulk", chart)
	e.expect(t, 400, "assistant", "POST", path+"/bulk", chart[:5])
	dup := append([]map[string]any{}, chart[:5]...)
	dup = append(dup, chart[0])
	e.expect(t, 400, "assistant", "POST", path+"/bulk", dup)
	out = e.expect(t, 200, "dentist", "GET", path, nil)
	var latest []struct {
		Site            string
		AttachmentLevel int `json:"attachmentLevel"`
	}
	json.Unmarshal(out, &latest)
	if len(latest) != 6 {
		t.Fatalf("se esperaban 6 sitios vigentes: %s", out)
	}
	for _, m := range latest {
		if m.Site == "MB" && m.AttachmentLevel != 2 {
			t.Fatalf("MB debe reflejar la última medición (NIC 2), fue %d", m.AttachmentLevel)
		}
	}
	var history int
	e.db.QueryRow(`SELECT COUNT(*) FROM dental_periodontal_measurements m JOIN dental_patients p ON p.id=m.patient_id WHERE p.code=$1`, patient).Scan(&history)
	if history != 7 {
		t.Fatalf("las mediciones deben conservarse como historial, hay %d", history)
	}
}

func TestTreatmentPlans(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "PLAN")
	e.expect(t, 400, "dentist", "POST", "/api/dental/treatment-plans", map[string]any{"patientId": patient, "title": "Vacío", "items": []any{}})
	e.expect(t, 400, "dentist", "POST", "/api/dental/treatment-plans", map[string]any{"patientId": patient, "title": "Negativo", "items": []any{map[string]any{"description": "X", "amount": -1}}})
	out := e.expect(t, 201, "dentist", "POST", "/api/dental/treatment-plans", map[string]any{"patientId": patient, "title": "Rehabilitación E2E", "items": []any{
		map[string]any{"description": "Endodoncia", "tooth": "16", "amount": 180.5},
		map[string]any{"description": "Corona", "tooth": "16", "phase": "Fase 2", "amount": 320},
	}})
	var plan struct {
		ID          int64   `json:"id"`
		TotalAmount float64 `json:"totalAmount"`
	}
	json.Unmarshal(out, &plan)
	if plan.TotalAmount != 500.5 {
		t.Fatalf("total del presupuesto = %v, se esperaba 500.5", plan.TotalAmount)
	}
	statusPath := fmt.Sprintf("/api/dental/treatment-plans/%d/status", plan.ID)
	e.expect(t, 400, "receptionist", "PATCH", statusPath, map[string]any{"status": "Regalado"})
	e.expect(t, 200, "receptionist", "PATCH", statusPath, map[string]any{"status": "Aprobado"})
	e.expect(t, 404, "receptionist", "PATCH", "/api/dental/treatment-plans/999999999/status", map[string]any{"status": "Aprobado"})
	out = e.expect(t, 200, "receptionist", "GET", "/api/dental/treatment-plans", nil)
	var plans []struct {
		ID     int64
		Status string
		Items  []struct{ Phase string }
	}
	json.Unmarshal(out, &plans)
	for _, p := range plans {
		if p.ID == plan.ID {
			if p.Status != "Aprobado" || len(p.Items) != 2 || p.Items[0].Phase != "Fase 1" {
				t.Fatalf("plan inesperado: %+v", p)
			}
			return
		}
	}
	t.Fatalf("el plan %d no aparece en el listado", plan.ID)
}

func TestInventory(t *testing.T) {
	e := setup(t)
	out := e.expect(t, 201, "assistant", "POST", "/api/dental/inventory", map[string]any{"name": "Resina E2E " + e.nonce, "quantity": 10, "minimum": 4})
	var item struct {
		ID  int64  `json:"id"`
		SKU string `json:"sku"`
	}
	json.Unmarshal(out, &item)
	if !strings.HasPrefix(item.SKU, "MAT-") {
		t.Fatalf("SKU inesperado: %s", out)
	}
	e.expect(t, 400, "assistant", "POST", "/api/dental/inventory", map[string]any{"name": " "})
	path := fmt.Sprintf("/api/dental/inventory/%d", item.ID)
	out = e.expect(t, 200, "assistant", "PATCH", path, map[string]any{"delta": -3, "reason": "Uso en procedimiento"})
	if !strings.Contains(string(out), `"quantity":7`) {
		t.Fatalf("existencia esperada 7: %s", out)
	}
	e.expect(t, 400, "assistant", "PATCH", path, map[string]any{"delta": -100})
	e.expect(t, 400, "assistant", "PATCH", path, map[string]any{"delta": 0})
	e.expect(t, 404, "assistant", "PATCH", "/api/dental/inventory/999999999", map[string]any{"delta": 1})
	var movements int
	var sum float64
	e.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(quantity_delta),0) FROM dental_inventory_movements WHERE item_id=$1`, item.ID).Scan(&movements, &sum)
	if movements != 2 || sum != 7 {
		t.Fatalf("kardex inesperado: movimientos=%d suma=%v", movements, sum)
	}
	out = e.expect(t, 200, "auditor", "GET", "/api/dental/inventory", nil)
	if !strings.Contains(string(out), item.SKU) {
		t.Fatalf("el insumo no aparece en el inventario")
	}
}

func TestPermissions(t *testing.T) {
	e := setup(t)
	patient := e.patient(t, "PERM")
	notes := "/api/dental/patients/" + patient + "/notes"
	cases := []struct {
		role, method, path string
		body               any
		want               int
	}{
		{"", "GET", "/api/dental/patients", nil, 401},
		{"none", "GET", "/api/dental/me", nil, 403},
		{"auditor", "GET", "/api/dental/me", nil, 200},
		{"receptionist", "GET", notes, nil, 403},
		{"receptionist", "POST", notes, map[string]any{"plan": "x"}, 403},
		{"auditor", "GET", notes, nil, 403},
		{"assistant", "POST", "/api/dental/patients", map[string]any{"name": "x"}, 403},
		{"assistant", "GET", "/api/dental/patients", nil, 200},
		{"receptionist", "GET", "/api/dental/inventory", nil, 403},
		{"auditor", "POST", "/api/dental/inventory", map[string]any{"name": "x"}, 403},
		{"dentist", "PATCH", "/api/dental/inventory/1", map[string]any{"delta": 1}, 403},
		{"receptionist", "POST", "/api/dental/treatment-plans", map[string]any{}, 403},
		{"assistant", "GET", "/api/dental/treatment-plans", nil, 403},
		{"dentist", "GET", "/api/dental/staff", nil, 403},
		{"dentist", "PUT", "/api/dental/staff/1/role", map[string]any{"role": "admin"}, 403},
		{"auditor", "POST", "/api/dental/appointments", map[string]any{}, 403},
		{"admin", "GET", "/api/dental/staff", nil, 200},
	}
	for _, tc := range cases {
		if status, out := e.call(t, tc.role, tc.method, tc.path, tc.body); status != tc.want {
			t.Errorf("%s %s como %q: status %d, se esperaba %d (%s)", tc.method, tc.path, tc.role, status, tc.want, out)
		}
	}
	var noneID int
	e.db.QueryRow(`SELECT id FROM usuarios WHERE email=$1`, "e2e-none-"+e.nonce+"@dentalia.test").Scan(&noneID)
	e.expect(t, 400, "admin", "PUT", fmt.Sprintf("/api/dental/staff/%d/role", noneID), map[string]any{"role": "superuser"})
	e.expect(t, 200, "admin", "PUT", fmt.Sprintf("/api/dental/staff/%d/role", noneID), map[string]any{"role": "auditor"})
	e.expect(t, 200, "none", "GET", "/api/dental/me", nil)
	e.expect(t, 200, "admin", "PUT", fmt.Sprintf("/api/dental/staff/%d/role", noneID), map[string]any{"role": "auditor", "active": false})
	e.expect(t, 403, "none", "GET", "/api/dental/me", nil)
}
