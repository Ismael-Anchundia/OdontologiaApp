package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"multicatalogo-backend/config"
	"multicatalogo-backend/models"

	"github.com/gofiber/fiber/v2"
)

type DentalController struct{}

func NewDentalController() *DentalController { return &DentalController{} }

func dentalError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(models.APIError{Status: status, Message: message})
}

func dentalUserID(c *fiber.Ctx) (int, bool) {
	id, ok := c.Locals("user_id").(int)
	return id, ok && id > 0
}

func dentalAudit(tx *sql.Tx, c *fiber.Ctx, userID int, action, entity, entityID string) error {
	_, err := tx.Exec(`INSERT INTO dental_audit_log (user_id, action, entity, entity_id, source_ip) VALUES ($1,$2,$3,$4,NULLIF($5,'')::inet)`, userID, action, entity, entityID, c.IP())
	return err
}

type dentalPatient struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Document  string `json:"document"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Birth     string `json:"birth"`
	Emergency string `json:"emergencyContact"`
	Guardian  string `json:"guardianName"`
	Insurance string `json:"insurance"`
	Alert     string `json:"alert"`
	History   string `json:"medicalHistory"`
	LastVisit string `json:"lastVisit"`
}

func (dc *DentalController) ListPatients(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	query := strings.TrimSpace(c.Query("q"))
	rows, err := config.DB.Query(`SELECT p.code,p.full_name,p.document_number,p.phone,p.email,COALESCE(to_char(p.birth_date,'YYYY-MM-DD'),''),p.emergency_contact,p.guardian_name,p.insurance,p.medical_alert,p.medical_history,COALESCE(to_char(MAX(a.starts_at AT TIME ZONE 'America/Guayaquil'),'DD Mon, HH24:MI'),'Sin visitas') FROM dental_patients p LEFT JOIN dental_appointments a ON a.patient_id=p.id WHERE p.active AND ($1='' OR p.full_name ILIKE '%'||$1||'%' OR p.document_number ILIKE '%'||$1||'%' OR p.code ILIKE '%'||$1||'%') GROUP BY p.id ORDER BY p.created_at DESC LIMIT 500`, query)
	if err != nil {
		return dentalError(c, 500, "No se pudieron consultar los pacientes")
	}
	defer rows.Close()
	result := make([]dentalPatient, 0)
	for rows.Next() {
		var p dentalPatient
		if err := rows.Scan(&p.ID, &p.Name, &p.Document, &p.Phone, &p.Email, &p.Birth, &p.Emergency, &p.Guardian, &p.Insurance, &p.Alert, &p.History, &p.LastVisit); err != nil {
			return dentalError(c, 500, "No se pudieron leer los pacientes")
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return dentalError(c, 500, "No se pudieron leer los pacientes")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log (user_id,action,entity,entity_id,source_ip) VALUES ($1,'READ','patients','list',NULLIF($2,'')::inet)`, userID, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso al expediente")
	}
	return c.JSON(result)
}

func (dc *DentalController) CreatePatient(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	var req struct {
		Name      string `json:"name"`
		Document  string `json:"document"`
		Phone     string `json:"phone"`
		Email     string `json:"email"`
		Birth     string `json:"birth"`
		Alert     string `json:"alert"`
		Emergency string `json:"emergencyContact"`
		Guardian  string `json:"guardianName"`
		Insurance string `json:"insurance"`
		History   string `json:"medicalHistory"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Datos de paciente inválidos")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Document = strings.TrimSpace(req.Document)
	if req.Name == "" || req.Document == "" || len(req.Name) > 255 || len(req.Document) > 32 {
		return dentalError(c, 400, "Nombre e identificación son obligatorios")
	}
	var birth any
	if req.Birth != "" {
		parsed, err := time.Parse("2006-01-02", req.Birth)
		if err != nil {
			return dentalError(c, 400, "Fecha de nacimiento inválida")
		}
		birth = parsed
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo iniciar el registro")
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRow(`INSERT INTO dental_patients (code,full_name,document_number,phone,email,birth_date,medical_alert,emergency_contact,guardian_name,insurance,medical_history) VALUES ('pending-'||nextval(pg_get_serial_sequence('dental_patients','id'))::text,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, req.Name, req.Document, req.Phone, req.Email, birth, req.Alert, req.Emergency, req.Guardian, req.Insurance, req.History).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return dentalError(c, 409, "Ya existe un paciente con esa identificación")
		}
		return dentalError(c, 500, "No se pudo registrar el paciente")
	}
	code := fmt.Sprintf("P-%04d", id)
	if _, err = tx.Exec(`UPDATE dental_patients SET code=$1 WHERE id=$2`, code, id); err != nil {
		return dentalError(c, 500, "No se pudo asignar el código del paciente")
	}
	if err = dentalAudit(tx, c, userID, "CREATE", "patient", strconv.FormatInt(id, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar el registro")
	}
	return c.Status(201).JSON(fiber.Map{"id": code, "name": req.Name, "document": req.Document, "phone": req.Phone, "email": req.Email, "birth": req.Birth, "emergencyContact": req.Emergency, "guardianName": req.Guardian, "insurance": req.Insurance, "alert": req.Alert, "medicalHistory": req.History, "lastVisit": "Sin visitas"})
}

