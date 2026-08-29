package tests

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/market/domain"
	"traiderBot/internalDDD/market/infrastructure"
)

func TestOrderBookStoreConcurrentReadWrite(t *testing.T) {
	store := infrastructure.NewOrderBookStore()

	const (
		writers = 10
		readers = 10
		loops   = 1000
	)

	var wg sync.WaitGroup

	wg.Add(writers + readers)

	for i := 0; i < writers; i++ {
		go func(i int) {
			defer wg.Done()

			for j := 0; j < loops; j++ {
				store.Set(domain.OrderBook{
					Symbol: "BTCUSDT",
					Bids: []domain.PriceLevel{
						{
							Price: decimal.NewFromInt(int64(100 + i)),
							Qty:   decimal.NewFromInt(int64(j + 1)),
						},
					},
				})
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		go func(i int) {
			defer wg.Done()

			for j := 0; j < loops; j++ {
				orderBook, ok := store.Get("BTCUSDT")

				if !ok {
					continue
				}

				if orderBook.Symbol != "BTCUSDT" {
					t.Errorf(
						"reader %d: unexpected symbol %q",
						i,
						orderBook.Symbol,
					)
				}

				if len(orderBook.Bids) > 0 &&
					orderBook.Bids[0].Price.LessThan(decimal.Zero) {
					t.Errorf(
						"reader %d: unexpected negative price %s",
						i,
						orderBook.Bids[0].Price,
					)
				}
			}
		}(i)
	}

	wg.Wait()

	orderBook, ok := store.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected BTCUSDT to exist after concurrent writes")
	}

	if orderBook.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected BTCUSDT, got %s",
			orderBook.Symbol,
		)
	}
}