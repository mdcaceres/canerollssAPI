package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"canerollss/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUC *usecase.AuthUseCase
}

func NewAuthHandler(uc *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: uc}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var input dto.RegisterRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	if validationErrors := utils.ValidateStruct(input); validationErrors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": validationErrors})
	}

	user := &domain.User{
		Username: input.Username,
		Role:     domain.RoleEmployee,
		IsActive: true,
	}

	if err := h.authUC.Register(user, input.Password); err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User created successfully"})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input dto.LoginRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	if validationErrors := utils.ValidateStruct(input); validationErrors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": validationErrors})
	}

	token, user, err := h.authUC.Login(input.Username, input.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Session logout successfully"})
}