func (dc *DentalController) UpdatePatient(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	code := strings.TrimSpace(c.Params("patientId"))
	var req struct {
		Name      string `json:"name"`
		Document  string `json:"document"`
		Phone     string `json:"phone"`
		Email     string `json:"email"`
		Birth     string `json:"birth"`
		Alert     string `json:"alert"`
		Emergency string `json:"emergencyContact"`
		Guardian  string `json:"guardianName"`
		Insurance string `json:"insurance"`
		History   string `json:"medicalHistory"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Datos de paciente inválidos")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Document = strings.TrimSpace(req.Document)
	if req.Name == "" || req.Document == "" || len(req.Name) > 255 || len(req.Document) > 32 {
		return dentalError(c, 400, "Nombre e identificación son obligatorios")
	}
	var birth any
	if req.Birth != "" {
		parsed, err := time.Parse("2006-01-02", req.Birth)
		if err != nil {
			return dentalError(c, 400, "Fecha de nacimiento inválida")
		}
		birth = parsed
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo iniciar la actualización")
	}
	defer tx.Rollback()
	var id int
	err = tx.QueryRow(`UPDATE dental_patients SET full_name=$1,document_number=$2,phone=$3,email=$4,birth_date=$5,medical_alert=$6,emergency_contact=$7,guardian_name=$8,insurance=$9,medical_history=$10,updated_at=NOW() WHERE code=$11 AND active RETURNING id`, req.Name, req.Document, req.Phone, req.Email, birth, req.Alert, req.Emergency, req.Guardian, req.Insurance, req.History, code).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return dentalError(c, 409, "Ya existe otro paciente con esa identificación")
		}
		return dentalError(c, 500, "No se pudo actualizar el paciente")
	}
	if err = dentalAudit(tx, c, userID, "UPDATE", "patient", strconv.Itoa(id)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la actualización")
	}
	return c.JSON(fiber.Map{"id": code, "name": req.Name, "document": req.Document, "phone": req.Phone, "email": req.Email, "birth": req.Birth, "emergencyContact": req.Emergency, "guardianName": req.Guardian, "insurance": req.Insurance, "alert": req.Alert, "medicalHistory": req.History})
}

type dentalAppointment struct {
	ID        int64  `json:"id"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	PatientID string `json:"patientId"`
	Procedure string `json:"procedure"`
	Doctor    string `json:"doctor"`
	Chair     string `json:"chair"`
	Status    string `json:"status"`
}

func (dc *DentalController) ListAppointments(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	date := strings.TrimSpace(c.Query("date"))
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return dentalError(c, 400, "Fecha inválida")
		}
	}
	rows, err := config.DB.Query(`SELECT a.id,to_char(a.starts_at AT TIME ZONE 'America/Guayaquil','YYYY-MM-DD'),to_char(a.starts_at AT TIME ZONE 'America/Guayaquil','HH24:MI'),p.code,a.procedure_name,a.professional_name,ch.name,a.status FROM dental_appointments a JOIN dental_patients p ON p.id=a.patient_id JOIN dental_chairs ch ON ch.id=a.chair_id WHERE ($1='' OR (a.starts_at AT TIME ZONE 'America/Guayaquil')::date=$1::date) ORDER BY a.starts_at LIMIT 1000`, date)
	if err != nil {
		return dentalError(c, 500, "No se pudieron consultar las citas")
	}
	defer rows.Close()
	result := make([]dentalAppointment, 0)
	for rows.Next() {
		var a dentalAppointment
		if err := rows.Scan(&a.ID, &a.Date, &a.Time, &a.PatientID, &a.Procedure, &a.Doctor, &a.Chair, &a.Status); err != nil {
			return dentalError(c, 500, "No se pudieron leer las citas")
		}
		result = append(result, a)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudieron leer las citas")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log (user_id,action,entity,entity_id,source_ip) VALUES ($1,'READ','appointments','list',NULLIF($2,'')::inet)`, userID, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso a la agenda")
	}
	return c.JSON(result)
}

