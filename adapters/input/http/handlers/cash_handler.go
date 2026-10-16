package handlers

import (
	"canerollss/core/usecase"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type CashHandler struct {
	cashUC *usecase.CashUseCase
}

func NewCashHandler(uc *usecase.CashUseCase) *CashHandler {
	return &CashHandler{cashUC: uc}
}

func (h *CashHandler) GetCurrent(c *fiber.Ctx) error {
	reg, err := h.cashUC.GetActive()
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(reg)
}

func (h *CashHandler) OpenRegister(c *fiber.Ctx) error {
	reg, err := h.cashUC.OpenRegister()
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(reg)
}

func (h *CashHandler) CloseRegister(c *fiber.Ctx) error {
	closedRegister, err := h.cashUC.CloseRegister()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(closedRegister)
}

func (h *CashHandler) GetCashReports(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.Query("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	cash, total, err := h.cashUC.GetCashHistory(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":      cash,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}
