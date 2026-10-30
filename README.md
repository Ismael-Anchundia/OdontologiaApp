# SIGO-Dental / Dentalia

MVP odontológico adaptado de Multicatálogo con interfaz React/TypeScript y API Go/Fiber/PostgreSQL. El PDF especifica requisitos funcionales y no impone un stack; se conservó el stack local para aprovechar la copia existente.

## Ejecución local

### Demo visual (sin backend)

```powershell
cd frontend/Practica03_a_05_de_la_Unidad
npm ci
$env:VITE_DEMO_MODE = "true"
npm run dev
```

Selecciona **Entrar a la demostración**. Emplea datos ficticios y `localStorage`; no requiere usuarios ni servicios externos.

### Modo conectado (API + PostgreSQL)

1. Desde `backend/multicatalogo-backend`, ejecuta `docker compose up -d` (PostgreSQL 16 en el puerto **5434** y Redis en 6379). Si Docker Hub limita la descarga (error 429), usa el mirror oficial: `docker pull mirror.gcr.io/library/postgres:16` y `docker tag mirror.gcr.io/library/postgres:16 postgres:16`.
2. Copia `.env.example` a `.env` y define un `JWT_SECRET` aleatorio de 32 caracteres o más. Una base nueva aplica las seis migraciones automáticamente. En una base existente aplica, en orden, `005_dental.sql`, `006_dental_access.sql` y `007_dental_notifications.sql` (son idempotentes).
3. Ejecuta `go run .` en el backend (puerto 3000).
4. Inicia Vite sin `VITE_DEMO_MODE` y entra con una cuenta que tenga perfil dental activo. En desarrollo, `admin@upse.edu.ec` / `123456` recibe el perfil `admin` en `006_dental_access.sql`. Al iniciar sesión el frontend consulta `/api/dental/me` y solo muestra los módulos que permite ese perfil.

Los pasos detallados, la verificación con `curl` y las pruebas E2E con PostgreSQL están en [backend/multicatalogo-backend/README.md](backend/multicatalogo-backend/README.md).

La contraseña inicial heredada solo sirve para desarrollo local; cámbiala antes de exponer cualquier servicio.

## Funcionalidad

- Pacientes, antecedentes médicos y alertas visibles.
- Agenda por día/semana, filtros por profesional/sillón, estados y prevención de solapamientos.
- Odontograma FDI/Universal con dentición temporal, superficies y categorías odontológicas.
- Periodontograma de seis sitios con cálculo de NIC.
- Evolución SOEP con notas append-only y auditoría inmutable.
- Presupuestos por fases y procedimientos, estados y sumatoria de importes.
- Existencias, ajustes y ledger de movimientos.
- Administración de perfiles dentales por rol y endpoints con denegación por defecto.
- Cola de confirmación y recordatorios de cita a 48 y 24 horas; adaptador webhook configurable.
- Interfaz adaptable, modo demo y manifiesto PWA.

## Configuración de notificaciones

Sin credenciales, citas y recordatorios quedan guardados en la outbox sin enviar. Para conectar un gateway compatible configura `NOTIFICATIONS_WEBHOOK_URL` y `NOTIFICATIONS_WEBHOOK_SECRET`; en producción la URL debe usar HTTPS. El servicio recibe JSON con `id`, `channel`, `recipient`, `template`, `message` y `scheduledAt`, cabecera Bearer e `Idempotency-Key`; debe responder HTTP 2xx y opcionalmente `X-Delivery-ID`. El worker reintenta y deja de reintentar después de cinco fallos. En la cita se usa WhatsApp como canal lógico cuando hay teléfono, o correo cuando solo hay email; el gateway determina el transporte real.

## API

La descripción OpenAPI 3 está en [openapi.yaml](openapi.yaml) e incluye todas las rutas dentales y `/api/login`, con los perfiles autorizados en cada una. Todas las rutas dentales requieren JWT y un rol dental activo. Roles: `admin`, `receptionist`, `dentist`, `assistant`, `auditor`. Los perfiles se administran desde Personal y roles o vía `/api/dental/staff`.

## Despliegue en Render

`render.yaml` es un Blueprint con tres recursos:

| Recurso | Tipo | Qué hace |
|---|---|---|
| `dentalia-db` | PostgreSQL (free) | Base de datos. |
| `dentalia-api` | Web service Docker (`backend/multicatalogo-backend/Dockerfile`) | API Go. Al arrancar aplica las migraciones pendientes (tabla `dental_schema_migrations`) y prepara el administrador. Health check: `/api/health`. |
| `dentalia-web` | Static site | Frontend conectado; `VITE_API_URL` toma el host de la API y el backend acepta ese host en CORS. |

Pasos:

1. Sube el repositorio a GitHub con estos archivos.
2. En Render: **New → Blueprint**, elige el repositorio y la rama.
3. Render pide `DENTAL_ADMIN_EMAIL` y `DENTAL_ADMIN_PASSWORD` (mínimo 12 caracteres): será la cuenta administradora clínica. Las variables de notificaciones pueden quedar vacías.
4. Aplica el Blueprint y espera a que `dentalia-api` quede *Live*; luego entra a la URL de `dentalia-web` con esa cuenta y crea al resto del personal desde **Personal y roles** (las cuentas deben existir en `usuarios`).

En producción (`APP_ENV=production`): `JWT_SECRET` se genera solo, `/api/register` queda deshabilitado (salvo `ALLOW_PUBLIC_REGISTER=true`) y la cuenta heredada `admin@upse.edu.ec` se desactiva si conserva la contraseña `123456`. Si no hay ningún administrador clínico activo, la API no arranca y lo indica en el log.

Limitaciones del plan gratuito: la API se duerme tras inactividad (primer acceso lento) y la base gratuita de Render caduca a los 30 días; para uso continuo cambia `plan` a uno de pago. Redis no se despliega: solo lo usan módulos heredados. No uses datos clínicos reales mientras falten MFA, cifrado en reposo, respaldos y lo pendiente en `SRS_CUMPLIMIENTO.md`.

Para publicar solo la demo sin servidor, crea un static site con `VITE_DEMO_MODE=true`.

## Alcance y validación

Consulta [SRS_CUMPLIMIENTO.md](SRS_CUMPLIMIENTO.md) para la matriz de requisitos, los pendientes externos y los límites de las pruebas. No introducir información clínica real hasta completar seguridad, firma, consentimiento, facturación, proveedores y validación clínica/legal.