func (dc *DentalController) CreateAppointment(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	var req struct {
		PatientID string `json:"patientId"`
		Date      string `json:"date"`
		Time      string `json:"time"`
		Chair     string `json:"chair"`
		Procedure string `json:"procedure"`
		Doctor    string `json:"doctor"`
		Duration  int    `json:"durationMinutes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Datos de cita inválidos")
	}
	if req.PatientID == "" || req.Procedure == "" || req.Chair == "" {
		return dentalError(c, 400, "Paciente, tratamiento y sillón son obligatorios")
	}
	if req.Duration == 0 {
		req.Duration = 30
	}
	if req.Duration < 5 || req.Duration > 480 {
		return dentalError(c, 400, "Duración inválida")
	}
	location, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		return dentalError(c, 500, "No se pudo cargar la zona horaria")
	}
	starts, err := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Time, location)
	if err != nil {
		return dentalError(c, 400, "Fecha u hora inválida")
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo iniciar la reserva")
	}
	defer tx.Rollback()
	var patientPK int
	err = tx.QueryRow(`SELECT id FROM dental_patients WHERE code=$1 AND active`, req.PatientID).Scan(&patientPK)
	if errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el paciente")
	}
	var chairID int
	err = tx.QueryRow(`SELECT id FROM dental_chairs WHERE name=$1 AND active`, req.Chair).Scan(&chairID)
	if errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 400, "Sillón no válido")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el sillón")
	}
	var appointmentID int64
	err = tx.QueryRow(`INSERT INTO dental_appointments(patient_id,chair_id,professional_user_id,professional_name,procedure_name,starts_at,duration_minutes,status,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,'Confirmada',$3) RETURNING id`, patientPK, chairID, userID, req.Doctor, req.Procedure, starts, req.Duration).Scan(&appointmentID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "exclusion constraint") || strings.Contains(strings.ToLower(err.Error()), "conflicting key") {
			return dentalError(c, 409, "El sillón ya tiene una cita en ese horario")
		}
		return dentalError(c, 500, "No se pudo agendar la cita")
	}
	if err = dentalAudit(tx, c, userID, "CREATE", "appointment", strconv.FormatInt(appointmentID, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la reserva")
	}
	return c.Status(201).JSON(dentalAppointment{ID: appointmentID, Date: req.Date, Time: req.Time, PatientID: req.PatientID, Procedure: req.Procedure, Doctor: req.Doctor, Chair: req.Chair, Status: "Confirmada"})
}

func (dc *DentalController) UpdateAppointmentStatus(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return dentalError(c, 400, "Cita inválida")
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Estado inválido")
	}
	allowed := map[string]bool{"Programada": true, "Confirmada": true, "En espera": true, "En atención": true, "Atendida": true, "Cancelada": true, "No asistió": true}
	if !allowed[req.Status] {
		return dentalError(c, 400, "Estado de cita no permitido")
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo actualizar la cita")
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE dental_appointments SET status=$1 WHERE id=$2`, req.Status, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "exclusion constraint") || strings.Contains(strings.ToLower(err.Error()), "conflicting key") {
			return dentalError(c, 409, "El sillón ya tiene una cita en ese horario")
		}
		return dentalError(c, 500, "No se pudo actualizar la cita")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return dentalError(c, 404, "Cita no encontrada")
	}
	if err = dentalAudit(tx, c, userID, "UPDATE", "appointment", strconv.FormatInt(id, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la actualización")
	}
	return c.JSON(fiber.Map{"id": id, "status": req.Status})
}

type dentalNote struct {
	ID           int64  `json:"id"`
	PatientID    string `json:"patientId"`
	Professional string `json:"professional"`
	Subjective   string `json:"subjective"`
	Objective    string `json:"objective"`
	Assessment   string `json:"assessment"`
	Plan         string `json:"plan"`
	CreatedAt    string `json:"createdAt"`
}

