package integration

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type WalletSuite struct {
	Suite

	logger *slog.Logger
	DB     *sql.DB
}

func TestWallet(t *testing.T) {
	if !isIntegrationTestsRun() {
		t.Skip()
		return
	}

	suite.Run(t, &WalletSuite{})
}

func (s *WalletSuite) SetupTest() {
	s.Suite.SetupTest()

	s.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(s.logger)

	s.initDatabase()
}

func (s *WalletSuite) TearDownTest() {
	s.logger = nil
	if s.DB != nil {
		s.Require().NoError(s.DB.Close())
	}
}

func (s *WalletSuite) initDatabase() {
	dsn := "postgres://postgres:postgres@localhost:5433/wallet?sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	s.Require().NoError(err)

	err = db.Ping()
	s.Require().NoError(err)

	s.DB = db
}

func (s *WalletSuite) clearDatabase() {
	_, err := s.DB.Exec(`TRUNCATE TABLE operations CASCADE`)
	s.NoError(err)
	_, err = s.DB.Exec(`TRUNCATE TABLE wallets CASCADE`)
	s.NoError(err)
}

func (s *WalletSuite) TestCreateWallet() {
	s.clearDatabase()

	walletID := s.createWallet()

	s.Eventually(
		func() bool {
			var id uuid.UUID
			var balance float64
			err := s.DB.QueryRow(
				`SELECT id, balance FROM wallets WHERE id = $1`,
				walletID,
			).Scan(&id, &balance)

			if errors.Is(err, sql.ErrNoRows) {
				return false
			}

			s.NoError(err)
			s.Equal(walletID, id.String())
			s.Equal(0.0, balance)

			return true
		},
		time.Second*5,
		time.Millisecond*100,
	)
}

func (s *WalletSuite) TestGetWallet() {
	s.clearDatabase()

	walletID := s.createWallet()

	respBody, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/wallets/%s", walletID), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var wallet struct {
		WalletID string  `json:"walletId"`
		Balance  float64 `json:"balance"`
	}
	err = jsoniter.Unmarshal(respBody, &wallet)
	s.NoError(err)

	s.Equal(walletID, wallet.WalletID)
	s.Equal(0.0, wallet.Balance)
}

func (s *WalletSuite) TestDeposit() {
	s.clearDatabase()

	walletID := s.createWallet()

	requestBody := fmt.Sprintf(`{
		"walletId": "%s",
		"operationType": "DEPOSIT",
		"amount": 1000.50
	}`, walletID)

	respBody, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(requestBody), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var response struct {
		Status string `json:"status"`
	}

	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)
	s.Equal("success", response.Status)

	var balance float64
	err = s.DB.QueryRow(`SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&balance)
	s.NoError(err)
	s.Equal(1000.50, balance)
}

func (s *WalletSuite) TestWithdraw() {
	s.clearDatabase()

	walletID := s.createWallet()

	depositBody := fmt.Sprintf(`{
		"walletId": "%s",
		"operationType": "DEPOSIT",
		"amount": 1000
	}`, walletID)

	_, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(depositBody), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	withdrawBody := fmt.Sprintf(`{
		"walletId": "%s",
		"operationType": "WITHDRAW",
		"amount": 500
	}`, walletID)

	respBody, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(withdrawBody), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var response struct {
		Status string `json:"status"`
	}
	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)
	s.Equal("success", response.Status)

	var balance float64
	err = s.DB.QueryRow(`SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&balance)
	s.NoError(err)
	s.Equal(500.0, balance)
}

func (s *WalletSuite) TestInsufficientFunds() {
	s.clearDatabase()

	walletID := s.createWallet()

	withdrawBody := fmt.Sprintf(`{
		"walletId": "%s",
		"operationType": "WITHDRAW",
		"amount": 1000
	}`, walletID)

	_, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(withdrawBody), nil)
	s.NoError(err)
	s.Equal(409, resp.StatusCode)
}

func (s *WalletSuite) TestWalletNotFound() {
	s.clearDatabase()

	nonExistentWalletID := uuid.New().String()

	_, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/wallets/%s", nonExistentWalletID), nil)
	s.NoError(err)
	s.Equal(404, resp.StatusCode)
}

func (s *WalletSuite) createWallet() string {
	respBody, resp, err := postAPIResponse(mainHost, "/api/v1/wallet/create", nil, nil)
	s.NoError(err)
	s.Equal(201, resp.StatusCode)

	var response struct {
		WalletID string `json:"walletId"`
	}

	err = jsoniter.Unmarshal(respBody, &response)
	s.NoError(err)
	s.NotEmpty(response.WalletID)

	return response.WalletID
}
