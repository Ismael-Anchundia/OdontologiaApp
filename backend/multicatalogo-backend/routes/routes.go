// Declaramos el paquete routes, cuya única responsabilidad es registrar los endpoints de la API.
package routes

import (
	"os"

	// Importamos Fiber para poder recibir y configurar la instancia principal de la aplicación.
	"github.com/gofiber/fiber/v2"
	// Importamos nuestro paquete de controladores, donde residen las funciones que se ejecutarán en cada ruta.
	"multicatalogo-backend/controllers"
	// Importamos el middleware de autenticación
	"multicatalogo-backend/middleware"
)

// SetupRoutes es una función que recibe un puntero a la aplicación Fiber (*fiber.App) y los controladores para inyectar las rutas.
func SetupRoutes(app *fiber.App, authController *controllers.AuthController, prodController *controllers.ProdController, redController *controllers.RedController, checkoutController *controllers.CheckoutController, registerController *controllers.RegisterController, adminProductosController *controllers.AdminProductosController, pedidosController *controllers.PedidosController, comisionesController *controllers.ComisionesController, liquidacionesController *controllers.LiquidacionesController, reportesController *controllers.ReportesController, usuariosController *controllers.UsuariosController, dentalController *controllers.DentalController) {
	// Definimos una ruta HTTP GET en "/api/health" para verificar el estado del servidor.
	app.Get("/api/health", controllers.Health)

	// SIGO-Dental usa perfiles propios para aplicar privilegio mínimo.
	app.Get("/api/dental/me", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist", "assistant", "auditor"), dentalController.DentalProfile)
	app.Get("/api/dental/staff", middleware.RequireAuth, middleware.RequireDentalRoles("admin"), dentalController.ListStaff)
	app.Put("/api/dental/staff/:id/role", middleware.RequireAuth, middleware.RequireDentalRoles("admin"), dentalController.UpsertStaffRole)
	app.Get("/api/dental/patients", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist", "assistant"), dentalController.ListPatients)
	app.Post("/api/dental/patients", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist"), dentalController.CreatePatient)
	app.Put("/api/dental/patients/:patientId", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist"), dentalController.UpdatePatient)
	app.Get("/api/dental/appointments", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist", "assistant"), dentalController.ListAppointments)
	app.Post("/api/dental/appointments", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist"), dentalController.CreateAppointment)
	app.Patch("/api/dental/appointments/:id/status", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "receptionist", "dentist", "assistant"), dentalController.UpdateAppointmentStatus)
	app.Get("/api/dental/patients/:patientId/notes", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.ListNotes)
	app.Post("/api/dental/patients/:patientId/notes", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.CreateNote)
	app.Get("/api/dental/patients/:patientId/odontogram", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.ListFindings)
	app.Put("/api/dental/patients/:patientId/odontogram", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.UpsertFinding)
	app.Get("/api/dental/patients/:patientId/periodontogram", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.ListPeriodontogram)
	app.Post("/api/dental/patients/:patientId/periodontogram/bulk", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.RecordPeriodontalChart)
	app.Post("/api/dental/patients/:patientId/periodontogram", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "assistant"), dentalController.RecordPeriodontalMeasurement)
	app.Get("/api/dental/inventory", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "assistant", "auditor"), dentalController.ListInventory)
	app.Post("/api/dental/inventory", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "assistant"), dentalController.CreateInventoryItem)
	app.Patch("/api/dental/inventory/:id", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "assistant"), dentalController.AdjustInventory)
	app.Get("/api/dental/treatment-plans", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "receptionist"), dentalController.ListTreatmentPlans)
	app.Post("/api/dental/treatment-plans", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist"), dentalController.CreateTreatmentPlan)
	app.Patch("/api/dental/treatment-plans/:id/status", middleware.RequireAuth, middleware.RequireDentalRoles("admin", "dentist", "receptionist"), dentalController.UpdateTreatmentPlanStatus)

	// Definimos una ruta HTTP POST en "/api/login" y la enlazamos al controlador con DI.
	app.Post("/api/login", authController.Login)

	// Definimos una ruta HTTP POST en "/api/register" (pública)
	if os.Getenv("APP_ENV") != "production" || os.Getenv("ALLOW_PUBLIC_REGISTER") == "true" {
		app.Post("/api/register", registerController.Register)
	}
	app.Get("/api/admin/usuarios", middleware.RequireAuth, middleware.RequireAdmin, usuariosController.Listar)
	app.Patch("/api/admin/usuarios/:id", middleware.RequireAuth, middleware.RequireAdmin, usuariosController.Actualizar)

	// Definimos una ruta HTTP GET en "/api/productos" y la enlazamos al controlador con DI.
	app.Get("/api/productos", prodController.GetProductos)

	// Definimos una ruta HTTP GET en "/api/productos/:id" y la enlazamos al controlador con DI.
	app.Get("/api/productos/:id", prodController.GetProductoPorID)

	// CRUD de productos (admin - protegido con RequireAdmin)
	app.Post("/api/admin/productos", middleware.RequireAuth, middleware.RequireAdmin, adminProductosController.CrearProducto)
	app.Put("/api/admin/productos/:id", middleware.RequireAuth, middleware.RequireAdmin, adminProductosController.ActualizarProducto)
	app.Delete("/api/admin/productos/:id", middleware.RequireAuth, middleware.RequireAdmin, adminProductosController.DesactivarProducto)
	app.Get("/api/admin/productos", middleware.RequireAuth, middleware.RequireAdmin, adminProductosController.ObtenerProductosAdmin)

	// Gestión de pedidos
	app.Get("/api/pedidos/mios", middleware.RequireAuth, pedidosController.ObtenerPedidosMios)
	app.Get("/api/pedidos", middleware.RequireAuth, middleware.RequireAdmin, pedidosController.ObtenerPedidosTodos)
	app.Get("/api/pedidos/:id", middleware.RequireAuth, pedidosController.ObtenerPedidoPorID)
	app.Patch("/api/pedidos/:id/estado", middleware.RequireAuth, middleware.RequireAdmin, pedidosController.ActualizarEstadoPedido)
	app.Post("/api/pedidos/:id/cancelar", middleware.RequireAuth, middleware.RequireAdmin, pedidosController.CancelarPedido)

	// Comisiones
	app.Get("/api/comisiones/mias", middleware.RequireAuth, comisionesController.ObtenerComisionesMias)
	app.Get("/api/comisiones/pendientes", middleware.RequireAuth, middleware.RequireAdmin, comisionesController.ObtenerComisionesPendientes)

	// Liquidaciones
	app.Post("/api/liquidaciones", middleware.RequireAuth, middleware.RequireAdmin, liquidacionesController.CrearLiquidacion)
	app.Get("/api/liquidaciones", middleware.RequireAuth, middleware.RequireAdmin, liquidacionesController.ObtenerLiquidacionesTodas)
	app.Get("/api/liquidaciones/mias", middleware.RequireAuth, liquidacionesController.ObtenerLiquidacionesMias)

	// Reportes
	app.Get("/api/reportes/resumen", middleware.RequireAuth, middleware.RequireAdmin, reportesController.ObtenerReporteResumen)
	app.Get("/api/reportes/top-productos", middleware.RequireAuth, middleware.RequireAdmin, reportesController.ObtenerTopProductos)
	app.Get("/api/reportes/top-referidos", middleware.RequireAuth, middleware.RequireAdmin, reportesController.ObtenerTopReferidos)
	app.Get("/api/reportes/comisiones-pendientes", middleware.RequireAuth, middleware.RequireAdmin, reportesController.ObtenerComisionesPendientes)

	// Definimos una ruta HTTP GET en "/api/red" y la enlazamos al controlador con DI (protegido con RequireAuth)
	app.Get("/api/red", middleware.RequireAuth, redController.GetRed)

	// Definimos una ruta HTTP GET en "/api/red/mia" - red del usuario autenticado
	app.Get("/api/red/mia", middleware.RequireAuth, redController.GetRedMia)

	// Definimos una ruta HTTP POST en "/api/checkout" y la enlazamos al controlador con DI (protegido con RequireAuth)
	app.Post("/api/checkout", middleware.RequireAuth, checkoutController.ProcesarCheckout)
}