func (dc *DentalController) ListNotes(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	patientCode := c.Params("patientId")
	rows, err := config.DB.Query(`SELECT n.id,p.code,COALESCE(u.nombre,u.email),n.subjective,n.objective,n.assessment,n.plan,to_char(n.created_at AT TIME ZONE 'America/Guayaquil','YYYY-MM-DD HH24:MI') FROM dental_clinical_notes n JOIN dental_patients p ON p.id=n.patient_id JOIN usuarios u ON u.id=n.professional_user_id WHERE p.code=$1 ORDER BY n.created_at DESC LIMIT 200`, patientCode)
	if err != nil {
		return dentalError(c, 500, "No se pudieron consultar las notas")
	}
	defer rows.Close()
	notes := make([]dentalNote, 0)
	for rows.Next() {
		var n dentalNote
		if err := rows.Scan(&n.ID, &n.PatientID, &n.Professional, &n.Subjective, &n.Objective, &n.Assessment, &n.Plan, &n.CreatedAt); err != nil {
			return dentalError(c, 500, "No se pudieron leer las notas")
		}
		notes = append(notes, n)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudieron leer las notas")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log (user_id,action,entity,entity_id,source_ip) VALUES ($1,'READ','clinical_notes',$2,NULLIF($3,'')::inet)`, userID, patientCode, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso a las notas")
	}
	return c.JSON(notes)
}

func (dc *DentalController) CreateNote(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	patientCode := c.Params("patientId")
	var req struct {
		Subjective string `json:"subjective"`
		Objective  string `json:"objective"`
		Assessment string `json:"assessment"`
		Plan       string `json:"plan"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Nota inválida")
	}
	if strings.TrimSpace(req.Subjective+req.Objective+req.Assessment+req.Plan) == "" {
		return dentalError(c, 400, "La nota debe incluir al menos un campo")
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo iniciar la nota")
	}
	defer tx.Rollback()
	var patientID int
	err = tx.QueryRow(`SELECT id FROM dental_patients WHERE code=$1 AND active`, patientCode).Scan(&patientID)
	if errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el paciente")
	}
	var id int64
	var created time.Time
	err = tx.QueryRow(`INSERT INTO dental_clinical_notes(patient_id,professional_user_id,subjective,objective,assessment,plan) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,created_at`, patientID, userID, req.Subjective, req.Objective, req.Assessment, req.Plan).Scan(&id, &created)
	if err != nil {
		return dentalError(c, 500, "No se pudo guardar la nota")
	}
	if err = dentalAudit(tx, c, userID, "CREATE", "clinical_note", strconv.FormatInt(id, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la nota")
	}
	return c.Status(201).JSON(dentalNote{ID: id, PatientID: patientCode, Subjective: req.Subjective, Objective: req.Objective, Assessment: req.Assessment, Plan: req.Plan, CreatedAt: created.In(time.Local).Format(time.RFC3339)})
}

type dentalFinding struct {
	Tooth   string `json:"tooth"`
	Surface string `json:"surface"`
	Type    string `json:"type"`
	Notes   string `json:"notes"`
}

func (dc *DentalController) ListFindings(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	code := c.Params("patientId")
	rows, err := config.DB.Query(`SELECT f.tooth_number,f.surface,f.finding_type,f.notes FROM dental_odontogram_findings f JOIN dental_patients p ON p.id=f.patient_id WHERE p.code=$1 ORDER BY f.tooth_number,f.surface`, code)
	if err != nil {
		return dentalError(c, 500, "No se pudo consultar el odontograma")
	}
	defer rows.Close()
	items := make([]dentalFinding, 0)
	for rows.Next() {
		var f dentalFinding
		if err := rows.Scan(&f.Tooth, &f.Surface, &f.Type, &f.Notes); err != nil {
			return dentalError(c, 500, "No se pudo leer el odontograma")
		}
		items = append(items, f)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudo leer el odontograma")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log(user_id,action,entity,entity_id,source_ip) VALUES($1,'READ','odontogram',$2,NULLIF($3,'')::inet)`, userID, code, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso al odontograma")
	}
	return c.JSON(items)
}

