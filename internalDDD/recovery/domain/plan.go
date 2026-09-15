package domain

import (
	"github.com/shopspring/decimal"
)

type RecoveryPlan struct {
	Asset string

	Route RecoveryRoute

	// сколько было до восстановления
	InitialAmount decimal.Decimal

	// сколько ожидаем получить
	SimulatedAmount decimal.Decimal

	Steps []RecoveryStep
}
