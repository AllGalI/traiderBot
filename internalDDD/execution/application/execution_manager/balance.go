package em

import (
	"fmt"

	executionDomain "traiderBot/internalDDD/execution/domain"

	"github.com/shopspring/decimal"
)

func (m *ExecutionManager) reserveStartBalance(
	asset string,
	requiredAmount decimal.Decimal,
) (decimal.Decimal, error) {
	availableAmount := m.balanceService.Available(asset)

	if availableAmount.LessThan(requiredAmount) {
		requiredAmount = availableAmount
	}

	if reserveErr := m.balanceService.Reserve(
		asset,
		requiredAmount,
	); reserveErr != nil {
		availableAmount = m.balanceService.Available(asset)

		return decimal.Zero, fmt.Errorf(
			"%w: asset=%s required=%s available=%s: %v",
			ErrInsufficientBalance,
			asset,
			requiredAmount,
			availableAmount,
			reserveErr,
		)
	}

	return requiredAmount, nil
}

func (m *ExecutionManager) releaseStartBalance(
	execution *executionDomain.Execution,
	asset string,
	amount decimal.Decimal,
) error {
	if err := m.balanceService.Release(
		asset,
		amount,
	); err != nil {
		fmt.Printf(
			"[BALANCE-RELEASE-FAILED] execution=%s asset=%s amount=%s error=%v\n",
			execution.ID(),
			asset,
			amount,
			err,
		)

		return err
	}

	return nil
}
