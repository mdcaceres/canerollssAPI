package handlers

import (
	"canerollss/adapters/input/http/dto"
	"canerollss/core/usecase"
	"canerollss/utils"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ReportHandler struct {
	reportUC *usecase.ReportUseCase
}

func NewReportHandler(uc *usecase.ReportUseCase) *ReportHandler {
	return &ReportHandler{reportUC: uc}
}

func (h *ReportHandler) GetReportHistory(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.Query("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	history, total, err := h.reportUC.GetReportHistory(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":      history,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *ReportHandler) CloseMonth(c *fiber.Ctx) error {
	var req dto.CloseMonthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	if validationErrors := utils.ValidateStruct(req); validationErrors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": validationErrors})
	}

	closing, err := h.reportUC.CloseMonth(req.Year, req.Month)
	if err != nil {
		if err.Error() == "invalid month or year" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "monthly closing ready",
		"data":    closing,
	})
}

func (h *ReportHandler) GetMonthlyClosing(c *fiber.Ctx) error {
	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid month"})
	}

	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid year"})
	}

	closing, err := h.reportUC.GetMonthlyClosing(month, year)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "monthly closing not found"})
		}
		if err.Error() == "invalid month or year" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": closing})
}
