package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/domain"
	"canerollss/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	orderUC *usecase.OrderUseCase
}

func NewOrderHandler(uc *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{orderUC: uc}
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req dto.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	order := &domain.Order{
		Note: req.Note,
	}

	for _, item := range req.Items {
		order.Items = append(order.Items, domain.OrderItem{
			ToppingID: item.ToppingID,
			Quantity:  item.Quantity,
			IsCombo:   item.IsCombo,
		})
	}

	if err := h.orderUC.CreateOrder(order, req.CustomerName, req.CustomerWhatsapp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *OrderHandler) CancelOrder(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order id",
		})
	}
	if err := h.orderUC.CancelOrder(uint(id)); err != nil {
		if err.Error() == "order is already cancelled" || err.Error() == "order not found" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "order cancelled",
	})
}