func (dc *DentalController) UpsertFinding(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	code := c.Params("patientId")
	var req dentalFinding
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Hallazgo inválido")
	}
	req.Tooth = strings.TrimSpace(req.Tooth)
	if req.Surface == "" {
		req.Surface = "general"
	}
	validTooth := false
	for _, n := range []string{"11", "12", "13", "14", "15", "16", "17", "18", "21", "22", "23", "24", "25", "26", "27", "28", "31", "32", "33", "34", "35", "36", "37", "38", "41", "42", "43", "44", "45", "46", "47", "48", "51", "52", "53", "54", "55", "61", "62", "63", "64", "65", "71", "72", "73", "74", "75", "81", "82", "83", "84", "85"} {
		if req.Tooth == n {
			validTooth = true
			break
		}
	}
	if !validTooth {
		return dentalError(c, 400, "Pieza dental FDI inválida")
	}
	if req.Type != "Caries" && req.Type != "Restauración" && req.Type != "Endodoncia" && req.Type != "Implante" && req.Type != "Prótesis" && req.Type != "Exodoncia" && req.Type != "Sellante" && req.Type != "Sano" {
		return dentalError(c, 400, "Tipo de hallazgo no permitido")
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo actualizar el odontograma")
	}
	defer tx.Rollback()
	var patientID int
	if err = tx.QueryRow(`SELECT id FROM dental_patients WHERE code=$1 AND active`, code).Scan(&patientID); errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el paciente")
	}
	if req.Type == "Sano" {
		_, err = tx.Exec(`DELETE FROM dental_odontogram_findings WHERE patient_id=$1 AND tooth_number=$2 AND surface=$3`, patientID, req.Tooth, req.Surface)
	} else {
		_, err = tx.Exec(`INSERT INTO dental_odontogram_findings(patient_id,tooth_number,surface,finding_type,notes,created_by,updated_at) VALUES($1,$2,$3,$4,$5,$6,NOW()) ON CONFLICT(patient_id,tooth_number,surface) DO UPDATE SET finding_type=EXCLUDED.finding_type,notes=EXCLUDED.notes,created_by=EXCLUDED.created_by,updated_at=NOW()`, patientID, req.Tooth, req.Surface, req.Type, req.Notes, userID)
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo guardar el hallazgo")
	}
	if err = dentalAudit(tx, c, userID, "UPDATE", "odontogram", code+":"+req.Tooth+":"+req.Surface); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la actualización")
	}
	return c.JSON(req)
}

type dentalPerioMeasurement struct {
	Tooth           string `json:"tooth"`
	Site            string `json:"site"`
	ProbingDepth    int    `json:"probingDepth"`
	GingivalMargin  int    `json:"gingivalMargin"`
	AttachmentLevel int    `json:"attachmentLevel"`
	Furcation       int    `json:"furcation"`
	Mobility        int    `json:"mobility"`
	Bleeding        bool   `json:"bleeding"`
	RecordedAt      string `json:"recordedAt"`
}

func (dc *DentalController) ListPeriodontogram(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	code := c.Params("patientId")
	rows, err := config.DB.Query(`SELECT DISTINCT ON (m.tooth_number,m.site) m.tooth_number,m.site,m.probing_depth,m.gingival_margin,m.attachment_level,m.furcation,m.mobility,m.bleeding,to_char(m.recorded_at AT TIME ZONE 'America/Guayaquil','YYYY-MM-DD HH24:MI') FROM dental_periodontal_measurements m JOIN dental_patients p ON p.id=m.patient_id WHERE p.code=$1 ORDER BY m.tooth_number,m.site,m.recorded_at DESC`, code)
	if err != nil {
		return dentalError(c, 500, "No se pudo consultar el periodontograma")
	}
	defer rows.Close()
	measurements := make([]dentalPerioMeasurement, 0)
	for rows.Next() {
		var m dentalPerioMeasurement
		if err := rows.Scan(&m.Tooth, &m.Site, &m.ProbingDepth, &m.GingivalMargin, &m.AttachmentLevel, &m.Furcation, &m.Mobility, &m.Bleeding, &m.RecordedAt); err != nil {
			return dentalError(c, 500, "No se pudieron leer las mediciones")
		}
		measurements = append(measurements, m)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudieron leer las mediciones")
	}
	if _, err = config.DB.Exec(`INSERT INTO dental_audit_log(user_id,action,entity,entity_id,source_ip) VALUES($1,'READ','periodontogram',$2,NULLIF($3,'')::inet)`, userID, code, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso al periodontograma")
	}
	return c.JSON(measurements)
}

