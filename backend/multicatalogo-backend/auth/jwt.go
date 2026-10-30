// Paquete auth maneja la autenticación y autorización con JWT
package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const localDevelopmentSecret = "sigo-dental-local-development-key-change-before-deployment"

func signingSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if (secret == "" || secret == "multicatalogo-secret-key-2026-very-secure") && os.Getenv("APP_ENV") != "production" {
		secret = localDevelopmentSecret
	}
	if len(secret) < 32 || (secret == "multicatalogo-secret-key-2026-very-secure" && os.Getenv("APP_ENV") == "production") {
		return nil, errors.New("JWT_SECRET must be at least 32 random characters and cannot be the legacy default")
	}
	if os.Getenv("APP_ENV") == "production" && secret == localDevelopmentSecret {
		return nil, errors.New("JWT_SECRET must be set explicitly in production")
	}
	return []byte(secret), nil
}

// Claims representa los claims del JWT
type Claims struct {
	Email      string `json:"email"`
	Rol        string `json:"rol"`
	Sub        int    `json:"sub"` // ID del usuario
	ReferidoID *int   `json:"referido_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken genera un token JWT para un usuario
func GenerateToken(userID int, email, rol string, referidoID *int) (string, error) {
	secret, err := signingSecret()
	if err != nil {
		return "", err
	}

	// Obtener tiempo de expiración
	expirationHours := os.Getenv("JWT_EXPIRATION_HOURS")
	if expirationHours == "" {
		expirationHours = "2" // 2 horas por defecto
	}

	hours, err := time.ParseDuration(expirationHours + "h")
	if err != nil || hours <= 0 {
		hours = 2 * time.Hour
	}
	expirationTime := time.Now().Add(hours)
	claims := &Claims{
		Email:      email,
		Rol:        rol,
		Sub:        userID,
		ReferidoID: referidoID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "multicatalogo-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateToken valida un token JWT y retorna los claims
func ValidateToken(tokenString string) (*Claims, error) {
	secret, err := signingSecret()
	if err != nil {
		return nil, fmt.Errorf("configuración JWT inválida: %w", err)
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("método de firma no permitido")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token inválido")
}

// ExtractToken extrae el token del header Authorization
func ExtractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("header Authorization faltante")
	}

	// El formato debe ser: "Bearer <token>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", errors.New("formato de Authorization inválido")
	}

	return authHeader[7:], nil
}
