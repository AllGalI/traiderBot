package infrastructure


import "context"


type BinanceAccountClient interface {

	GetAccount(
		ctx context.Context,
	) (
	AccountInfo,
	error,
	)
}


type AccountInfo struct {

	Balances []AccountBalance
}


type AccountBalance struct {

	Asset string

	Free string

	Locked string
}