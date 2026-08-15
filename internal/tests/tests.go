package tests

import (
	"fmt"
	"time"
	"traiderBot/internal/arbitrage"
)

func SpeedTest(name string, fn func()) {
	start := time.Now()
	fn()
	fmt.Printf("%s выполнена за %v\n", name, time.Since(start))
}

func TestLiquidityFailsAdd() {
	liquidityFails := arbitrage.NewLiquidityFails(3)

	for i := 0; i < 5; i++ {
		liquidityFails.Add(arbitrage.LiquidityFailData{
			Symbol:      "BTCUSDT",
			Time:        time.Now(),
			IsBuy:       true,
			AmountIn:    float64(i + 1),
			Filled:      float64(i),
			Remaining:   1,
			LevelsCount: i + 1,
			BestPrice:   100,
			WorstPrice:  90,
			Reason:      arbitrage.ReasonInsufficientDepth,
		})
	}

}
