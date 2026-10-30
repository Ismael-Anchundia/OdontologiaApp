# SIGO-Dental — demostración

Adaptación del frontend React y la API Go/PostgreSQL de Multicatálogo a un MVP odontológico. El detalle de alcance y la siguiente fase están en [ODONTOLOGIA_APP_PLAN.md](ODONTOLOGIA_APP_PLAN.md).

## Abrir la demostración local

```powershell
cd frontend/Practica03_a_05_de_la_Unidad
npm install
$env:VITE_DEMO_MODE = "true"
npm run dev
```

Al abrir la dirección local que muestra Vite, pulsa **Entrar a la demostración**; no requiere backend ni credenciales. Usa solo la información ficticia de ejemplo. Para volver al modo conectado, cierra esa terminal y ejecuta Vite sin `VITE_DEMO_MODE`.

## Iniciar el modo conectado

Inicia PostgreSQL y Redis según `backend/multicatalogo-backend/README.md`, levanta el backend y ejecuta el frontend sin `VITE_DEMO_MODE`. El login heredado usa la cuenta inicial `admin@upse.edu.ec` / `123456`. Para una base ya existente, aplica una sola vez `backend/multicatalogo-backend/database/005_dental.sql` antes de usar los módulos clínicos.

## Módulos de demostración

- Resumen y alertas médicas.
- Registro y búsqueda de pacientes.
- Agenda con estados y bloqueo de solapamientos en PostgreSQL.
- Odontograma FDI, notas SOEP inmutables y periodontograma con cálculo NIC.
- Tratamientos persistidos en PostgreSQL e inventario con ledger de ajustes.
- Reportes ilustrativos; facturación y caja siguen pendientes.

El modo conectado persiste pacientes, citas, notas, hallazgos, periodontograma, inventario y planes en PostgreSQL. El modo demo estático guarda datos ficticios en el navegador (`localStorage`, clave `sigo-dental-demo`). Para restablecerlos, borra esa clave desde DevTools y recarga.

## Publicar la demostración en Render

El `render.yaml` de esta carpeta configura un sitio estático de demostración. Conecta a Render un repositorio que contenga esta carpeta y despliega el Blueprint; el build activa `VITE_DEMO_MODE` para permitir el acceso sin conectar el login viejo. La URL pública sirve únicamente el prototipo y datos ficticios guardados en el navegador.

## Estado de producción

Esto no es todavía un sistema clínico listo para producción. El login y la tabla de usuarios aún son los de Multicatálogo; las rutas dentales requieren el rol `admin`. Faltan MFA, RBAC por perfil, cifrado de expedientes, consentimiento y firma digital, facturación fiscal y copias de seguridad. No registrar pacientes reales. El Blueprint de Render es solo una demo pública con datos ficticios.
