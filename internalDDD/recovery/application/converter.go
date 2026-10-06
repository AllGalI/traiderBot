package application

import (
	"context"
	"fmt"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"

	"github.com/shopspring/decimal"
)

type ConversionPreview struct {
	TargetAsset string
	State       recoveryDomain.State
	Plans       []recoveryDomain.RecoveryPlan
}

type Converter struct {
	planner  Planner
	recovery RecoveryService
}

func NewConverter(
	planner Planner,
	recovery RecoveryService,
) Converter {
	return Converter{
		planner:  planner,
		recovery: recovery,
	}
}

func (c Converter) Preview(
	ctx context.Context,
	state recoveryDomain.State,
	targetAsset string,
	marketContext MarketContext,
) (ConversionPreview, error) {
	if err := ctx.Err(); err != nil {
		return ConversionPreview{}, err
	}

	if targetAsset == "" {
		return ConversionPreview{}, fmt.Errorf(
			"conversion target asset is required",
		)
	}

	actualState := CloneState(state)

	plans := c.planner.CreatePlan(
		recoveryDomain.RecoveryInput{
			ActualState: actualState,
		},
		targetAsset,
		marketContext,
	)

	return ConversionPreview{
		TargetAsset: targetAsset,
		State:       actualState,
		Plans:       plans,
	}, nil
}

func (c Converter) Execute(
	ctx context.Context,
	state recoveryDomain.State,
	targetAsset string,
	marketContext MarketContext,
) (recoveryDomain.State, error) {
	if targetAsset == "" {
		return nil, fmt.Errorf(
			"conversion target asset is required",
		)
	}

	input := recoveryDomain.NewRecoveryInput(
		"",
		CloneState(state),
		CloneState(state),
	)

	return c.recovery.Recover(
		ctx,
		input,
		targetAsset,
		marketContext,
	)
}

type BalanceAmount struct {
	Asset  string
	Amount decimal.Decimal
}

func BuildStateFromBalances(
	balances []BalanceAmount,
) recoveryDomain.State {
	state := make(recoveryDomain.State)

	for _, balance := range balances {
		if balance.Amount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		state[balance.Asset] = balance.Amount
	}

	return state
}