package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrTooManyRetries    = errors.New("too many retries")
)

type Wallet struct {
	ID        uuid.UUID
	Balance   decimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
}

//go:generate go run go.uber.org/mock/mockgen@latest -destination=wallet_mock.go -source=wallet.go -package=repository
type Repository interface {
	Create(ctx context.Context, walletID uuid.UUID) error
	GetByID(ctx context.Context, walletID uuid.UUID) (*Wallet, error)
	ApplyOperation(ctx context.Context, walletID uuid.UUID, opType string, amount decimal.Decimal) error
}

type walletRepo struct {
	pool         *pgxpool.Pool
	log          *slog.Logger
	maxRetries   int
	baseDelayMS  int
}

type Config struct {
	MaxRetries  int
	BaseDelayMS int
}

func New(pool *pgxpool.Pool, log *slog.Logger, cfg Config) Repository {
	return &walletRepo{
		pool:        pool,
		log:         log.With(slog.String("component", "repository/wallet")),
		maxRetries:  cfg.MaxRetries,
		baseDelayMS: cfg.BaseDelayMS,
	}
}

func (r *walletRepo) Create(ctx context.Context, walletID uuid.UUID) error {
	sql, args, err := squirrel.Insert("wallets").
		Columns("id", "balance", "created_at", "updated_at").
		Values(walletID, 0, squirrel.Expr("NOW()"), squirrel.Expr("NOW()")).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("wallet already exists: %w", err)
		}
		r.log.Error("failed to create wallet", slog.String("error", err.Error()), slog.String("wallet_id", walletID.String()))
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	r.log.Info("wallet created", slog.String("wallet_id", walletID.String()))
	return nil
}

func (r *walletRepo) GetByID(ctx context.Context, walletID uuid.UUID) (*Wallet, error) {
	sql, args, err := squirrel.Select("id", "balance", "created_at", "updated_at").
		From("wallets").
		Where(squirrel.Eq{"id": walletID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var w Wallet
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&w.ID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		r.log.Error("failed to get wallet", slog.String("error", err.Error()), slog.String("wallet_id", walletID.String()))
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return &w, nil
}

func (r *walletRepo) ApplyOperation(ctx context.Context, walletID uuid.UUID, opType string, amount decimal.Decimal) error {
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		err := r.executeOperation(ctx, walletID, opType, amount)
		if err == nil {
			return nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40001" {
			backoff := time.Duration(1<<attempt) * time.Duration(r.baseDelayMS) * time.Millisecond
			r.log.Warn("serialization failure, retrying",
				slog.String("wallet_id", walletID.String()),
				slog.Int("attempt", attempt+1),
				slog.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
			continue
		}
		return err
	}
	return ErrTooManyRetries
}

func (r *walletRepo) executeOperation(ctx context.Context, walletID uuid.UUID, opType string, amount decimal.Decimal) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	lockSQL, _, err := squirrel.Select("pg_advisory_xact_lock(hashtext(?))").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build lock SQL: %w", err)
	}
	_, err = tx.Exec(ctx, lockSQL, walletID.String())
	if err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	delta := amount
	if opType == "WITHDRAW" {
		delta = amount.Neg()
	}

	updateSQL, updateArgs, err := squirrel.Update("wallets").
		Set("balance", squirrel.Expr("balance + ?", delta)).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where("id = ? AND (balance + ?) >= 0", walletID, delta).
		Suffix("RETURNING balance").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update SQL: %w", err)
	}

	var newBalance decimal.Decimal
	err = tx.QueryRow(ctx, updateSQL, updateArgs...).Scan(&newBalance)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			checkSQL, _, buildErr := squirrel.Select("EXISTS(SELECT 1 FROM wallets WHERE id = ?)").
				PlaceholderFormat(squirrel.Dollar).
				ToSql()
			if buildErr != nil {
				return fmt.Errorf("failed to build check SQL: %w", buildErr)
			}

			var exists bool
			checkErr := tx.QueryRow(ctx, checkSQL, walletID).Scan(&exists)
			if checkErr != nil {
				return fmt.Errorf("failed to check wallet existence: %w", checkErr)
			}
			if !exists {
				return ErrWalletNotFound
			}
			return ErrInsufficientFunds
		}
		return fmt.Errorf("failed to update balance: %w", err)
	}

	insertSQL, insertArgs, err := squirrel.Insert("operations").
		Columns("wallet_id", "operation_type", "amount", "balance_after").
		Values(walletID, opType, amount, newBalance).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert SQL: %w", err)
	}

	_, err = tx.Exec(ctx, insertSQL, insertArgs...)
	if err != nil {
		return fmt.Errorf("failed to insert operation record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("operation applied",
		slog.String("wallet_id", walletID.String()),
		slog.String("operation_type", opType),
		slog.String("amount", amount.String()),
		slog.String("new_balance", newBalance.String()),
	)

	return nil
}