func (dc *DentalController) RecordPeriodontalMeasurement(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	code := c.Params("patientId")
	var req dentalPerioMeasurement
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Medición periodontal inválida")
	}
	req.Tooth = strings.TrimSpace(req.Tooth)
	req.Site = strings.ToUpper(strings.TrimSpace(req.Site))
	validTooth := false
	for _, n := range []string{"11", "12", "13", "14", "15", "16", "17", "18", "21", "22", "23", "24", "25", "26", "27", "28", "31", "32", "33", "34", "35", "36", "37", "38", "41", "42", "43", "44", "45", "46", "47", "48", "51", "52", "53", "54", "55", "61", "62", "63", "64", "65", "71", "72", "73", "74", "75", "81", "82", "83", "84", "85"} {
		if req.Tooth == n {
			validTooth = true
			break
		}
	}
	validSite := map[string]bool{"MB": true, "B": true, "DB": true, "ML": true, "L": true, "DL": true}[req.Site]
	if !validTooth || !validSite || req.ProbingDepth < 0 || req.ProbingDepth > 15 || req.GingivalMargin < -10 || req.GingivalMargin > 15 || req.Furcation < 0 || req.Furcation > 3 || req.Mobility < 0 || req.Mobility > 3 {
		return dentalError(c, 400, "Revisa pieza, sitio y valores clínicos")
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo guardar la medición")
	}
	defer tx.Rollback()
	var patientID int
	if err = tx.QueryRow(`SELECT id FROM dental_patients WHERE code=$1 AND active`, code).Scan(&patientID); errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el paciente")
	}
	var level int
	err = tx.QueryRow(`INSERT INTO dental_periodontal_measurements(patient_id,tooth_number,site,probing_depth,gingival_margin,furcation,mobility,bleeding,recorded_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING attachment_level`, patientID, req.Tooth, req.Site, req.ProbingDepth, req.GingivalMargin, req.Furcation, req.Mobility, req.Bleeding, userID).Scan(&level)
	if err != nil {
		return dentalError(c, 500, "No se pudo guardar la medición")
	}
	req.AttachmentLevel = level
	if err = dentalAudit(tx, c, userID, "CREATE", "periodontal_measurement", code+":"+req.Tooth+":"+req.Site); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar la medición")
	}
	return c.Status(201).JSON(req)
}

type dentalSupply struct {
	ID       int64   `json:"id"`
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Unit     string  `json:"unit"`
	Quantity float64 `json:"quantity"`
	Minimum  float64 `json:"minimum"`
}

func (dc *DentalController) ListInventory(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	rows, err := config.DB.Query(`SELECT id,sku,name,category,unit,quantity,minimum_quantity FROM dental_inventory_items WHERE active ORDER BY name`)
	if err != nil {
		return dentalError(c, 500, "No se pudo consultar el inventario")
	}
	defer rows.Close()
	items := make([]dentalSupply, 0)
	for rows.Next() {
		var x dentalSupply
		if err := rows.Scan(&x.ID, &x.SKU, &x.Name, &x.Category, &x.Unit, &x.Quantity, &x.Minimum); err != nil {
			return dentalError(c, 500, "No se pudo leer el inventario")
		}
		items = append(items, x)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudo leer el inventario")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log(user_id,action,entity,entity_id,source_ip) VALUES($1,'READ','inventory','list',NULLIF($2,'')::inet)`, userID, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso al inventario")
	}
	return c.JSON(items)
}

