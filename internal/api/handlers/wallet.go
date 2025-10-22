//go:generate go run go.uber.org/mock/mockgen@latest -destination=wallet_mock.go -source=wallet.go -package=handlers

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"ITK/internal/service"
	"ITK/pkg/api/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletService interface {
	CreateWallet(ctx context.Context) (uuid.UUID, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (*service.WalletBalance, error)
	Deposit(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error
	Withdraw(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error
}

type CreateWalletResponse struct {
	WalletID string `json:"walletId" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type OperationRequest struct {
	WalletID      string  `json:"walletId" example:"550e8400-e29b-41d4-a716-446655440000"`
	OperationType string  `json:"operationType" example:"DEPOSIT" enums:"DEPOSIT,WITHDRAW"`
	Amount        float64 `json:"amount" example:"1000.50"`
}

type BalanceResponse struct {
	WalletID string  `json:"walletId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Balance  float64 `json:"balance" example:"5000.50"`
}

type SuccessResponse struct {
	Status string `json:"status" example:"success"`
}

type Handler struct {
	service WalletService
	log     *slog.Logger
}

func New(service WalletService, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log.With(slog.String("component", "handlers/wallet")),
	}
}

// Create godoc
// @Summary Create new wallet
// @Description Creates a new wallet with zero balance and returns its UUID
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 201 {object} CreateWalletResponse
// @Failure 500 {object} response.Response
// @Router /api/v1/wallet/create [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	walletID, err := h.service.CreateWallet(ctx)
	if err != nil {
		h.log.Error("failed to create wallet", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusInternalServerError, "failed to create wallet")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateWalletResponse{
		WalletID: walletID.String(),
	})
}

// Operation godoc
// @Summary Execute wallet operation
// @Description Executes a deposit or withdrawal operation on a wallet
// @Tags Wallet
// @Accept json
// @Produce json
// @Param request body OperationRequest true "Operation details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} response.Response "Invalid request or insufficient funds"
// @Failure 404 {object} response.Response "Wallet not found"
// @Failure 500 {object} response.Response
// @Router /api/v1/wallet [post]
func (h *Handler) Operation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req OperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate wallet ID
	walletID, err := uuid.Parse(req.WalletID)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid wallet ID format")
		return
	}

	// Validate operation type
	if req.OperationType != "DEPOSIT" && req.OperationType != "WITHDRAW" {
		response.WriteError(w, http.StatusBadRequest, "operation type must be DEPOSIT or WITHDRAW")
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		response.WriteError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	amount := decimal.NewFromFloat(req.Amount)

	// Execute operation
	var opErr error
	if req.OperationType == "DEPOSIT" {
		opErr = h.service.Deposit(ctx, walletID, amount)
	} else {
		opErr = h.service.Withdraw(ctx, walletID, amount)
	}

	if opErr != nil {
		if errors.Is(opErr, service.ErrWalletNotFound) {
			response.WriteError(w, http.StatusNotFound, "wallet not found")
			return
		}
		if errors.Is(opErr, service.ErrInsufficientFunds) {
			response.WriteError(w, http.StatusConflict, "insufficient funds")
			return
		}
		if errors.Is(opErr, service.ErrInvalidAmount) {
			response.WriteError(w, http.StatusBadRequest, "amount must be positive")
			return
		}
		h.log.Error("failed to execute operation",
			slog.String("error", opErr.Error()),
			slog.String("wallet_id", walletID.String()),
			slog.String("operation", req.OperationType),
		)
		response.WriteError(w, http.StatusInternalServerError, "failed to execute operation")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{Status: "success"})
}

// GetBalance godoc
// @Summary Get wallet balance
// @Description Returns the current balance of a wallet
// @Tags Wallet
// @Produce json
// @Param id path string true "Wallet UUID"
// @Success 200 {object} BalanceResponse
// @Failure 400 {object} response.Response "Invalid wallet ID"
// @Failure 404 {object} response.Response "Wallet not found"
// @Failure 500 {object} response.Response
// @Router /api/v1/wallets/{id} [get]
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	walletIDStr := chi.URLParam(r, "id")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid wallet ID format")
		return
	}

	balance, err := h.service.GetBalance(ctx, walletID)
	if err != nil {
		if errors.Is(err, service.ErrWalletNotFound) {
			response.WriteError(w, http.StatusNotFound, "wallet not found")
			return
		}
		h.log.Error("failed to get balance",
			slog.String("error", err.Error()),
			slog.String("wallet_id", walletID.String()),
		)
		response.WriteError(w, http.StatusInternalServerError, "failed to get balance")
		return
	}

	balanceFloat, _ := balance.Balance.Float64()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(BalanceResponse{
		WalletID: balance.WalletID.String(),
		Balance:  balanceFloat,
	})
}
