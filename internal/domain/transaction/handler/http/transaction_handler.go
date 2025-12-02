package http

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kenziehh/cashflow-be/internal/domain/transaction/dto"
	"github.com/kenziehh/cashflow-be/internal/domain/transaction/service"
	"github.com/kenziehh/cashflow-be/pkg/errx"
	"github.com/kenziehh/cashflow-be/pkg/response"
)

type TransactionHandler struct {
	service  service.TransactionService
	validate *validator.Validate
}

func NewTransactionHandler(service service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		service:  service,
		validate: validator.New(),
	}
}

// CreateTransaction godoc
// @Summary Create a new transaction
// @Description Create a new transaction for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body dto.CreateTransactionRequest true "Create transaction request"
// @Success 201 {object} response.Response{data=entity.Transaction}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions [post]
func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	// Parse JSON
	var req dto.CreateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return errx.NewBadRequestError("Invalid request body")
	}

	// Validasi input JSON
	if err := h.validate.Struct(req); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			field := err.Field()
			tag := err.Tag()
			switch tag {
			case "required":
				validationErrors = append(validationErrors, fmt.Sprintf("%s is required", field))
			case "oneof":
				validationErrors = append(validationErrors, fmt.Sprintf("%s must be one of the allowed values", field))
			case "datetime":
				validationErrors = append(validationErrors, fmt.Sprintf("%s must follow format YYYY-MM-DD", field))
			default:
				validationErrors = append(validationErrors, fmt.Sprintf("%s is invalid", field))
			}
		}
		return errx.NewBadRequestError(strings.Join(validationErrors, ", "))
	}

	// === File Upload (optional) ===
	var proofPath string
	file, err := c.FormFile("proofFile")
	if err == nil && file != nil {
		uploadDir := "./uploads/proofs"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return errx.NewInternalServerError("Failed to create upload directory")
		}

		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, filePath); err != nil {
			return errx.NewInternalServerError("Failed to save file")
		}

		proofPath = filePath
	}

	// Panggil service
	result, err := h.service.CreateTransaction(c.Context(), req, userID, proofPath)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.SuccessResponse("Transaction created successfully", result))
}

// GetTransactionByID godoc
// @Summary Get transaction by ID
// @Description Get a transaction by its ID for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} response.Response{data=entity.Transaction}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions/{id} [get]
func (h *TransactionHandler) GetTransactionByID(c *fiber.Ctx) error {

	userID, ok := c.Locals("userID").(uuid.UUID)
	fmt.Println("DEBUG: Retrieved userID from context:", userID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}
	
	idParam := c.Params("id")
	if strings.TrimSpace(idParam) == "" {
		return errx.NewBadRequestError("Transaction ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return errx.NewBadRequestError("Invalid transaction ID format")
	}

	result, err := h.service.GetTransactionByID(c.Context(), id)
	if err != nil {
		return err
	}

	// if result.UserID != userID {
	// 	return errx.NewUnauthorizedError("You do not have access to this transaction")
	// }


	return c.JSON(response.SuccessResponse("Transaction retrieved successfully", result))
}

// UpdateTransaction godoc
// @Summary Update a transaction
// @Description Update a transaction by its ID for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param request body dto.UpdateTransactionRequest true "Update transaction request"
// @Success 200 {object} response.Response{data=entity.Transaction}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions/{id} [put]
func (h *TransactionHandler) UpdateTransaction(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	idParam := c.Params("id")
	if strings.TrimSpace(idParam) == "" {
		return errx.NewBadRequestError("Transaction ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return errx.NewBadRequestError("Invalid transaction ID format")
	}

	// Parse form-data (karena kita pakai file upload)
	var req dto.UpdateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return errx.NewBadRequestError("Invalid form data")
	}

	if err := h.validate.Struct(req); err != nil {
		return errx.NewBadRequestError(err.Error())
	}

	existingTx, err := h.service.GetTransactionByID(c.Context(), id)
	if err != nil {
		return err
	}

	if existingTx.UserID != userID {
		return errx.NewUnauthorizedError("You do not have access to this transaction")
	}

	// === File Upload Handling ===
	file, err := c.FormFile("proofFile")
	var proofPath string

	if err == nil && file != nil {
		uploadDir := "./uploads/proofs"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return errx.NewInternalServerError("Failed to create upload directory")
		}

		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		// Simpan file baru
		if err := c.SaveFile(file, filePath); err != nil {
			return errx.NewInternalServerError("Failed to save proof file")
		}

		proofPath = filePath

		// Hapus file lama (jika ada)
		if existingTx.ProofFile != "" {
			oldFile := filepath.Join("uploads", "proofs", filepath.Base(existingTx.ProofFile))
			if _, err := os.Stat(oldFile); err == nil {
				os.Remove(oldFile)
			}
		}
	} else {
		// Jika tidak ada file baru, tetap pakai yang lama
		proofPath = existingTx.ProofFile
	}

	// Update transaction di service
	result, err := h.service.UpdateTransaction(c.Context(), id, req, proofPath)
	if err != nil {
		return err
	}

	return c.JSON(response.SuccessResponse("Transaction updated successfully", result))
}

