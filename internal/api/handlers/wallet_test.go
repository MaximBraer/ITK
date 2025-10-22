package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"ITK/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type WalletHandlersSuite struct {
	suite.Suite

	ctrl          *gomock.Controller
	walletService *MockWalletService
	handler       *Handler
	logger        *slog.Logger
	ctx           context.Context
}

func TestWalletHandlers(t *testing.T) {
	suite.Run(t, &WalletHandlersSuite{})
}

func (s *WalletHandlersSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.walletService = NewMockWalletService(s.ctrl)

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	s.ctx = context.Background()

	s.handler = New(s.walletService, s.logger)
}

func (s *WalletHandlersSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *WalletHandlersSuite) TestCreate_Success() {
	walletID := uuid.New()

	s.walletService.EXPECT().
		CreateWallet(gomock.Any()).
		Return(walletID, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/create", nil)
	w := httptest.NewRecorder()

	s.handler.Create(w, req)

	s.Equal(http.StatusCreated, w.Code)

	var response CreateWalletResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(walletID.String(), response.WalletID)
}

func (s *WalletHandlersSuite) TestCreate_ServiceError() {
	s.walletService.EXPECT().
		CreateWallet(gomock.Any()).
		Return(uuid.Nil, service.ErrInvalidAmount)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/create", nil)
	w := httptest.NewRecorder()

	s.handler.Create(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)
}

func (s *WalletHandlersSuite) TestOperation_DepositSuccess() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000.50)

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "DEPOSIT",
		Amount:        1000.50,
	}
	body, _ := json.Marshal(operationReq)

	s.walletService.EXPECT().
		Deposit(gomock.Any(), walletID, amount).
		Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("success", response.Status)
}

func (s *WalletHandlersSuite) TestOperation_WithdrawSuccess() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(500.25)

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "WITHDRAW",
		Amount:        500.25,
	}
	body, _ := json.Marshal(operationReq)

	s.walletService.EXPECT().
		Withdraw(gomock.Any(), walletID, amount).
		Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("success", response.Status)
}

func (s *WalletHandlersSuite) TestOperation_InvalidWalletID() {
	operationReq := OperationRequest{
		WalletID:      "invalid-uuid",
		OperationType: "DEPOSIT",
		Amount:        1000,
	}
	body, _ := json.Marshal(operationReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *WalletHandlersSuite) TestOperation_InvalidOperationType() {
	walletID := uuid.New()

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "INVALID",
		Amount:        1000,
	}
	body, _ := json.Marshal(operationReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *WalletHandlersSuite) TestOperation_NegativeAmount() {
	walletID := uuid.New()

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "DEPOSIT",
		Amount:        -100,
	}
	body, _ := json.Marshal(operationReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *WalletHandlersSuite) TestOperation_WalletNotFound() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000)

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "DEPOSIT",
		Amount:        1000,
	}
	body, _ := json.Marshal(operationReq)

	s.walletService.EXPECT().
		Deposit(gomock.Any(), walletID, amount).
		Return(service.ErrWalletNotFound)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *WalletHandlersSuite) TestOperation_InsufficientFunds() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000)

	operationReq := OperationRequest{
		WalletID:      walletID.String(),
		OperationType: "WITHDRAW",
		Amount:        1000,
	}
	body, _ := json.Marshal(operationReq)

	s.walletService.EXPECT().
		Withdraw(gomock.Any(), walletID, amount).
		Return(service.ErrInsufficientFunds)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.Operation(w, req)

	s.Equal(http.StatusConflict, w.Code)
}

func (s *WalletHandlersSuite) TestGetBalance_Success() {
	walletID := uuid.New()
	balance := &service.WalletBalance{
		WalletID: walletID,
		Balance:  decimal.NewFromFloat(5000.50),
	}

	s.walletService.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(balance, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", walletID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	s.handler.GetBalance(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(walletID.String(), response.WalletID)
	s.Equal(5000.50, response.Balance)
}

func (s *WalletHandlersSuite) TestGetBalance_InvalidWalletID() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/invalid-uuid", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	s.handler.GetBalance(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *WalletHandlersSuite) TestGetBalance_WalletNotFound() {
	walletID := uuid.New()

	s.walletService.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(nil, service.ErrWalletNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", walletID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	s.handler.GetBalance(w, req)

	s.Equal(http.StatusNotFound, w.Code)
}
