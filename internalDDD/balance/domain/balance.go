package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type AssetBalance struct {
	Asset string

	Total decimal.Decimal

	Reserved decimal.Decimal
}

func NewAssetBalance(
	asset string,
	amount decimal.Decimal,
) AssetBalance {
	return AssetBalance{
		Asset:    asset,
		Total:    amount,
		Reserved: decimal.Zero,
	}
}

func (b AssetBalance) Available() decimal.Decimal {
	return b.Total.Sub(b.Reserved)
}

func (b *AssetBalance) Reserve(
	amount decimal.Decimal,
) error {

	if amount.IsNegative() {
		return fmt.Errorf(
			"reserve amount cannot be negative",
		)
	}

	if b.Available().LessThan(amount) {
		return fmt.Errorf(
			"insufficient available balance %s: requested %s available %s",
			b.Asset,
			amount,
			b.Available(),
		)
	}

	b.Reserved = b.Reserved.Add(amount)

	return nil
}


func (b *AssetBalance) Release(
	amount decimal.Decimal,
) error {

	if amount.IsNegative() {
		return fmt.Errorf(
			"release amount cannot be negative",
		)
	}

	if b.Reserved.LessThan(amount) {
		return fmt.Errorf(
			"cannot release more than reserved",
		)
	}

	b.Reserved = b.Reserved.Sub(amount)

	return nil
}


func (b *AssetBalance) Add(
	amount decimal.Decimal,
) {
	b.Total = b.Total.Add(amount)
}


func (b *AssetBalance) Sub(
	amount decimal.Decimal,
) error {

	if b.Total.LessThan(amount) {
		return fmt.Errorf(
			"insufficient total balance",
		)
	}

	b.Total = b.Total.Sub(amount)

	return nil
}