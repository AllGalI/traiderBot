package domain

import (
	"fmt"
	"time"

	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

type Execution struct {
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

func (e *Execution) Abort() error {
	if e.IsFinished() {
		return nil
	}

	e.Status = ExecutionAborted
	e.FinishedAt = time.Now()
	e.UpdatedAt = e.FinishedAt

	return nil
}

func (e *Execution) Fail() error {
	if e.IsFinished() {
		return nil
	}

	e.Status = ExecutionFailed
	e.FinishedAt = time.Now()
	e.UpdatedAt = e.FinishedAt

	return nil
}

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

func (e *Execution) CurrentOrder() *Order {
	if len(e.Orders) == 0 {
		return nil
	}

	return &e.Orders[len(e.Orders)-1]
}

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
