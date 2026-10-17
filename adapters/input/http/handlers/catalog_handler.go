package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"canerollss/utils"

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

func (h *CatalogHandler) UpdateToppingPrices(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid topping id",
		})
	}

	var req dto.UpdateToppingPriceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.SinglePrice == nil && req.ComboPrice == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least one price (single_price or combo_price) must be provided",
		})
	}

	if validationErrors := utils.ValidateStruct(req); len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation failed",
			"details": validationErrors,
		})
	}

	updated, err := h.catalogUC.UpdateToppingPrices(uint(id), req.SinglePrice, req.ComboPrice)
	if err != nil {
		if err.Error() == "topping not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "topping not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(updated)
}

