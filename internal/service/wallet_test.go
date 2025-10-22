package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"ITK/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type WalletServiceSuite struct {
	suite.Suite

	ctrl          *gomock.Controller
	walletRepo    *repository.MockRepository
	walletService *walletService
	logger        *slog.Logger
	ctx           context.Context
}

func TestWalletService(t *testing.T) {
	suite.Run(t, &WalletServiceSuite{})
}

func (s *WalletServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.walletRepo = repository.NewMockRepository(s.ctrl)

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	s.ctx = context.Background()

	s.walletService = &walletService{
		repo: s.walletRepo,
		log:  s.logger,
	}
}

func (s *WalletServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *WalletServiceSuite) TestCreateWallet_Success() {
	s.walletRepo.EXPECT().
		Create(s.ctx, gomock.Any()).
		Return(nil)

	walletID, err := s.walletService.CreateWallet(s.ctx)

	s.NoError(err)
	s.NotEqual(uuid.Nil, walletID)
}

func (s *WalletServiceSuite) TestCreateWallet_RepositoryError() {
	repoError := errors.New("database error")

	s.walletRepo.EXPECT().
		Create(s.ctx, gomock.Any()).
		Return(repoError)

	walletID, err := s.walletService.CreateWallet(s.ctx)

	s.Error(err)
	s.Equal(uuid.Nil, walletID)
	s.Contains(err.Error(), "database error")
}

func (s *WalletServiceSuite) TestGetBalance_Success() {
	walletID := uuid.New()
	balance := decimal.NewFromFloat(500.50)

	wallet := &repository.Wallet{
		ID:      walletID,
		Balance: balance,
	}

	s.walletRepo.EXPECT().
		GetByID(s.ctx, walletID).
		Return(wallet, nil)

	result, err := s.walletService.GetBalance(s.ctx, walletID)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(walletID, result.WalletID)
	s.True(result.Balance.Equal(balance))
}

func (s *WalletServiceSuite) TestGetBalance_WalletNotFound() {
	walletID := uuid.New()

	s.walletRepo.EXPECT().
		GetByID(s.ctx, walletID).
		Return(nil, repository.ErrWalletNotFound)

	result, err := s.walletService.GetBalance(s.ctx, walletID)

	s.Error(err)
	s.Nil(result)
	s.ErrorIs(err, ErrWalletNotFound)
}

func (s *WalletServiceSuite) TestDeposit_Success() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000)

	s.walletRepo.EXPECT().
		ApplyOperation(s.ctx, walletID, "DEPOSIT", amount).
		Return(nil)

	err := s.walletService.Deposit(s.ctx, walletID, amount)

	s.NoError(err)
}

func (s *WalletServiceSuite) TestDeposit_InvalidAmount_Zero() {
	walletID := uuid.New()
	amount := decimal.Zero

	err := s.walletService.Deposit(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrInvalidAmount)
}

func (s *WalletServiceSuite) TestDeposit_InvalidAmount_Negative() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(-100)

	err := s.walletService.Deposit(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrInvalidAmount)
}

func (s *WalletServiceSuite) TestDeposit_WalletNotFound() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000)

	s.walletRepo.EXPECT().
		ApplyOperation(s.ctx, walletID, "DEPOSIT", amount).
		Return(repository.ErrWalletNotFound)

	err := s.walletService.Deposit(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrWalletNotFound)
}

func (s *WalletServiceSuite) TestWithdraw_Success() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(500)

	s.walletRepo.EXPECT().
		ApplyOperation(s.ctx, walletID, "WITHDRAW", amount).
		Return(nil)

	err := s.walletService.Withdraw(s.ctx, walletID, amount)

	s.NoError(err)
}

func (s *WalletServiceSuite) TestWithdraw_InvalidAmount_Zero() {
	walletID := uuid.New()
	amount := decimal.Zero

	err := s.walletService.Withdraw(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrInvalidAmount)
}

func (s *WalletServiceSuite) TestWithdraw_InvalidAmount_Negative() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(-100)

	err := s.walletService.Withdraw(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrInvalidAmount)
}

func (s *WalletServiceSuite) TestWithdraw_InsufficientFunds() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1000)

	s.walletRepo.EXPECT().
		ApplyOperation(s.ctx, walletID, "WITHDRAW", amount).
		Return(repository.ErrInsufficientFunds)

	err := s.walletService.Withdraw(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrInsufficientFunds)
}

func (s *WalletServiceSuite) TestWithdraw_WalletNotFound() {
	walletID := uuid.New()
	amount := decimal.NewFromFloat(500)

	s.walletRepo.EXPECT().
		ApplyOperation(s.ctx, walletID, "WITHDRAW", amount).
		Return(repository.ErrWalletNotFound)

	err := s.walletService.Withdraw(s.ctx, walletID, amount)

	s.Error(err)
	s.ErrorIs(err, ErrWalletNotFound)
}
