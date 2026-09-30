package application

import (

	"context"

	balanceDomain "traiderBot/internalDDD/balance/domain"
)


type BalanceProvider interface {

	GetBalances(
		ctx context.Context,
	) (
		balanceDomain.State,
		error,
	)
}