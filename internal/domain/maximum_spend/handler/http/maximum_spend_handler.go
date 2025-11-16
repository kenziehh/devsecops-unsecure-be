package http

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kenziehh/cashflow-be/internal/domain/maximum_spend/dto"
	"github.com/kenziehh/cashflow-be/internal/domain/maximum_spend/service"
)

type MaximumSpendHandler struct {
	service  service.MaximumSpendService
	validate *validator.Validate
}

func NewMaximumSpendHandler(svc service.MaximumSpendService) *MaximumSpendHandler {
	return &MaximumSpendHandler{
		service:  svc,
		validate: validator.New(),
	}
}

// Update Maximum Spend godoc
// @Summary Set or update maximum spend limits
// @Description Set or update daily, monthly, and yearly maximum spend limits for the authenticated user
// @Tags maximum-spends
// @Accept json
// @Produce json
// @Param request body dto.MaximumSpendRequest true "Maximum Spend Request"
// @Success 200 {object} dto.MaximumSpendResponse

// @Router /maximum-spends [post]
func (h *MaximumSpendHandler) SetMaximumSpend(c *fiber.Ctx) error {
	var req dto.MaximumSpendRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	userIDStr := c.Locals("user_id")
	if userIDStr == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	ms, err := h.service.SetMaximumSpend(c.Context(), userID, req.DailyLimit, req.MonthlyLimit, req.YearlyLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	resp := dto.MaximumSpendResponse{
		ID:           ms.ID,
		UserID:       ms.UserID.String(),
		DailyLimit:   ms.DailyLimit,
		MonthlyLimit: ms.MonthlyLimit,
		YearlyLimit:  ms.YearlyLimit,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// Get Maximum Spend godoc
// @Summary Get maximum spend limits
// @Description Retrieve the daily, monthly, and yearly maximum spend limits for the authenticated user
// @Tags maximum-spends
// @Accept json
// @Produce json
// @Success 200 {object} dto.MaximumSpendResponse

// @Router /maximum-spends [get]
func (h *MaximumSpendHandler) GetMaximumSpend(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id")
	if userIDStr == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	ms, err := h.service.GetMaximumSpend(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	resp := dto.MaximumSpendResponse{
		ID:           ms.ID,
		UserID:       ms.UserID.String(),
		DailyLimit:   ms.DailyLimit,
		MonthlyLimit: ms.MonthlyLimit,
		YearlyLimit:  ms.YearlyLimit,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
