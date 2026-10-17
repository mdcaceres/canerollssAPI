package middleware

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Cookies("session_token")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected token algorithm")
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}

		return c.Next()
	}
}

func StoreOperator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "operator authentication is not configured"})
		}
		token, err := jwt.Parse(c.Cookies("session_token"), func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected token algorithm")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.SendStatus(fiber.StatusForbidden)
		}
		role, _ := claims["role"].(string)
		if role != "ADMIN" && role != "EMPLOYEE" {
			return c.SendStatus(fiber.StatusForbidden)
		}
		return c.Next()
	}
}
