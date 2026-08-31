# execution — domain

Source: `internalDDD/execution/domain/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/execution/domain/execution.go

Package: `domain`

### Structs

```go
Execution struct {
	ID string

	Opportunity simulationDomain.Opportunity

	Status ExecutionStatus

	CurrentStep int

	Orders []Order

	RequestedStartAmount decimal.Decimal
	ExecutedStartAmount  decimal.Decimal
	FinalAmount          decimal.Decimal
	CurrentAmount        decimal.Decimal

	StartedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt time.Time
}
```

### Constructors

```go
func NewExecution(
	id string,
	opportunity simulationDomain.Opportunity,
) *Execution {
	return &Execution{
		ID:          id,
		Opportunity: opportunity,

		Status:      ExecutionCreated,
		CurrentStep: 0,

		Orders: make([]Order, 0),

		RequestedStartAmount: opportunity.RequestedStartAmount,
		ExecutedStartAmount:  opportunity.ExecutedStartAmount,
		CurrentAmount:        opportunity.RequestedStartAmount,
	}
}
```

### Methods

```go
func (e *Execution) Abort() error {
	if e.IsFinished() {
		return nil
	}

	e.Status = ExecutionAborted
	e.FinishedAt = time.Now()
	e.UpdatedAt = e.FinishedAt

	return nil
}
```

```go
func (e *Execution) AddOrder(order Order) error {
	if e.IsFinished() {
		return fmt.Errorf(
			"cannot add order to finished execution",
		)
	}

	e.Orders = append(e.Orders, order)
	e.UpdatedAt = time.Now()

	return nil
}
```

```go
func (e *Execution) AdvanceStep(
	amountOut decimal.Decimal,
) error {
	if e.Status != ExecutionExecuting {
		return fmt.Errorf(
			"execution is not running",
		)
	}

	e.CurrentAmount = amountOut
	e.CurrentStep++
	e.UpdatedAt = time.Now()

	return nil
}
```

```go
func (e *Execution) Complete(
	finalAmount decimal.Decimal,
) error {
	if e.Status != ExecutionExecuting {
		return fmt.Errorf(
			"execution cannot complete from status %s",
			e.Status,
		)
	}

	e.FinalAmount = finalAmount
	e.CurrentAmount = finalAmount
	e.Status = ExecutionCompleted
	e.FinishedAt = time.Now()
	e.UpdatedAt = e.FinishedAt

	return nil
}
```

```go
func (e *Execution) CurrentOrder() *Order {
	if len(e.Orders) == 0 {
		return nil
	}

	return &e.Orders[len(e.Orders)-1]
}
```

```go
func (e *Execution) Fail() error {
	if e.IsFinished() {
		return nil
	}

	e.Status = ExecutionFailed
	e.FinishedAt = time.Now()
	e.UpdatedAt = e.FinishedAt

	return nil
}
```

```go
func (e *Execution) IsFinished() bool {
	switch e.Status {
	case ExecutionCompleted,
		ExecutionAborted,
		ExecutionFailed:
		return true

	default:
		return false
	}
}
```

```go
func (e *Execution) Start() error {
	if e.Status != ExecutionCreated {
		return fmt.Errorf(
			"execution cannot start from status %s",
			e.Status,
		)
	}

	e.Status = ExecutionExecuting
	e.StartedAt = time.Now()
	e.UpdatedAt = e.StartedAt

	return nil
}
```

## internalDDD/execution/domain/execution_action.go

Package: `domain`

### Methods

```go
func (a ExecutionAction) String() string {
	switch a {
	case ActionContinue:
		return "continue"

	case ActionAbort:
		return "abort"

	case ActionCancel:
		return "cancel"

	default:
		return "unknown"
	}
}
```

## internalDDD/execution/domain/execution_policy.go

Package: `domain`

### Interfaces

```go
ExecutionPolicy interface {
	Decide(execution *Execution) ExecutionDecision
}
```

### Structs

```go
ExecutionDecision struct {
	Type   DecisionType
	Reason string
}
```

## internalDDD/execution/domain/execution_state.go

Package: `domain`

### Methods

```go
func (s ExecutionStatus) String() string {
	switch s {
	case ExecutionCreated:
		return "created"

	case ExecutionExecuting:
		return "executing"

	case ExecutionCompleted:
		return "completed"

	case ExecutionAborted:
		return "aborted"

	case ExecutionFailed:
		return "failed"

	default:
		return "unknown"
	}
}
```

## internalDDD/execution/domain/order.go

Package: `domain`

### Structs

```go
Order struct {
	StepIndex int

	Symbol    string
	Direction marketDomain.TradeDirection

	ClientOrderID   string
	ExchangeOrderID string

	RequestedAmount   decimal.Decimal
	ExpectedAmountOut decimal.Decimal
	ExecutedAmount    decimal.Decimal
	ReceivedAmount    decimal.Decimal

	Fee      decimal.Decimal
	FeeAsset string

	Status OrderStatus

	CreatedAt   time.Time
	SubmittedAt time.Time
	FilledAt    time.Time
}
```

### Constructors

```go
func NewOrder(
	stepIndex int,
	result OrderExecutionResult,
) Order {
	return Order{
		StepIndex: stepIndex,

		Symbol:    result.Symbol,
		Direction: result.Direction,

		ClientOrderID:   result.ClientOrderID,
		ExchangeOrderID: result.ExchangeOrderID,

		RequestedAmount:   result.RequestedAmount,
		ExpectedAmountOut: result.ExpectedAmountOut,
		ExecutedAmount:    result.ExecutedAmount,
		ReceivedAmount:    result.ReceivedAmount,

		Fee:      result.Fee,
		FeeAsset: result.FeeAsset,

		Status: result.Status,

		CreatedAt:   result.CreatedAt,
		SubmittedAt: result.SubmittedAt,
		FilledAt:    result.FilledAt,
	}
}
```

### Methods

```go
func (s OrderStatus) String() string {
	switch s {
	case OrderPending:
		return "pending"

	case OrderSubmitted:
		return "submitted"

	case OrderPartiallyFilled:
		return "partially_filled"

	case OrderFilled:
		return "filled"

	case OrderCancelled:
		return "cancelled"

	case OrderFailed:
		return "failed"

	default:
		return "unknown"
	}
}
```

## internalDDD/execution/domain/order_execution_result.go

Package: `domain`

### Structs

```go
OrderExecutionResult struct {
	StepIndex int

	Symbol    string
	Direction marketDomain.TradeDirection

	ClientOrderID   string
	ExchangeOrderID string

	RequestedAmount   decimal.Decimal
	ExpectedAmountOut decimal.Decimal
	ExecutedAmount    decimal.Decimal
	ReceivedAmount    decimal.Decimal

	Fee      decimal.Decimal
	FeeAsset string

	Status OrderStatus

	CreatedAt   time.Time
	SubmittedAt time.Time
	FilledAt    time.Time
}
```

```go
OrderRequest struct {
	StepIndex int

	Edge marketDomain.GraphEdge

	Amount            decimal.Decimal
	ExpectedAmountOut decimal.Decimal

	ClientOrderID string
}
```

### Constructors

```go
func NewOrderExecutionResult(
	request OrderRequest,
	executedAmount decimal.Decimal,
	receivedAmount decimal.Decimal,
	status OrderStatus,
	clientOrderID string,
	exchangeOrderID string,
	createdAt time.Time,
	submittedAt time.Time,
	filledAt time.Time,
) OrderExecutionResult {
	return OrderExecutionResult{
		StepIndex: request.StepIndex,

		Symbol:    request.Edge.Symbol,
		Direction: request.Edge.Direction,

		ClientOrderID:   clientOrderID,
		ExchangeOrderID: exchangeOrderID,

		RequestedAmount:   request.Amount,
		ExpectedAmountOut: request.ExpectedAmountOut,
		ExecutedAmount:    executedAmount,
		ReceivedAmount:    receivedAmount,

		Status: status,

		CreatedAt:   createdAt,
		SubmittedAt: submittedAt,
		FilledAt:    filledAt,
	}
}
```

