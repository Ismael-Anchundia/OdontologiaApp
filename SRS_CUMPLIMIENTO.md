# Matriz de cumplimiento del SRS — SIGO-Dental

Base de contraste: *Documento de Requerimientos de Software - Sistema Odontológico.md.pdf* (SRS v1.0, 29/10/2026, 7 páginas). Estados: **Implementado** (funciona y tiene evidencia), **Parcial**, **Pendiente**, **No validado** (requiere entorno o medición que no existe).

Evidencia automatizada: `backend/multicatalogo-backend/e2e/dental_e2e_test.go` (8 pruebas E2E contra PostgreSQL 16, ver README del backend). Cobertura de esas pruebas: 48,7 % de las sentencias de `controllers` + `middleware` (incluye controladores heredados sin pruebas); los handlers dentales quedan entre 62 % y 100 %.

## Requerimientos funcionales

| Req. | Estado | Evidencia / brecha |
|---|---|---|
| RF-01 Registro unificado | Implementado | Alta, consulta/búsqueda y modificación de ficha con contacto de emergencia, tutor y seguro. E2E `TestPatients` (alta, duplicado 409, validaciones 400, edición, búsqueda). |
| RF-02 Anamnesis y alertas | Implementado | Historia médica obligatoria (400 si falta) y alerta crítica visible en la cabecera del expediente. Las alertas son texto libre, no un catálogo estructurado de enfermedades/alergias. |
| RF-03 Odontograma interactivo | Parcial | FDI/Universal, 32 permanentes + 20 temporales, superficies y catálogo de hallazgos con validación de pieza/superficie/tipo (`TestOdontogram`). Representación funcional, no anatómica validada clínicamente. El hallazgo vigente se sobrescribe por pieza/superficie: no queda historial versionado del odontograma (solo auditoría del cambio). |
| RF-04 Periodontograma | Implementado | Seis sitios, profundidad, margen, NIC calculado en PostgreSQL, furca, movilidad y sangrado; historial conservado y lectura de la última medición (`TestPeriodontogram`). |
| RF-05 Notas SOEP | Parcial | Bitácora cronológica append-only: no hay ruta de edición y PostgreSQL rechaza `UPDATE`/`DELETE` (`TestImmutableClinicalNotes`). Diferencias: el bloqueo es inmediato (no hay ventana configurable de 24 h), no hay firma digital del profesional ni endpoint de adendas (la columna `amended_note_id` existe sin flujo). |
| RF-06 Consentimiento informado | Pendiente | Sin plantillas, versionado ni captura biométrica/OTP. Requiere proveedor de firma/OTP y revisión legal. |
| RF-07 Presupuestos multinivel | Parcial | Procedimientos por pieza y fase, montos y total calculado (`TestTreatmentPlans`). Faltan costo de laboratorio, descuentos autorizados y control de cambios. |
| RF-08 Aprobación total/parcial y tareas | Parcial | Solo estado global del plan (Pendiente/Aprobado/…). No hay aprobación por ítem ni conversión de ítems aprobados en tareas o citas. |
| RF-09 Agenda por sillón/profesional | Parcial | Vistas día/semana y filtros por sillón y profesional; consulta por fecha o rango (`TestAppointmentsAndOverlap`). Sillones y profesionales están fijos en el frontend; el profesional se guarda como texto y `professional_user_id` registra a quien agenda; la UI siempre agenda 30 min (la API acepta 5–480). |
| RF-10 Estados de cita | Implementado | Los siete estados del SRS; estado inválido 400; reactivar una cita cancelada que choque devuelve 409. |
| RF-11 Notificaciones 48 h/24 h | Implementado con proveedor pendiente | Confirmación y recordatorios en outbox transaccional, deduplicados y cancelados al cancelar la cita (verificado en E2E). El envío real necesita `NOTIFICATIONS_WEBHOOK_URL`; no hay integración directa con Meta/Twilio. |
| RF-12 POS y facturación electrónica | Pendiente | Sin caja, pagos mixtos ni emisión SRI. Requiere proveedor autorizado y ambiente de pruebas SRI; no se simulan facturas autorizadas. |
| RF-13 Liquidación de honorarios | Pendiente | Las comisiones existentes son del Multicatálogo heredado (`/api/comisiones`) y no aplican a odontólogos. |
| RF-14 Descargo automático de insumos | Parcial | Insumos, ajustes manuales y kardex de movimientos; impide existencias negativas (`TestInventory`). Falta receta de consumo por procedimiento y descargo al marcar el procedimiento como realizado. |

## Requerimientos no funcionales