func (dc *DentalController) CreateInventoryItem(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	var req struct {
		Name     string  `json:"name"`
		Category string  `json:"category"`
		Unit     string  `json:"unit"`
		Quantity float64 `json:"quantity"`
		Minimum  float64 `json:"minimum"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Insumo inválido")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return dentalError(c, 400, "El nombre es obligatorio")
	}
	if req.Unit == "" {
		req.Unit = "unidades"
	}
	if req.Category == "" {
		req.Category = "Materiales"
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo crear el insumo")
	}
	defer tx.Rollback()
	var id int64
	var sku string
	err = tx.QueryRow(`INSERT INTO dental_inventory_items(sku,name,category,unit,quantity,minimum_quantity) VALUES('pending-'||nextval(pg_get_serial_sequence('dental_inventory_items','id'))::text,$1,$2,$3,0,$4) RETURNING id`, req.Name, req.Category, req.Unit, req.Minimum).Scan(&id)
	if err != nil {
		return dentalError(c, 500, "No se pudo crear el insumo")
	}
	sku = fmt.Sprintf("MAT-%04d", id)
	if _, err = tx.Exec(`UPDATE dental_inventory_items SET sku=$1 WHERE id=$2`, sku, id); err != nil {
		return dentalError(c, 500, "No se pudo asignar el código")
	}
	if req.Quantity > 0 {
		if _, err = tx.Exec(`UPDATE dental_inventory_items SET quantity=$1 WHERE id=$2`, req.Quantity, id); err != nil {
			return dentalError(c, 500, "No se pudo establecer existencias")
		}
		if _, err = tx.Exec(`INSERT INTO dental_inventory_movements(item_id,quantity_delta,reason,performed_by) VALUES($1,$2,'Existencia inicial',$3)`, id, req.Quantity, userID); err != nil {
			return dentalError(c, 500, "No se pudo registrar el movimiento")
		}
	}
	if err = dentalAudit(tx, c, userID, "CREATE", "inventory_item", strconv.FormatInt(id, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar el insumo")
	}
	return c.Status(201).JSON(dentalSupply{ID: id, SKU: sku, Name: req.Name, Category: req.Category, Unit: req.Unit, Quantity: req.Quantity, Minimum: req.Minimum})
}

func (dc *DentalController) AdjustInventory(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return dentalError(c, 400, "Insumo inválido")
	}
	var req struct {
		Delta  float64 `json:"delta"`
		Reason string  `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Delta == 0 {
		return dentalError(c, 400, "Indica un ajuste distinto de cero")
	}
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Ajuste manual"
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo actualizar el inventario")
	}
	defer tx.Rollback()
	var quantity float64
	err = tx.QueryRow(`UPDATE dental_inventory_items SET quantity=quantity+$1 WHERE id=$2 AND active AND quantity+$1>=0 RETURNING quantity`, req.Delta, id).Scan(&quantity)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		_ = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM dental_inventory_items WHERE id=$1 AND active)`, id).Scan(&exists)
		if !exists {
			return dentalError(c, 404, "Insumo no encontrado")
		}
		return dentalError(c, 400, "El ajuste no puede dejar existencias negativas")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo ajustar el inventario")
	}
	if _, err = tx.Exec(`INSERT INTO dental_inventory_movements(item_id,quantity_delta,reason,performed_by) VALUES($1,$2,$3,$4)`, id, req.Delta, req.Reason, userID); err != nil {
		return dentalError(c, 500, "No se pudo registrar el movimiento")
	}
	if err = dentalAudit(tx, c, userID, "UPDATE", "inventory_item", strconv.FormatInt(id, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo completar el ajuste")
	}
	return c.JSON(fiber.Map{"id": id, "quantity": quantity})
}

type dentalTreatmentItem struct {
	ID          int64   `json:"id"`
	Description string  `json:"description"`
	Tooth       string  `json:"tooth"`
	Phase       string  `json:"phase"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
}

type dentalTreatmentPlan struct {
	ID          int64                 `json:"id"`
	PatientID   string                `json:"patientId"`
	PatientName string                `json:"patientName"`
	Title       string                `json:"title"`
	Status      string                `json:"status"`
	TotalAmount float64               `json:"totalAmount"`
	CreatedAt   string                `json:"createdAt"`
	Items       []dentalTreatmentItem `json:"items"`
}

