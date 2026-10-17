package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	orderUC    *usecase.OrderUseCase
	deliveryUC *usecase.DeliveryUseCase
}

func NewOrderHandler(uc *usecase.OrderUseCase, delivery ...*usecase.DeliveryUseCase) *OrderHandler {
	h := &OrderHandler{orderUC: uc}
	if len(delivery) > 0 {
		h.deliveryUC = delivery[0]
	}
	return h
}

func (h *OrderHandler) DispatchDelivery(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order id"})
	}
	if h.deliveryUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "delivery service unavailable"})
	}
	order, err := h.deliveryUC.Dispatch(context.Background(), uint(id))
	if err != nil {
		if order != nil && order.Status == domain.OrderStatusDispatched {
			return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"order_id": order.ID, "status": order.Status, "tracking_url": order.TrackingURL, "warning": err.Error()})
		}
		status := fiber.StatusBadGateway
		if strings.Contains(err.Error(), "must be PAID") || strings.Contains(err.Error(), "validated delivery address") || strings.Contains(err.Error(), "required") {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"order_id": order.ID, "status": order.Status, "external_delivery_id": order.ExternalDeliveryID, "tracking_url": order.TrackingURL})
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req dto.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	order := &domain.Order{
		CustomerID: req.CustomerID,
		Note:       req.Note,
	}
	for _, item := range req.Items {
		order.Items = append(order.Items, domain.OrderItem{
			ToppingID: item.ToppingID,
			Quantity:  item.Quantity,
			IsCombo:   item.IsCombo,
		})
	}

	if err := h.orderUC.CreateOrder(order); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "no open register found" || err.Error() == "customer not found" {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
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
		if err.Error() == "order is already cancelled" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "order cancelled"})
}
