# Plan de adaptación SIGO-Dental

Se conservó la aplicación Multicatálogo ubicada en esta carpeta para acelerar la tarea. El PDF especifica flujos/requisitos, no un stack tecnológico. La arquitectura actual es React 19 + TypeScript + Vite/Tailwind, API Go/Fiber y PostgreSQL; Render publica únicamente el modo estático de demostración.

## Trabajo realizado

- [x] Sustituir la navegación y módulos del catálogo por el tablero Dentalia.
- [x] Fichas de pacientes, antecedentes médicos y alertas prioritarias.
- [x] Agenda diaria/semanal, estados y bloqueo de solapamientos de sillón.
- [x] Odontograma FDI y Universal con dentición temporal, superficies y categorías.
- [x] Periodontograma de seis sitios por pieza y cálculo de NIC.
- [x] Notas SOEP inmutables con bitácora append-only.
- [x] Presupuestos por fases y múltiples procedimientos con seguimiento de estado global.
- [x] Inventario, ajuste de existencias y ledger de movimientos.
- [x] Perfiles dentales y control de acceso por ruta con denegación por defecto.
- [x] Cola de confirmación/recordatorios con idempotencia, reintento y webhook configurable.
- [x] Modo de demostración desconectado, PWA y configuración estática de Render.
- [x] Documentación OpenAPI 3 inicial y matriz de correspondencia con el SRS.

## Trabajo externo requerido para conformidad integral

Consultar [SRS_CUMPLIMIENTO.md](SRS_CUMPLIMIENTO.md). Quedan pendientes: consentimientos y firma electrónica certificada, MFA/logout de inactividad, cobro POS, pagos mixtos, emisión electrónica SRI, comisiones dentales, aprobación granular, consumo automático por procedimiento, cifrado/backups de producción y pruebas completas de seguridad/carga/dispositivos. La ausencia de credenciales impide homologar los conectores externos.

## Comprobaciones locales

Se ejecutan `go test -p 1 ./...`, `npm run lint` y `npm run build`; la sección de cumplimiento registra sus resultados y las verificaciones que requieren base de datos o infraestructura. No se suben cambios ni se despliega sin instrucción del usuario.
