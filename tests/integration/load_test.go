package integration

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/suite"
)

type LoadTestSuite struct {
	Suite
}

func TestLoadTest(t *testing.T) {
	if !isIntegrationTestsRun() {
		t.Skip()
		return
	}

	suite.Run(t, &LoadTestSuite{})
}

func (s *LoadTestSuite) TestConcurrent1000Requests() {
	walletID := s.createWallet()

	initialDeposit := 100000.0
	depositBody := fmt.Sprintf(`{
		"walletId": "%s",
		"operationType": "DEPOSIT",
		"amount": %.2f
	}`, walletID, initialDeposit)

	_, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(depositBody), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	numGoroutines := 1000
	depositsPerGoroutine := 5
	withdrawsPerGoroutine := 3
	depositAmount := 10.0
	withdrawAmount := 5.0

	expectedBalance := initialDeposit + 
		float64(numGoroutines*depositsPerGoroutine)*depositAmount - 
		float64(numGoroutines*withdrawsPerGoroutine)*withdrawAmount

	var wg sync.WaitGroup
	var successCount, failCount atomic.Int64
	var fivexxCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < depositsPerGoroutine; j++ {
				body := fmt.Sprintf(`{
					"walletId": "%s",
					"operationType": "DEPOSIT",
					"amount": %.2f
				}`, walletID, depositAmount)

				_, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(body), nil)
				if err != nil {
					failCount.Add(1)
					continue
				}

				if resp.StatusCode >= 500 {
					fivexxCount.Add(1)
					failCount.Add(1)
				} else if resp.StatusCode == 200 {
					successCount.Add(1)
				} else {
					failCount.Add(1)
				}
			}

			for j := 0; j < withdrawsPerGoroutine; j++ {
				body := fmt.Sprintf(`{
					"walletId": "%s",
					"operationType": "WITHDRAW",
					"amount": %.2f
				}`, walletID, withdrawAmount)

				_, resp, err := postAPIResponse(mainHost, "/api/v1/wallet", []byte(body), nil)
				if err != nil {
					failCount.Add(1)
					continue
				}

				if resp.StatusCode >= 500 {
					fivexxCount.Add(1)
					failCount.Add(1)
				} else if resp.StatusCode == 200 {
					successCount.Add(1)
				} else {
					failCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	totalOperations := numGoroutines * (depositsPerGoroutine + withdrawsPerGoroutine)
	s.T().Logf("Total operations: %d", totalOperations)
	s.T().Logf("Success: %d", successCount.Load())
	s.T().Logf("Failed: %d", failCount.Load())
	s.T().Logf("5xx errors: %d", fivexxCount.Load())

	s.Equal(int64(0), fivexxCount.Load(), "No 5xx errors should occur")
	s.Equal(int64(totalOperations), successCount.Load(), "All operations should succeed")

	respBody, resp, err := getAPIResponse(mainHost, fmt.Sprintf("/api/v1/wallets/%s", walletID), nil)
	s.NoError(err)
	s.Equal(200, resp.StatusCode)

	var balanceResp struct {
		WalletID string  `json:"walletId"`
		Balance  float64 `json:"balance"`
	}
	err = jsoniter.Unmarshal(respBody, &balanceResp)
	s.NoError(err)

	s.InDelta(expectedBalance, balanceResp.Balance, 0.01, "Final balance should match expected")
}

func (s *LoadTestSuite) createWallet() string {
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

