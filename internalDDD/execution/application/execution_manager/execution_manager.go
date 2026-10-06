package em

import (
	"context"
	"errors"
	"fmt"

	balanceApplication "traiderBot/internalDDD/balance/application"
	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	executionDomain "traiderBot/internalDDD/execution/domain"
	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"traiderBot/internalDDD/execution/application"
)

var ErrInsufficientBalance = errors.New("insufficient available balance")

type ExecutionManager struct {
	exchangeAdapter  exchangeDomain.Adapter
	resultMapper     executionDomain.ActualStepResultMapper
	evaluator        application.Evaluator
	orderIDGenerator *exchangeDomain.OrderIDGenerator

	balanceService *balanceApplication.BalanceService

	executionStore application.ExecutionStore
}

func NewExecutionManager(
	exchangeAdapter exchangeDomain.Adapter,
	resultMapper executionDomain.ActualStepResultMapper,
	evaluator application.Evaluator,
	orderIDGenerator *exchangeDomain.OrderIDGenerator,
	executionStore application.ExecutionStore,
	balanceService *balanceApplication.BalanceService,
) *ExecutionManager {
	return &ExecutionManager{
		exchangeAdapter:  exchangeAdapter,
		resultMapper:     resultMapper,
		evaluator:        evaluator,
		orderIDGenerator: orderIDGenerator,
		executionStore:   executionStore,
		balanceService:   balanceService,
	}
}

func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity simulationDomain.Opportunity,
	marketContext recoveryApplication.MarketContext,
) (
	execution *executionDomain.Execution,
	actualState recoveryDomain.State,
	err error,
) {
	if err := m.validateManager(opportunity); err != nil {
		return nil, nil, err
	}

	execution = executionDomain.NewExecution(opportunity)

	if err := execution.StartPreparing(); err != nil {
		return execution, nil, err
	}

	steps := m.buildExecutionSteps(opportunity)
	initialState := m.buildInitialState(opportunity)
	actualState = recoveryApplication.CloneState(initialState)

	defer func() {
		if execution == nil {
			return
		}

		if err == nil {
			return
		}

		if execution.IsFinished() {
			return
		}

		_ = m.failExecution(
			execution,
			len(steps),
			initialState,
			actualState,
			err,
		)
	}()

	m.createExecution(
		execution,
		0,
		len(steps),
		initialState,
		actualState,
		nil,
	)

	startAsset := opportunity.Route.Edges[0].From
	requiredAmount := opportunity.ExecutedStartAmount

	if requiredAmount.IsZero() || requiredAmount.IsNegative() {
		return execution, actualState, fmt.Errorf(
			"executed start amount must be greater than zero: %s",
			requiredAmount,
		)
	}

	requiredAmount, err = m.reserveStartBalance(
		startAsset,
		requiredAmount,
	)
	if err != nil {
		return execution, actualState, err
	}

	reserved := true

	defer func() {
		if !reserved {
			return
		}

		if releaseErr := m.releaseStartBalance(
			execution,
			startAsset,
			requiredAmount,
		); releaseErr != nil {
			return
		}

		reserved = false
	}()

	// Important:
	// execution must continue with the actually reserved amount.
	currentAmount := requiredAmount

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return execution, actualState, err
		}

		if err := execution.StartExecutingStep(); err != nil {
			return execution, actualState, err
		}

		m.persistExecution(
			execution,
			step.ID,
			len(steps),
			initialState,
			actualState,
			nil,
		)

		requestAmount := currentAmount

		if step.ID == 0 {
			requestAmount = step.PlannedInput.Quantity
		}

		actualResult, result, err := m.executeStep(ctx,
			step,
			requestAmount,
			marketContext,
		)
		if err != nil {
			return execution, actualState, err
		}

		execution.AddActualStepResult(actualResult)

		actualState = m.applyOrderResult(
			actualState,
			step,
			result,
		)

		currentAmount = actualResult.NetOutput.Quantity

		if step.ID >= len(steps)-1 {
			continue
		}

		if err := m.evaluateStep(
			execution,
			opportunity,
			step,
			actualResult,
		); err != nil {
			return execution, actualState, err
		}

	}

	if err := m.completeExecution(
		execution,
		len(steps),
		initialState,
		actualState,
		nil,
	); err != nil {
		return execution, actualState, err
	}

	return execution, actualState, nil
}

func (m *ExecutionManager) validateManager(
	opportunity simulationDomain.Opportunity,
) error {
	if m.exchangeAdapter == nil {
		return fmt.Errorf("exchange adapter is not configured")
	}

	if m.orderIDGenerator == nil {
		return fmt.Errorf("order ID generator is not configured")
	}

	if m.balanceService == nil {
		return fmt.Errorf("balance service is not configured")
	}

	if len(opportunity.Route.Edges) == 0 {
		return fmt.Errorf("opportunity route has no edges")
	}

	return nil
}

func (m *ExecutionManager) evaluateStep(
	execution *executionDomain.Execution,
	opportunity simulationDomain.Opportunity,
	step executionDomain.ExecutionStep,
	actualResult executionDomain.ActualStepResult,
) error {
	if err := execution.StartEvaluating(); err != nil {
		return err
	}

	decision := m.evaluator.Evaluate(
		opportunity.Steps[step.ID],
		actualResult,
	)

	if err := execution.ApplyDecision(decision); err != nil {
		return err
	}

	if decision == executionDomain.DecisionRecoveryRequired {
		if err := execution.StartEvaluating(); err != nil {
			return err
		}

		// Recovery intentionally disabled for the current
		// simplified execution model.
	}

	return nil
}
