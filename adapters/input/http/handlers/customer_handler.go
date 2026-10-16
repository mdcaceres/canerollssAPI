package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type CustomerHandler struct {
	customerUC *usecase.CustomerUseCase
}

func NewCustomerHandler(uc *usecase.CustomerUseCase) *CustomerHandler {
	return &CustomerHandler{customerUC: uc}
}

func (h *CustomerHandler) LookupByWhatsapp(c *fiber.Ctx) error {
	whatsapp := c.Query("whatsapp")
	if whatsapp == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "whatsapp query param is required",
		})
	}

	customer, err := h.customerUC.LookupByWhatsapp(whatsapp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if customer == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(customer)
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	customer, err := h.customerUC.Create(req.Name, req.Whatsapp)
	if err != nil {
		if err.Error() == "whatsapp already registered" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(customer)
}

func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid customer id",
		})
	}

	customer, err := h.customerUC.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(customer)
}
