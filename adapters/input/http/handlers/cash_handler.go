package handlers

import (
	"canerollss/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type CashHandler struct {
	cashUC *usecase.CashUseCase
}

func NewCashHandler(uc *usecase.CashUseCase) *CashHandler {
	return &CashHandler{cashUC: uc}
}

func (h *CashHandler) OpenRegister(c *fiber.Ctx) error {
	if err := h.cashUC.OpenRegister(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Cash register has been opened"})
}

func (h *CashHandler) CloseRegister(c *fiber.Ctx) error {
	closedRegister, err := h.cashUC.CloseRegister()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Cash register has been closed",
		"data":    closedRegister,
	})
}
