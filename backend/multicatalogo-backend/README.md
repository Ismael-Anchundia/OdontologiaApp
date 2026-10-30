# Dentalia backend — API Go/Fiber

Backend PostgreSQL y Redis heredado, adaptado con endpoints dentales. La API dental requiere token JWT y un rol dental activo. La configuración segura y las limitaciones del MVP están en el README raíz y `SRS_CUMPLIMIENTO.md`.

## Desarrollo local

Requisitos: Docker, Go 1.26 (el toolchain se descarga solo si tienes Go ≥ 1.21).

```powershell
Copy-Item .env.example .env
# Ajusta JWT_SECRET (32+ caracteres) y la conexión en .env

docker compose up -d          # PostgreSQL 16 en localhost:5434, Redis en 6379
go test -p 1 ./...
go run .                      # API en http://localhost:3000
```

Si `docker compose` falla con `429 Too Many Requests` al descargar `postgres:16`, descarga la imagen oficial desde el mirror de Google y vuelve a levantar:

```powershell
docker pull mirror.gcr.io/library/postgres:16
docker tag mirror.gcr.io/library/postgres:16 postgres:16
docker compose up -d
```

### Migraciones

Al arrancar, la API aplica en orden los scripts embebidos que aún no figuren en `dental_schema_migrations` (con bloqueo consultivo para varias instancias). Todos son idempotentes, así que una base creada por Docker solo los vuelve a ejecutar una vez.

Variables de arranque relevantes: `DATABASE_URL` (o `DB_*`), `JWT_SECRET`, `APP_ENV`, `CORS_ALLOWED_ORIGINS` (acepta hosts sin esquema y asume HTTPS), `DENTAL_ADMIN_EMAIL` + `DENTAL_ADMIN_PASSWORD` (crea o actualiza un administrador clínico) y `ALLOW_PUBLIC_REGISTER`.

Docker inicializa una base nueva con `init.sql`, `003_seguridad.sql`, `004_backoffice.sql`, `005_dental.sql`, `006_dental_access.sql` y `007_dental_notifications.sql`. Esos scripts solo corren al crear el volumen. Para una base existente, o para comprobar que siguen siendo idempotentes:

```bash
for f in init.sql 003_seguridad.sql 004_backoffice.sql 005_dental.sql 006_dental_access.sql 007_dental_notifications.sql; do
  docker exec -i multicatalogo-postgres psql -q -v ON_ERROR_STOP=1 -U multicatalogo -d multicatalogo < database/$f || break
done
```

Para recrear la base desde cero: `docker compose down -v` y `docker compose up -d`. Si tu volumen se creó antes de corregir `005_dental.sql`, las tablas dentales no existen (la migración abortaba); aplica de nuevo `005`, `006` y `007` o recrea el volumen.

### Verificación rápida

```bash
TOKEN=$(curl -s -X POST localhost:3000/api/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@upse.edu.ec","password":"123456"}' | python -c 'import json,sys;print(json.load(sys.stdin)["token"])')
curl -s localhost:3000/api/dental/me -H "Authorization: Bearer $TOKEN"
curl -s "localhost:3000/api/dental/appointments?date=2026-10-30" -H "Authorization: Bearer $TOKEN"
```

`/api/health` responde 503 solo si PostgreSQL no está disponible; si Redis falta informa `"redis":"down"` con 200.

## Pruebas E2E con PostgreSQL

`e2e/dental_e2e_test.go` levanta la app Fiber completa en proceso contra una base real y cubre pacientes, citas y solapamientos, notas inmutables, odontograma, periodontograma, presupuestos, inventario/kardex y permisos por perfil. Crea usuarios y datos propios con sufijos únicos, por lo que puede repetirse sobre la misma base; no la uses contra una base con datos reales. Sin la variable `DENTAL_E2E_DATABASE_URL` las pruebas se omiten.

```bash
export DENTAL_E2E_DATABASE_URL='postgres://multicatalogo:multicatalogo123@localhost:5434/multicatalogo?sslmode=disable'
go test -count=1 -v ./e2e
# Cobertura de controladores y middleware:
go test -count=1 -coverpkg=./controllers,./middleware -coverprofile=cover.out ./e2e && go tool cover -func=cover.out
```

PowerShell: `$env:DENTAL_E2E_DATABASE_URL = 'postgres://multicatalogo:multicatalogo123@localhost:5434/multicatalogo?sslmode=disable'`.

## Notificaciones

Sin `NOTIFICATIONS_WEBHOOK_URL` y `NOTIFICATIONS_WEBHOOK_SECRET`, la app mantiene la cola de entrega pendiente sin enviar mensajes. Configurar un HTTPS webhook en producción. El contrato acepta JSON de entrega con canal y destinatario, Authorization Bearer, `Idempotency-Key`, respuesta 2xx y, opcionalmente, `X-Delivery-ID`. El worker vence bloqueos obsoletos y reintenta hasta cinco veces.

## API dental

`../openapi.yaml` describe todas las rutas dentales (perfil y roles, pacientes, agenda, notas SOEP, odontograma, periodontograma, inventario y planes de tratamiento), sus cuerpos, códigos de respuesta y perfiles autorizados. Las brechas frente al SRS están en `../SRS_CUMPLIMIENTO.md`.

La API depende de PostgreSQL para persistencia. Redis solo se utiliza por funcionalidades heredadas de Multicatálogo y no es requisito de los módulos dentales. Para Render con backend conectado hace falta un servicio Go y PostgreSQL administrado; el archivo Blueprint actual es para frontend demo estático.
