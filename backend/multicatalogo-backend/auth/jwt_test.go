package auth

import "testing"

func TestGenerateAndValidateDentalToken(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("JWT_SECRET", "a-random-test-secret-with-more-than-32-characters")
	t.Setenv("JWT_EXPIRATION_HOURS", "2")

	token, err := GenerateToken(42, "odontologo@example.test", "cliente", nil)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.Sub != 42 || claims.Email != "odontologo@example.test" || claims.Rol != "cliente" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestProductionRejectsMissingOrLegacySecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	for _, secret := range []string{"", "multicatalogo-secret-key-2026-very-secure", "short"} {
		t.Setenv("JWT_SECRET", secret)
		if _, err := GenerateToken(1, "admin@example.test", "admin", nil); err == nil {
			t.Errorf("GenerateToken() accepted unsafe production secret %q", secret)
		}
	}
}

func TestExtractToken(t *testing.T) {
	token, err := ExtractToken("Bearer abc123")
	if err != nil || token != "abc123" {
		t.Fatalf("ExtractToken() = %q, %v", token, err)
	}
	if _, err := ExtractToken("Basic abc123"); err == nil {
		t.Fatal("ExtractToken() accepted a non-Bearer scheme")
	}
}
