package tests

import (
	"testing"
	"time"
	"traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestActualStepResult(t *testing.T) {
	completedAt := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	result := domain.ActualStepResult{
		StepID: 2,
		Input: domain.AssetAmount{
			Asset:    "USDT",
			Quantity: decimal.NewFromFloat(100),
		},
		Output: domain.AssetAmount{
			Asset:    "BTC",
			Quantity: decimal.NewFromFloat(0.00179),
		},
		CompletedAt: completedAt,
	}

	require.Equal(t, 2, result.StepID)

	require.Equal(t, "USDT", result.Input.Asset)
	require.True(t, decimal.NewFromFloat(100).Equal(result.Input.Quantity))

	require.Equal(t, "BTC", result.Output.Asset)
	require.True(t, decimal.NewFromFloat(0.00179).Equal(result.Output.Quantity))

	require.Equal(t, completedAt, result.CompletedAt)
}

func TestExecutionStoresActualStepResults(t *testing.T) {
	opportunity := simulationDomain.Opportunity{}

	execution := domain.NewExecution(opportunity)

	result := domain.ActualStepResult{
		StepID: 0,
		Input: domain.AssetAmount{
			Asset:    "USDT",
			Quantity: decimal.NewFromInt(100),
		},
		Output: domain.AssetAmount{
			Asset:    "BTC",
			Quantity: decimal.NewFromFloat(0.0018),
		},
		CompletedAt: time.Now(),
	}

	execution.AddActualStepResult(result)

	require.Len(t, execution.ActualStepResults(), 1)
	require.Equal(t, result, execution.ActualStepResults()[0])
}

func TestNewExecution(t *testing.T) {
	opportunity := simulationDomain.Opportunity{}

	execution := domain.NewExecution(opportunity)

	require.Equal(t, domain.StateNotStarted, execution.State())
	require.Equal(t, opportunity, execution.Opportunity())
	require.Empty(t, execution.ActualStepResults())
}