// DeleteTransaction godoc
// @Summary Delete a transaction
// @Description Delete a transaction by its ID for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	idParam := c.Params("id")
	if strings.TrimSpace(idParam) == "" {
		return errx.NewBadRequestError("Transaction ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return errx.NewBadRequestError("Invalid transaction ID format")
	}

	existingTx, err := h.service.GetTransactionByID(c.Context(), id)
	if err != nil {
		return err
	}

	if existingTx.UserID != userID {
		return errx.NewUnauthorizedError("You do not have access to this transaction")
	}

	if err := h.service.DeleteTransaction(c.Context(), id); err != nil {
		return err
	}

	return c.JSON(response.SuccessResponse("Transaction deleted successfully", nil))
}

// GetTransactionsWithPagination godoc
// @Summary Get transactions with pagination
// @Description Get a paginated list of transactions for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Param sort_by query string false "Field to sort by" Enums(date, amount, created_at) default(date)
// @Param order query string false "Sort order" Enums(asc, desc) default(desc)
// @Success 200 {object} response.Response{data=dto.PaginatedTransactionsResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions [get]
func (h *TransactionHandler) GetTransactionsWithPagination(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	var params dto.TransactionListParams
	if err := c.QueryParser(&params); err != nil {
		return errx.NewBadRequestError("Invalid query parameters")
	}

	if params.Page == 0 {
		params.Page = 1
	}
	if params.Limit == 0 {
		params.Limit = 10
	}
	if params.SortBy == "" {
		params.SortBy = "date"
	}
	if params.OrderBy == "" {
		params.OrderBy = "desc"
	}

	if err := h.validate.Struct(params); err != nil {
		return errx.NewBadRequestError(err.Error())
	}

	result, err := h.service.GetTransactionsWithPagination(c.Context(), userID, params)
	if err != nil {
		return err
	}

	return c.JSON(response.SuccessResponse("Transactions retrieved successfully", result))
}

func (h *TransactionHandler) GetProofFile(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	fmt.Println("DEBUG: Retrieved userID from context:", userID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return errx.NewBadRequestError("Invalid transaction ID")
	}

	tx, err := h.service.GetTransactionByID(c.Context(), id)
	if err != nil {
		return err
	}
	if tx == nil {
		return errx.NewNotFoundError("Transaction not found")
	}

	// if tx.UserID != userID {
	// 	return errx.NewUnauthorizedError("You do not have access to this transaction")
	// }

	if tx.ProofFile == "" {
		return errx.NewNotFoundError("No proof file")
	}

	safePath := filepath.Join("uploads", "proofs", filepath.Base(tx.ProofFile))

	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		return errx.NewNotFoundError("File not found")
	}

	return c.SendFile(safePath, true)
}

// GetSummaryTransaction godoc
// @Summary Get summary of transactions
// @Description Get a summary of total income and expenses for the authenticated user
// @Tags transactions
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=dto.SummaryTransactionResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Security BearerAuth
// @Router /transactions/summary [get]
func (h *TransactionHandler) GetSummaryTransaction(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return errx.NewUnauthorizedError("Invalid user ID")
	}

	result, err := h.service.GetSummaryTransaction(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(response.SuccessResponse("Transaction summary retrieved successfully", result))
}