func (dc *DentalController) ListTreatmentPlans(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	rows, err := config.DB.Query(`SELECT tp.id,p.code,p.full_name,tp.title,tp.status,tp.total_amount,to_char(tp.created_at AT TIME ZONE 'America/Guayaquil','YYYY-MM-DD HH24:MI'),COALESCE((SELECT json_agg(json_build_object('id',ti.id,'description',ti.description,'tooth',ti.tooth_number,'phase',ti.phase,'amount',ti.amount,'status',ti.status) ORDER BY ti.id) FROM dental_treatment_items ti WHERE ti.plan_id=tp.id),'[]'::json) FROM dental_treatment_plans tp JOIN dental_patients p ON p.id=tp.patient_id ORDER BY tp.created_at DESC LIMIT 500`)
	if err != nil {
		return dentalError(c, 500, "No se pudieron consultar los planes")
	}
	defer rows.Close()
	plans := make([]dentalTreatmentPlan, 0)
	for rows.Next() {
		var p dentalTreatmentPlan
		var itemsJSON []byte
		if err := rows.Scan(&p.ID, &p.PatientID, &p.PatientName, &p.Title, &p.Status, &p.TotalAmount, &p.CreatedAt, &itemsJSON); err != nil {
			return dentalError(c, 500, "No se pudieron leer los planes")
		}
		if err := json.Unmarshal(itemsJSON, &p.Items); err != nil {
			return dentalError(c, 500, "No se pudieron leer los procedimientos")
		}
		plans = append(plans, p)
	}
	if rows.Err() != nil {
		return dentalError(c, 500, "No se pudieron leer los planes")
	}
	if _, err := config.DB.Exec(`INSERT INTO dental_audit_log(user_id,action,entity,entity_id,source_ip) VALUES($1,'READ','treatment_plans','list',NULLIF($2,'')::inet)`, userID, c.IP()); err != nil {
		return dentalError(c, 500, "No se pudo registrar el acceso a los planes")
	}
	return c.JSON(plans)
}

func (dc *DentalController) CreateTreatmentPlan(c *fiber.Ctx) error {
	userID, ok := dentalUserID(c)
	if !ok {
		return dentalError(c, 401, "Sesión inválida")
	}
	var req struct {
		PatientID string                `json:"patientId"`
		Title     string                `json:"title"`
		Items     []dentalTreatmentItem `json:"items"`
	}
	if err := c.BodyParser(&req); err != nil {
		return dentalError(c, 400, "Plan de tratamiento inválido")
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.PatientID == "" || req.Title == "" || len(req.Items) == 0 {
		return dentalError(c, 400, "Paciente, nombre y al menos un procedimiento son obligatorios")
	}
	total := 0.0
	for i := range req.Items {
		req.Items[i].Description = strings.TrimSpace(req.Items[i].Description)
		if req.Items[i].Description == "" || req.Items[i].Amount < 0 {
			return dentalError(c, 400, "Revisa los procedimientos y sus montos")
		}
		if req.Items[i].Phase == "" {
			req.Items[i].Phase = "Fase 1"
		}
		if req.Items[i].Status == "" {
			req.Items[i].Status = "Pendiente"
		}
		total += req.Items[i].Amount
	}
	tx, err := config.DB.Begin()
	if err != nil {
		return dentalError(c, 500, "No se pudo iniciar el plan")
	}
	defer tx.Rollback()
	var patientID int
	err = tx.QueryRow(`SELECT id FROM dental_patients WHERE code=$1 AND active`, req.PatientID).Scan(&patientID)
	if errors.Is(err, sql.ErrNoRows) {
		return dentalError(c, 404, "Paciente no encontrado")
	}
	if err != nil {
		return dentalError(c, 500, "No se pudo validar el paciente")
	}
	var planID int64
	err = tx.QueryRow(`INSERT INTO dental_treatment_plans(patient_id,title,total_amount,created_by) VALUES($1,$2,$3,$4) RETURNING id`, patientID, req.Title, total, userID).Scan(&planID)
	if err != nil {
		return dentalError(c, 500, "No se pudo crear el plan")
	}
	for _, item := range req.Items {
		if _, err = tx.Exec(`INSERT INTO dental_treatment_items(plan_id,description,tooth_number,phase,amount,status) VALUES($1,$2,$3,$4,$5,$6)`, planID, item.Description, item.Tooth, item.Phase, item.Amount, item.Status); err != nil {
			return dentalError(c, 500, "No se pudieron agregar los procedimientos")
		}
	}
	if err = dentalAudit(tx, c, userID, "CREATE", "treatment_plan", strconv.FormatInt(planID, 10)); err != nil {
		return dentalError(c, 500, "No se pudo guardar la auditoría")
	}
	if err = tx.Commit(); err != nil {
		return dentalError(c, 500, "No se pudo guardar el plan")
	}
	return c.Status(201).JSON(fiber.Map{"id": planID, "patientId": req.PatientID, "title": req.Title, "status": "Pendiente", "totalAmount": total, "items": req.Items})
}
