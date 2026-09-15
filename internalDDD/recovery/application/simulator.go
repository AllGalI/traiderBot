package application

import (
	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

type Simulator struct {
}

func NewSimulator() Simulator {
	return Simulator{}
}

func (s Simulator) Simulate(
	route recoveryDomain.RecoveryRoute,
	amount decimal.Decimal,
	ctx MarketContext,
) recoveryDomain.SimulationResult {

	currentAmount := amount

	steps := make(
		[]recoveryDomain.RecoveryStep,
		0,
		len(route.Edges),
	)

	for _, edge := range route.Edges {

		_, ok :=
			ctx.Symbols[edge.Symbol]

		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		book, ok :=
			ctx.OrderBooks[edge.Symbol]

		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		fee, ok :=
			ctx.Fees[edge.Symbol]

		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		output, price, feeAmount, ok :=
			s.simulateEdge(
				edge,
				currentAmount,
				book,
				fee,
			)

		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		step := recoveryDomain.RecoveryStep{

			FromAsset: edge.From,

			ToAsset: edge.To,

			Symbol: edge.Symbol,

			Direction: edge.Direction,

			InputAmount: currentAmount,

			SimulatedOutput: output,

			SimulatedPrice: price,

			SimulatedFee: feeAmount,
		}

		steps = append(
			steps,
			step,
		)

		currentAmount = output
	}

	return recoveryDomain.SimulationResult{

		Route: route,

		InputAmount: amount,

		OutputAmount: currentAmount,

		Steps: steps,
	}
}

func (s Simulator) simulateEdge(
	edge marketDomain.GraphEdge,
	amount decimal.Decimal,
	book marketDomain.OrderBook,
	fee marketDomain.FeeRate,
) (
	output decimal.Decimal,
	price decimal.Decimal,
	feeAmount decimal.Decimal,
	ok bool,
) {

	switch edge.Direction {

	case marketDomain.Sell:

		return s.simulateSell(
			amount,
			book,
			fee,
		)

	case marketDomain.Buy:

		return s.simulateBuy(
			amount,
			book,
			fee,
		)
	}

	return decimal.Zero,
		decimal.Zero,
		decimal.Zero,
		false
}

func (s Simulator) simulateBuy(
	amount decimal.Decimal,
	book marketDomain.OrderBook,
	fee marketDomain.FeeRate,
) (
	decimal.Decimal,
	decimal.Decimal,
	decimal.Decimal,
	bool,
) {

	best, ok :=
		book.GetBestAsk()

	if !ok {
		return decimal.Zero,
			decimal.Zero,
			decimal.Zero,
			false
	}

	gross :=
		amount.Div(
			best.Price,
		)

	feeAmount :=
		gross.Mul(
			fee.Taker,
		)

	net :=
		gross.Sub(
			feeAmount,
		)

	return net,
		best.Price,
		feeAmount,
		true
}

func (s Simulator) simulateSell(
	amount decimal.Decimal,
	book marketDomain.OrderBook,
	fee marketDomain.FeeRate,
) (
	decimal.Decimal,
	decimal.Decimal,
	decimal.Decimal,
	bool,
) {

	best, ok :=
		book.GetBestBid()

	if !ok {
		return decimal.Zero,
			decimal.Zero,
			decimal.Zero,
			false
	}

	gross :=
		amount.Mul(
			best.Price,
		)

	feeAmount :=
		gross.Mul(
			fee.Taker,
		)

	net :=
		gross.Sub(
			feeAmount,
		)

	return net,
		best.Price,
		feeAmount,
		true
}
