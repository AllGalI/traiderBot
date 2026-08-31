# execution — application

Source: `internalDDD/execution/application/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/execution/application/default_policy.go

Package: `application`

### Structs

```go
DefaultExecutionPolicy struct {
	minReceivedRatio decimal.Decimal
}
```

### Constructors

```go
func NewDefaultExecutionPolicy(
	minReceivedRatio decimal.Decimal,
) *DefaultExecutionPolicy {
	return &DefaultExecutionPolicy{
		minReceivedRatio: minReceivedRatio,
	}
}
```

### Methods

```go
func (p *DefaultExecutionPolicy) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	if execution == nil {
		return executionDomain.ExecutionDecision{
			Type:   executionDomain.DecisionAbort,
			Reason: "execution is nil",
		}
	}

	order := execution.CurrentOrder()

	if order == nil {
		return executionDomain.ExecutionDecision{
			Type: executionDomain.DecisionContinue,
		}
	}

	if order.ExpectedAmountOut.IsZero() {
		return executionDomain.ExecutionDecision{
			Type:   executionDomain.DecisionAbort,
			Reason: "expected amount out is zero",
		}
	}

	ratio := order.ReceivedAmount.Div(order.ExpectedAmountOut)

	if ratio.LessThan(p.minReceivedRatio) {
		return executionDomain.ExecutionDecision{
			Type: executionDomain.DecisionAbort,
			Reason: fmt.Sprintf(
				"received amount ratio %s is below minimum %s",
				ratio,
				p.minReceivedRatio,
			),
		}
	}

	return executionDomain.ExecutionDecision{
		Type: executionDomain.DecisionContinue,
	}
}
```

## internalDDD/execution/application/exchange_executor.go

Package: `application`

### Interfaces

```go
ExchangeExecutor interface {
	Execute(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderExecutionResult, error)
}
```

```go
ExchangeOrderQuery interface {
	GetOrder(
		ctx context.Context,
		symbol string,
		orderID string,
	) (executionDomain.OrderExecutionResult, error)
}
```

## internalDDD/execution/application/execution_manager.go

Package: `application`

### Structs

```go
ExecutionManager struct {
	executor   ExchangeExecutor
	orderQuery ExchangeOrderQuery
	policy     executionDomain.ExecutionPolicy

	counter uint64
}
```

### Constructors

```go
func NewExecutionManager(
	executor ExchangeExecutor,
	orderQuery ExchangeOrderQuery,
	policy executionDomain.ExecutionPolicy,
) *ExecutionManager {
	return &ExecutionManager{
		executor:   executor,
		orderQuery: orderQuery,
		policy:     policy,
	}
}
```

### Methods

```go
func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity *simulationDomain.Opportunity,
) (*executionDomain.Execution, error) {
	if opportunity == nil {
		return nil, fmt.Errorf("opportunity must not be nil")
	}

	if len(opportunity.Steps) == 0 {
		return nil, fmt.Errorf("opportunity has no steps")
	}

	if m.executor == nil {
		return nil, fmt.Errorf("executor is not configured")
	}

	if m.policy == nil {
		return nil, fmt.Errorf("execution policy is not configured")
	}

	execution := executionDomain.NewExecution(
		m.nextID(),
		*opportunity,
	)

	if err := execution.Start(); err != nil {
		return execution, err
	}

	for stepIndex, step := range opportunity.Steps {
		if err := ctx.Err(); err != nil {
			_ = execution.Abort()
			return execution, err
		}

		decision := m.policy.Decide(execution)

		if decision.Type == executionDomain.DecisionAbort {
			_ = execution.Abort()

			return execution, fmt.Errorf(
				"execution aborted: %s",
				decision.Reason,
			)
		}

		amountIn := execution.CurrentAmount

		if stepIndex == 0 {
			amountIn = step.AmountIn
		}

		clientOrderID := m.nextOrderID(
			execution.ID,
			stepIndex,
		)

		request := executionDomain.OrderRequest{
			StepIndex: stepIndex,

			Edge:              step.Edge,
			Amount:            amountIn,
			ExpectedAmountOut: step.AmountOut,

			ClientOrderID: clientOrderID,
		}

		result, err := m.executor.Execute(
			ctx,
			request,
		)

		if err != nil {
			executorErr := err

			reconciledResult, reconcileErr := m.reconcileOrder(
				ctx,
				request,
			)

			if reconcileErr != nil {
				_ = execution.Fail()

				return execution, fmt.Errorf(
					"execute step %d failed: %w; reconciliation failed: %v",
					stepIndex,
					executorErr,
					reconcileErr,
				)
			}

			result = reconciledResult
		}
		if err := m.handleOrderResult(
			execution,
			result,
		); err != nil {
			return execution, err
		}

		if execution.IsFinished() {
			return execution, nil
		}
	}

	if err := execution.Complete(
		execution.CurrentAmount,
	); err != nil {
		return execution, err
	}

	return execution, nil
}
```

```go
func (m *ExecutionManager) handleOrderResult(
	execution *executionDomain.Execution,
	result executionDomain.OrderExecutionResult,
) error {
	order := executionDomain.NewOrder(
		result.StepIndex,
		result,
	)

	if err := execution.AddOrder(order); err != nil {
		_ = execution.Fail()
		return err
	}

	switch result.Status {
	case executionDomain.OrderFilled:
		if err := execution.AdvanceStep(
			result.ReceivedAmount,
		); err != nil {
			_ = execution.Fail()
			return err
		}

		return nil

	case executionDomain.OrderPartiallyFilled:
		_ = execution.Abort()

		return nil

	case executionDomain.OrderCancelled:
		_ = execution.Abort()

		return nil

	case executionDomain.OrderFailed:
		_ = execution.Abort()

		return nil

	default:
		_ = execution.Abort()

		return nil
	}
}
```

```go
func (m *ExecutionManager) nextID() string {
	id := atomic.AddUint64(&m.counter, 1)

	return fmt.Sprintf(
		"execution-%d",
		id,
	)
}
```

```go
func (m *ExecutionManager) nextOrderID(
	executionID string,
	stepIndex int,
) string {
	id := atomic.AddUint64(&m.counter, 1)

	return fmt.Sprintf(
		"%s-order-%d-%d",
		executionID,
		stepIndex,
		id,
	)
}
```

```go
func (m *ExecutionManager) reconcileOrder(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if m.orderQuery == nil {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"order query is not configured",
		)
	}

	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	return m.orderQuery.GetOrder(
		ctx,
		request.Edge.Symbol,
		request.ClientOrderID,
	)
}
```

## internalDDD/execution/application/execution_repository.go

Package: `application`

### Interfaces

```go
ExecutionRepository interface {
	Save(
		ctx context.Context,
		execution *executionDomain.Execution,
	) error

	Get(
		ctx context.Context,
		id string,
	) (*executionDomain.Execution, error)

	ListActive(
		ctx context.Context,
	) ([]*executionDomain.Execution, error)
}
```

## internalDDD/execution/application/tests/fakes.go

Package: `tests`

### Structs

```go
fakeExchangeExecutor struct {
	amountsIn  []decimal.Decimal
	amountsOut []decimal.Decimal

	results  []decimal.Decimal
	statuses []executionDomain.OrderStatus

	requests []executionDomain.OrderRequest
}
```

```go
fakeExchangeExecutorError struct {
	err error
}
```

```go
fakeExchangeOrderQuery struct {
	result executionDomain.OrderExecutionResult
	err    error

	calls int
}
```

```go
fakeExecutionPolicy struct{}
```

```go
fakeExecutionPolicyAbort struct{}
```

### Methods

```go
func (f *fakeExchangeExecutor) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	callIndex := len(f.amountsOut)

	f.amountsIn = append(
		f.amountsIn,
		request.Amount,
	)

	f.requests = append(
		f.requests,
		request,
	)

	result := f.results[callIndex]

	f.amountsOut = append(
		f.amountsOut,
		result,
	)

	status := executionDomain.OrderFilled

	if len(f.statuses) > callIndex {
		status = f.statuses[callIndex]
	}

	now := time.Now()

	return executionDomain.OrderExecutionResult{
		StepIndex: request.StepIndex,

		Symbol:    request.Edge.Symbol,
		Direction: request.Edge.Direction,

		ClientOrderID: request.ClientOrderID,

		RequestedAmount:   request.Amount,
		ExpectedAmountOut: request.ExpectedAmountOut,
		ExecutedAmount:    request.Amount,
		ReceivedAmount:    result,

		Status: status,

		CreatedAt:   now,
		SubmittedAt: now,
		FilledAt:    now,
	}, nil
}
```

```go
func (f *fakeExchangeExecutorError) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	return executionDomain.OrderExecutionResult{}, f.err
}
```

```go
func (f *fakeExchangeOrderQuery) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderExecutionResult, error) {
	f.calls++

	if f.err != nil {
		return executionDomain.OrderExecutionResult{}, f.err
	}

	return f.result, nil
}
```

```go
func (fakeExecutionPolicy) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	return executionDomain.ExecutionDecision{
		Type: executionDomain.DecisionContinue,
	}
}
```

```go
func (fakeExecutionPolicyAbort) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	return executionDomain.ExecutionDecision{
		Type:   executionDomain.DecisionAbort,
		Reason: "test abort",
	}
}
```

