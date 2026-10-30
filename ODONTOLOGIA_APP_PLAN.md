# Adaptación de Multicatálogo a SIGO-Dental

## Arquitectura conservada

- Interfaz: React 19, TypeScript, Vite, Tailwind CSS y React Router.
- API: Go con Fiber.
- Persistencia: PostgreSQL; Redis continúa opcional para la API heredada.
- Se conservó este stack para aprovechar el proyecto; el PDF propone tecnologías, pero no obliga a usar alguna.

## MVP local implementado

- [x] Reemplazar la navegación de tienda, catálogo, red y pedidos por módulos clínicos.
- [x] Crear resumen de actividad y alertas médicas.
- [x] Crear fichas con identificación, contacto de emergencia, tutor, seguro, antecedentes y alertas.
- [x] Crear agenda con estados y bloqueo en PostgreSQL de solapamientos por sillón y duración.
- [x] Crear odontograma FDI, periodontograma con NIC calculado y notas SOEP append-only.
- [x] Crear planes de tratamiento con fases e importes.
- [x] Crear inventario con ajustes, ledger de movimientos y aviso de stock mínimo.
- [x] Añadir migración dental `005_dental.sql`, endpoints protegidos y registro de auditoría append-only.
- [x] Conservar el login heredado en desarrollo y crear modo demo estático independiente.

## Fase siguiente para producción clínica

- [ ] Retirar el esquema heredado de referidos, productos y pedidos cuando deje de usarse para autenticación.
- [ ] Implementar RBAC clínico por rol y permisos por clínica; actualmente todas las rutas dentales requieren el rol heredado `admin`.
- [ ] Implementar MFA, cierre por inactividad y revisión de sesiones.
- [ ] Añadir consentimiento informado, firma digital y flujo de adendas firmadas.
- [ ] Completar presupuestos multinivel, cobros, caja y facturación electrónica local.
- [ ] Implementar consumo de inventario por procedimiento, avisos a pacientes y proveedores externos.
- [ ] Configurar cifrado y respaldos del PostgreSQL, secretos, TLS, observabilidad y pruebas de seguridad.

## Límites actuales

El modo conectado persiste los módulos dentales en PostgreSQL, pero la autenticación y tabla de usuarios siguen heredadas de Multicatálogo. No hay MFA, RBAC clínico por perfiles, cifrado del expediente ni respaldos configurados. El Blueprint estático de Render funciona como demo independiente con datos ficticios en `localStorage`. No registrar pacientes reales ni usarlo como expediente clínico.
