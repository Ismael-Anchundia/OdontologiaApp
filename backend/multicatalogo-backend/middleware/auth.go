// Paquete middleware contiene los middlewares de autenticación y autorización
package middleware

import (
	"database/sql"
	"log"
	"multicatalogo-backend/auth"
	"multicatalogo-backend/config"
	"multicatalogo-backend/models"
	"os"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth verifica que el token JWT sea válido
func RequireAuth(c *fiber.Ctx) error {
	// Extraer token del header Authorization
	authHeader := c.Get("Authorization")
	tokenString, err := auth.ExtractToken(authHeader)
	if err != nil {
		return c.Status(401).JSON(models.APIError{
			Status:  401,
			Message: "No autorizado: token faltante o inválido",
		})
	}

	// Validar token
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		log.Printf("Error validando token: %v", err)
		return c.Status(401).JSON(models.APIError{
			Status:  401,
			Message: "No autorizado: token inválido o expirado",
		})
	}

	// Guardar claims en el contexto para uso posterior
	c.Locals("user_id", claims.Sub)
	c.Locals("email", claims.Email)
	c.Locals("rol", claims.Rol)
	// Las rutas esperan un int, no el *int usado para representar un claim opcional.
	if claims.ReferidoID != nil {
		c.Locals("referido_id", *claims.ReferidoID)
	} else {
		c.Locals("referido_id", nil)
	}

	return c.Next()
}

// RequireDentalRoles autoriza un endpoint clínico por perfil dental activo.
func RequireDentalRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(int)
		if !ok || userID < 1 {
			return c.Status(401).JSON(models.APIError{Status: 401, Message: "Sesión inválida"})
		}
		if config.DB == nil {
			return c.Status(503).JSON(models.APIError{Status: 503, Message: "Autorización clínica no disponible"})
		}
		var role string
		err := config.DB.QueryRow(`SELECT role FROM dental_staff_roles WHERE user_id=$1 AND active`, userID).Scan(&role)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(403).JSON(models.APIError{Status: 403, Message: "La cuenta no tiene un perfil clínico activo"})
			}
			return c.Status(503).JSON(models.APIError{Status: 503, Message: "No se pudo comprobar el perfil clínico"})
		}
		for _, allowed := range roles {
			if role == allowed {
				c.Locals("dental_role", role)
				return c.Next()
			}
		}
		return c.Status(403).JSON(models.APIError{Status: 403, Message: "El perfil clínico no permite esta acción"})
	}
}

// RequireAdmin verifica que el usuario tenga rol admin
func RequireAdmin(c *fiber.Ctx) error {
	// Verificar que RequireAuth ya se ejecutó
	rol := c.Locals("rol")
	if rol == nil {
		return c.Status(401).JSON(models.APIError{
			Status:  401,
			Message: "No autorizado: sesión no iniciada",
		})
	}

	// Verificar que el rol sea admin
	if rol != "admin" {
		return c.Status(403).JSON(models.APIError{
			Status:  403,
			Message: "Prohibido: se requiere rol admin",
		})
	}

	return c.Next()
}

// SanitizeError sanitiza los errores según el entorno
func SanitizeError(apiError models.APIError) models.APIError {
	// Solo mostrar detalles en desarrollo
	if os.Getenv("APP_ENV") != "development" {
		apiError.Details = nil
	}
	return apiError
}
