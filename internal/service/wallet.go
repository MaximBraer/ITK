package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"ITK/internal/repository"
	pkgsync "ITK/pkg/sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrWalletNotFound    = repository.ErrWalletNotFound
	ErrInsufficientFunds = repository.ErrInsufficientFunds
)

//go:generate go run go.uber.org/mock/mockgen@latest -destination=wallet_mock.go -source=wallet.go -package=service
type Service interface {
	CreateWallet(ctx context.Context) (uuid.UUID, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (*WalletBalance, error)
	Deposit(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error
	Withdraw(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error
}

type WalletBalance struct {
	WalletID uuid.UUID       `json:"walletId"`
	Balance  decimal.Decimal `json:"balance"`
}

type walletService struct {
	repo       repository.Repository
	log        *slog.Logger
	walletLock *pkgsync.KeyedMutex
}

func New(repo repository.Repository, log *slog.Logger) Service {
	return &walletService{
		repo:       repo,
		log:        log.With(slog.String("component", "service/wallet")),
		walletLock: pkgsync.NewKeyedMutex(),
	}
}

func (s *walletService) CreateWallet(ctx context.Context) (uuid.UUID, error) {
	walletID := uuid.New()

	err := s.repo.Create(ctx, walletID)
	if err != nil {
		s.log.Error("failed to create wallet", slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	s.log.Debug("wallet created", slog.String("wallet_id", walletID.String()))
	return walletID, nil
}

func (s *walletService) GetBalance(ctx context.Context, walletID uuid.UUID) (*WalletBalance, error) {
	wallet, err := s.repo.GetByID(ctx, walletID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			return nil, ErrWalletNotFound
		}
		s.log.Error("failed to get balance", slog.String("error", err.Error()), slog.String("wallet_id", walletID.String()))
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return &WalletBalance{
		WalletID: wallet.ID,
		Balance:  wallet.Balance,
	}, nil
}

func (s *walletService) Deposit(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	key := walletID.String()
	s.walletLock.Lock(key)
	defer s.walletLock.Unlock(key)

	err := s.repo.ApplyOperation(ctx, walletID, "DEPOSIT", amount)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			return ErrWalletNotFound
		}
		s.log.Error("failed to deposit", slog.String("error", err.Error()), slog.String("wallet_id", walletID.String()))
		return fmt.Errorf("failed to deposit: %w", err)
	}

	return nil
}

func (s *walletService) Withdraw(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	key := walletID.String()
	s.walletLock.Lock(key)
	defer s.walletLock.Unlock(key)

	err := s.repo.ApplyOperation(ctx, walletID, "WITHDRAW", amount)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			return ErrWalletNotFound
		}
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		s.log.Error("failed to withdraw", slog.String("error", err.Error()), slog.String("wallet_id", walletID.String()))
		return fmt.Errorf("failed to withdraw: %w", err)
	}

	return nil
}