| Req. | Estado | Evidencia / brecha |
|---|---|---|
| RNF-01 Respuesta < 1,5 s | No validado | Sin pruebas de rendimiento ni entorno comparable. |
| RNF-02 Disponibilidad 99,9 % | No validado | `render.yaml` define API + PostgreSQL + frontend con health check, migraciones al arrancar y administrador por variables; aún no se ha desplegado ni medido. El plan gratuito de Render no ofrece SLA. |
| RNF-03 150 usuarios concurrentes | No validado | Sin pruebas de carga. |
| RNF-04 TLS 1.3 / AES-256 en reposo | Pendiente | Depende de infraestructura (proxy TLS, cifrado de disco/BD administrada). No configurado en el repo. |
| RNF-05 RBAC, MFA, cierre por inactividad | Parcial | RBAC con cinco perfiles y denegación por defecto (`TestPermissions`: 401 sin token, 403 sin perfil o perfil no autorizado, desactivación de perfil). Cierre de sesión a los 15 min de inactividad en el frontend (`AuthContext.tsx`). Falta MFA. El frontend consulta `/api/dental/me` al iniciar sesión y oculta/bloquea los módulos no permitidos. En producción se desactiva la cuenta heredada con contraseña de ejemplo y se cierra `/api/register`. |
| RNF-06 Auditoría inmutable | Parcial | `dental_audit_log` append-only (verificado en E2E) con usuario, acción, entidad, fecha (`TIMESTAMPTZ`) e IP en altas, cambios y consultas de listados dentales. No existe exportación que auditar; los módulos heredados no se auditan. |
| RNF-07 Responsive | Parcial | Layout adaptable y PWA; sin pruebas en las resoluciones/tablets indicadas. |
| RNF-08 Navegadores | No validado | Sin matriz de pruebas Chrome/Firefox/Edge/Safari. |

## Reglas de negocio

| Regla | Estado | Evidencia / brecha |
|---|---|---|
| RN-01 Inmutabilidad del historial | Parcial | Notas y auditoría no se pueden modificar ni borrar. Los hallazgos del odontograma sí se sobrescriben/eliminan (marcar "Sano") y no hay adendas firmadas. |
| RN-02 Cierre de caja ciego | Pendiente | No existe módulo de caja. |
| RN-03 Sin solapamiento de sillones | Implementado | Restricción `EXCLUDE` en PostgreSQL; `TestAppointmentsAndOverlap` prueba solape parcial (409), contiguas y otro sillón (201) y reutilización tras cancelar. |

## Integraciones, entregables y aceptación

| Punto | Estado | Evidencia / brecha |
|---|---|---|
| Facturación fiscal (SRI) | Pendiente | Sin proveedor ni credenciales. |
| Pasarela de pagos | Pendiente | Sin proveedor ni credenciales. |
| Gateway de mensajería | Parcial | Adaptador webhook genérico configurable; falta el proveedor real. |
| PACS/DICOM | Fuera de alcance (futuro) | No aplica a Fase 1. |
| Almacenamiento de archivos (Rx, fotos, PDF) | Pendiente | No hay carga de archivos. |
| OpenAPI 3.0 | Implementado | `openapi.yaml` revisado contra `routes/routes.go`: cubre todas las rutas dentales y el login. |
| Cobertura ≥ 80 % en endpoints críticos | Parcial | Pruebas E2E de los flujos dentales críticos; cobertura medida 62–100 % por handler dental, no llega al 80 % global. |
| DAST/SAST sin altas/críticas | No validado | No se ejecutaron escáneres. |

## Pendientes para declarar conformidad

1. Consentimiento informado versionado con firma biométrica u OTP (proveedor + revisión legal).
2. Firma digital del profesional, ventana de edición configurable y adendas firmadas para SOEP; historial versionado del odontograma.
3. Aprobación por ítem del presupuesto, costo de laboratorio, descuentos autorizados y conversión a tareas/citas.
4. Caja/POS con pagos mixtos, cierre ciego (RN-02), facturación SRI vía proveedor autorizado y pasarela de pagos.
5. Reglas de honorarios de odontólogos y receta de consumo de insumos con descargo automático.
6. MFA, TLS 1.3, cifrado en reposo, respaldos con restauración probada.
7. Despliegue de API y BD administrada; pruebas de rendimiento, carga, navegadores/dispositivos y SAST/DAST.
8. Ajustes del frontend: profesional/sillón/duración desde la API (hoy fijos en la UI).

No se simulan firmas legales, facturas autorizadas, pagos ni envíos de mensajes. El MVP conectado sirve como base de desarrollo y no debe manejar expedientes reales.
