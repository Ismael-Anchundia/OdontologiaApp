package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"multicatalogo-backend/config"
)

func TestDentalRolesFailClosedWithoutUserContext(t *testing.T) {
	app := fiber.New()
	app.Get("/clinical", RequireDentalRoles("dentist"), func(c *fiber.Ctx) error { return c.SendStatus(204) })
	response, err := app.Test(httptest.NewRequest("GET", "/clinical", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestDentalRolesFailClosedWhenDatabaseUnavailable(t *testing.T) {
	previous := config.DB
	config.DB = nil
	t.Cleanup(func() { config.DB = previous })
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", 7); return c.Next() })
	app.Get("/clinical", RequireDentalRoles("dentist"), func(c *fiber.Ctx) error { return c.SendStatus(204) })
	response, err := app.Test(httptest.NewRequest("GET", "/clinical", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 503 {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
}
