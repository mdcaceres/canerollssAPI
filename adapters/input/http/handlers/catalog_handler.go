package handlers

import (
	"canerollss/core/domain"
	"canerollss/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type CatalogHandler struct {
	catalogUC *usecase.CatalogUseCase
}

func NewCatalogHandler(uc *usecase.CatalogUseCase) *CatalogHandler {
	return &CatalogHandler{catalogUC: uc}
}

func (h *CatalogHandler) CreateTopping(c *fiber.Ctx) error {
	var topping domain.Topping
	if err := c.BodyParser(&topping); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.catalogUC.CreateTopping(&topping); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(topping)
}

func (h *CatalogHandler) ListToppings(c *fiber.Ctx) error {
	toppings, err := h.catalogUC.ListToppings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(toppings)
}
