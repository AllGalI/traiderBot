package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	dataDir  = "/home/ali/goProjects/crypto-monitor/data"
	usdtAsset = "USDT"
)

// =========================
// JSON
// =========================

type assetAmount struct {
	Asset    string `json:"Asset"`
	Quantity string `json:"Quantity"`
}

type executionStep struct {
	RequestedInput assetAmount `json:"RequestedInput"`
	ActualInput    assetAmount `json:"ActualInput"`
	GrossOutput    assetAmount `json:"GrossOutput"`
	Fee            assetAmount `json:"Fee"`
	NetOutput      assetAmount `json:"NetOutput"`
}

type executionSnapshot struct {
	ID            string `json:"ID"`
	OpportunityID string `json:"OpportunityID"`

	UpdatedAt time.Time `json:"UpdatedAt"`

	InitialState []assetAmount     `json:"InitialState"`
	ActualState  []assetAmount     `json:"ActualState"`
	Steps        []executionStep   `json:"Steps"`
}

type opportunityEdge struct {
	From      string `json:"From"`
	To        string `json:"To"`
	Symbol    string `json:"Symbol"`
	Direction int    `json:"Direction"`
}

type opportunityStep struct {
	Edge opportunityEdge `json:"Edge"`
	AvgPrice string       `json:"AvgPrice"`
}

type opportunity struct {
	ID string `json:"ID"`

	Steps []opportunityStep `json:"Steps"`

	ExecutedStartAmount string `json:"ExecutedStartAmount"`
	EndAmount           string `json:"EndAmount"`
	ProfitAbs           string `json:"ProfitAbs"`
	ProfitPct           string `json:"ProfitPct"`

	Timestamp time.Time `json:"Timestamp"`
}

// =========================
// Conversion
// =========================

type conversionEdge struct {
	To   string
	Rate decimal.Decimal
}

// =========================
// Audit
// =========================

type auditResult struct {
	Execution   executionSnapshot
	Opportunity *opportunity

	InitialUSDT decimal.Decimal
	FinalUSDT   decimal.Decimal
	USDTDelta   decimal.Decimal

	InitialPortfolio decimal.Decimal
	FinalPortfolio   decimal.Decimal
	RealizedPnL      decimal.Decimal

	ExpectedPnL decimal.Decimal
	PnLDiff     decimal.Decimal

	ResidualAssets map[string]decimal.Decimal
	ResidualValue  decimal.Decimal

	ValuationComplete bool
}

type assetTotal struct {
	Quantity decimal.Decimal
	ValueUSDT decimal.Decimal
	Count    int
}

type summary struct {
	Executions int
	Matched    int

	Profitable int
	Losing     int
	Zero       int

	USDTUp   int
	USDTDown int

	InitialUSDT decimal.Decimal
	FinalUSDT   decimal.Decimal
	USDTDelta   decimal.Decimal

	InitialPortfolio decimal.Decimal
	FinalPortfolio   decimal.Decimal
	RealizedPnL      decimal.Decimal

	ExpectedPnL decimal.Decimal
	PnLDiff     decimal.Decimal

	ResidualValue decimal.Decimal

	ResidualAssets map[string]*assetTotal

	Best  *auditResult
	Worst *auditResult

	BiggestUSDTDrop *auditResult
	BiggestUSDTGain *auditResult
}

// =========================
// MAIN
// =========================

func main() {
	fmt.Println("==============================================")
	fmt.Println("              TRADE ECONOMIC AUDIT")
	fmt.Println("==============================================")
	fmt.Println()

	executionDir := filepath.Join(dataDir, "executions")
	opportunityDir := filepath.Join(dataDir, "opportunities")

	opportunities, err := loadOpportunities(opportunityDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load opportunities: %v\n", err)
		os.Exit(1)
	}

	executions, err := loadExecutions(executionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load executions: %v\n", err)
		os.Exit(1)
	}

	results := make([]auditResult, 0, len(executions))

	for _, execution := range executions {
		var opp *opportunity

		if value, ok := opportunities[execution.OpportunityID]; ok {
			copy := value
			opp = &copy
		}

		results = append(results, analyzeExecution(execution, opp))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Execution.UpdatedAt.Before(results[j].Execution.UpdatedAt)
	})

	s := buildSummary(results)

	printMainSummary(s)
	printAssetBreakdown(s)
	printEconomics(s)
	printProblems(results, s)
}

// =========================
// LOAD
// =========================

func loadExecutions(dir string) ([]executionSnapshot, error) {
	var result []executionSnapshot

	err := walkJSON(dir, func(data []byte, filename string) error {
		var value executionSnapshot

		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}

		if value.ID != "" {
			result = append(result, value)
		}

		return nil
	})

	return result, err
}

func loadOpportunities(dir string) (map[string]opportunity, error) {
	result := make(map[string]opportunity)

	err := walkJSON(dir, func(data []byte, filename string) error {
		var value opportunity

		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}

		if value.ID != "" {
			result[value.ID] = value
		}

		return nil
	})

	return result, err
}

func walkJSON(
	dir string,
	callback func([]byte, string) error,
) error {
	_, err := os.Stat(dir)

	if err != nil {
		return err
	}

	return filepath.WalkDir(
		dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if !strings.HasSuffix(
				strings.ToLower(d.Name()),
				".json",
			) {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Buffer(
				make([]byte, 64*1024),
				20*1024*1024,
			)

			line := 0

			for scanner.Scan() {
				line++

				data := scanner.Bytes()

				if len(strings.TrimSpace(string(data))) == 0 {
					continue
				}

				copyData := append([]byte(nil), data...)

				if err := callback(
					copyData,
					fmt.Sprintf("%s:%d", path, line),
				); err != nil {
					return err
				}
			}

			return scanner.Err()
		},
	)
}

// =========================
// ANALYSIS
// =========================

func analyzeExecution(
	execution executionSnapshot,
	opp *opportunity,
) auditResult {
	result := auditResult{
		Execution:      execution,
		Opportunity:    opp,
		ResidualAssets: make(map[string]decimal.Decimal),
	}

	result.InitialUSDT = stateAsset(
		execution.InitialState,
		usdtAsset,
	)

	result.FinalUSDT = stateAsset(
		execution.ActualState,
		usdtAsset,
	)

	result.USDTDelta =
		result.FinalUSDT.Sub(result.InitialUSDT)

	// Все остаточные активы после execution.
	for _, amount := range execution.ActualState {
		qty, err := decimal.NewFromString(amount.Quantity)

		if err != nil || qty.IsZero() {
			continue
		}

		if amount.Asset == usdtAsset {
			continue
		}

		result.ResidualAssets[amount.Asset] =
			result.ResidualAssets[amount.Asset].Add(qty)
	}

	// Если opportunity нет — можем считать только USDT.
	if opp == nil {
		result.InitialPortfolio = result.InitialUSDT
		result.FinalPortfolio = result.FinalUSDT
		result.RealizedPnL = result.USDTDelta

		return result
	}

	graph := buildConversionGraph(*opp)

	initialValue, initialOK :=
		valueStateInUSDT(
			execution.InitialState,
			graph,
		)

	finalValue, finalOK :=
		valueStateInUSDT(
			execution.ActualState,
			graph,
		)

	if initialOK {
		result.InitialPortfolio = initialValue
	} else {
		result.InitialPortfolio = result.InitialUSDT
	}

	if finalOK {
		result.FinalPortfolio = finalValue
		result.ValuationComplete = true
	} else {
		result.FinalPortfolio = result.FinalUSDT
	}

	result.RealizedPnL =
		result.FinalPortfolio.Sub(result.InitialPortfolio)

	if value, err :=
		decimal.NewFromString(opp.ProfitAbs); err == nil {
		result.ExpectedPnL = value
	}

	result.PnLDiff =
		result.RealizedPnL.Sub(result.ExpectedPnL)

	result.ResidualValue =
		result.FinalPortfolio.Sub(result.FinalUSDT)

	return result
}

// =========================
// SUMMARY
// =========================

func buildSummary(results []auditResult) summary {
	s := summary{
		ResidualAssets: make(map[string]*assetTotal),
	}

	for i := range results {
		r := &results[i]

		s.Executions++

		if r.Opportunity != nil {
			s.Matched++
		}

		s.InitialUSDT =
			s.InitialUSDT.Add(r.InitialUSDT)

		s.FinalUSDT =
			s.FinalUSDT.Add(r.FinalUSDT)

		s.USDTDelta =
			s.USDTDelta.Add(r.USDTDelta)

		s.InitialPortfolio =
			s.InitialPortfolio.Add(r.InitialPortfolio)

		s.FinalPortfolio =
			s.FinalPortfolio.Add(r.FinalPortfolio)

		s.RealizedPnL =
			s.RealizedPnL.Add(r.RealizedPnL)

		s.ExpectedPnL =
			s.ExpectedPnL.Add(r.ExpectedPnL)

		s.PnLDiff =
			s.PnLDiff.Add(r.PnLDiff)

		s.ResidualValue =
			s.ResidualValue.Add(r.ResidualValue)

		if r.RealizedPnL.IsPositive() {
			s.Profitable++
		} else if r.RealizedPnL.IsNegative() {
			s.Losing++
		} else {
			s.Zero++
		}

		if r.USDTDelta.IsPositive() {
			s.USDTUp++
		} else if r.USDTDelta.IsNegative() {
			s.USDTDown++
		}

		for asset, quantity := range r.ResidualAssets {
			item := s.ResidualAssets[asset]

			if item == nil {
				item = &assetTotal{}
				s.ResidualAssets[asset] = item
			}

			item.Quantity =
				item.Quantity.Add(quantity)

			item.Count++
		}

		if s.Best == nil ||
			r.RealizedPnL.GreaterThan(
				s.Best.RealizedPnL,
			) {
			s.Best = r
		}

		if s.Worst == nil ||
			r.RealizedPnL.LessThan(
				s.Worst.RealizedPnL,
			) {
			s.Worst = r
		}

		if s.BiggestUSDTDrop == nil ||
			r.USDTDelta.LessThan(
				s.BiggestUSDTDrop.USDTDelta,
			) {
			s.BiggestUSDTDrop = r
		}

		if s.BiggestUSDTGain == nil ||
			r.USDTDelta.GreaterThan(
				s.BiggestUSDTGain.USDTDelta,
			) {
			s.BiggestUSDTGain = r
		}
	}

	return s
}

// =========================
// REPORT
// =========================

func printMainSummary(s summary) {
	fmt.Println("==============================================")
	fmt.Println("                  SUMMARY")
	fmt.Println("==============================================")

	fmt.Printf(
		"Executions:                 %d\n",
		s.Executions,
	)

	fmt.Printf(
		"Matched opportunities:      %d\n",
		s.Matched,
	)

	fmt.Println()

	fmt.Printf(
		"Profitable executions:      %d\n",
		s.Profitable,
	)

	fmt.Printf(
		"Losing executions:          %d\n",
		s.Losing,
	)

	fmt.Printf(
		"Zero PnL executions:        %d\n",
		s.Zero,
	)

	fmt.Println()

	fmt.Printf(
		"USDT increased:             %d\n",
		s.USDTUp,
	)

	fmt.Printf(
		"USDT decreased:             %d\n",
		s.USDTDown,
	)

	fmt.Println()

	fmt.Printf(
		"Initial USDT:               %s\n",
		money(s.InitialUSDT),
	)

	fmt.Printf(
		"Final USDT:                 %s\n",
		money(s.FinalUSDT),
	)

	fmt.Printf(
		"USDT delta:                 %s\n",
		signedMoney(s.USDTDelta),
	)

	fmt.Println()

	fmt.Printf(
		"Initial portfolio value:    %s USDT\n",
		money(s.InitialPortfolio),
	)

	fmt.Printf(
		"Final portfolio value:      %s USDT\n",
		money(s.FinalPortfolio),
	)

	fmt.Printf(
		"REALIZED PnL:               %s USDT\n",
		signedMoney(s.RealizedPnL),
	)

	fmt.Println()

	fmt.Printf(
		"Expected PnL:               %s USDT\n",
		signedMoney(s.ExpectedPnL),
	)

	fmt.Printf(
		"Actual - expected:          %s USDT\n",
		signedMoney(s.PnLDiff),
	)

	fmt.Println()

	if s.Executions > 0 {
		avg := s.RealizedPnL.
			Div(decimal.NewFromInt(int64(s.Executions)))

		fmt.Printf(
			"Average PnL/execution:      %s USDT\n",
			signedMoney(avg),
		)
	}

	fmt.Println()
}

// =========================
// ASSETS
// =========================

func printAssetBreakdown(s summary) {
	fmt.Println("==============================================")
	fmt.Println("             RESIDUAL ASSETS")
	fmt.Println("==============================================")

	if len(s.ResidualAssets) == 0 {
		fmt.Println("No residual non-USDT assets.")
		fmt.Println()
		return
	}

	// Для стоимости используем accumulated
	// residual value из executions.
	type row struct {
		asset    string
		quantity decimal.Decimal
		value    decimal.Decimal
		count    int
	}

	var rows []row

	for asset, item := range s.ResidualAssets {
		rows = append(rows, row{
			asset:    asset,
			quantity: item.Quantity,
			count:    item.Count,
		})
	}

	// Пока стоимость будет рассчитана отдельно ниже
	// через повторный проход невозможно без execution context.
	// Поэтому показываем количество и долю execution.
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].asset < rows[j].asset
	})

	for _, r := range rows {
		fmt.Printf(
			"%-10s quantity=%-24s executions=%d\n",
			r.asset,
			r.quantity.String(),
			r.count,
		)
	}

	fmt.Println()

	fmt.Printf(
		"Total residual value:      %s USDT\n",
		money(s.ResidualValue),
	)

	fmt.Println()

	fmt.Println(
		"Это активы, которые остались после циклов",
		"и поэтому не попали обратно полностью в USDT.",
	)

	fmt.Println()
}

// =========================
// ECONOMICS
// =========================

func printEconomics(s summary) {
	fmt.Println("==============================================")
	fmt.Println("                ECONOMICS")
	fmt.Println("==============================================")

	if s.InitialPortfolio.IsZero() {
		fmt.Println("No portfolio data.")
		return
	}

	pnlPct := s.RealizedPnL.
		Div(s.InitialPortfolio).
		Mul(decimal.NewFromInt(100))

	expectedPct := s.ExpectedPnL.
		Div(s.InitialPortfolio).
		Mul(decimal.NewFromInt(100))

	fmt.Printf(
		"Realized return:           %s %%\n",
		pnlPct.Round(6).String(),
	)

	fmt.Printf(
		"Expected return:           %s %%\n",
		expectedPct.Round(6).String(),
	)

	fmt.Println()

	if s.Best != nil {
		fmt.Printf(
			"Best execution:            %s  (%s USDT)\n",
			s.Best.Execution.ID,
			signedMoney(s.Best.RealizedPnL),
		)
	}

	if s.Worst != nil {
		fmt.Printf(
			"Worst execution:           %s  (%s USDT)\n",
			s.Worst.Execution.ID,
			signedMoney(s.Worst.RealizedPnL),
		)
	}

	fmt.Println()

	if s.BiggestUSDTGain != nil {
		fmt.Printf(
			"Biggest USDT gain:         %s  (%s)\n",
			s.BiggestUSDTGain.Execution.ID,
			signedMoney(s.BiggestUSDTGain.USDTDelta),
		)
	}

	if s.BiggestUSDTDrop != nil {
		fmt.Printf(
			"Biggest USDT drop:         %s  (%s)\n",
			s.BiggestUSDTDrop.Execution.ID,
			signedMoney(s.BiggestUSDTDrop.USDTDelta),
		)
	}

	fmt.Println()
}

// =========================
// PROBLEMS
// =========================

func printProblems(
	results []auditResult,
	s summary,
) {
	fmt.Println("==============================================")
	fmt.Println("                  WARNINGS")
	fmt.Println("==============================================")

	usdtDownButPortfolioUp := 0
	realLosses := 0
	mismatch := 0
	missingOpportunity := 0

	for _, r := range results {
		if r.USDTDelta.IsNegative() &&
			r.RealizedPnL.IsPositive() {
			usdtDownButPortfolioUp++
		}

		if r.RealizedPnL.IsNegative() {
			realLosses++
		}

		if !r.PnLDiff.Abs().LessThan(
			decimal.NewFromFloat(0.0000001),
		) {
			mismatch++
		}

		if r.Opportunity == nil {
			missingOpportunity++
		}
	}

	fmt.Printf(
		"USDT down, portfolio up:   %d\n",
		usdtDownButPortfolioUp,
	)

	fmt.Printf(
		"Real portfolio losses:     %d\n",
		realLosses,
	)

	fmt.Printf(
		"PnL mismatches:             %d\n",
		mismatch,
	)

	fmt.Printf(
		"Missing opportunities:      %d\n",
		missingOpportunity,
	)

	fmt.Println()

	if s.USDTDown > 0 &&
		s.ResidualValue.GreaterThan(decimal.Zero) {

		fmt.Println(
			"ВАЖНО: USDT уменьшается,",
			"но часть средств остаётся в других активах.",
		)

		fmt.Printf(
			"Residual assets value: %s USDT\n",
			money(s.ResidualValue),
		)
	}

	fmt.Println()
}

// =========================
// VALUATION
// =========================

func stateAsset(
	state []assetAmount,
	asset string,
) decimal.Decimal {
	for _, item := range state {
		if item.Asset != asset {
			continue
		}

		value, err := decimal.NewFromString(item.Quantity)

		if err != nil {
			return decimal.Zero
		}

		return value
	}

	return decimal.Zero
}

func buildConversionGraph(
	opp opportunity,
) map[string][]conversionEdge {
	graph := make(map[string][]conversionEdge)

	for _, step := range opp.Steps {
		price, err :=
			decimal.NewFromString(step.AvgPrice)

		if err != nil || price.IsZero() {
			continue
		}

		from := step.Edge.From
		to := step.Edge.To

		var fromTo decimal.Decimal

		// BUY:
		// quote -> base = 1 / price
		//
		// SELL:
		// base -> quote = price
		if step.Edge.Direction == 0 {
			fromTo =
				decimal.NewFromInt(1).Div(price)
		} else {
			fromTo = price
		}

		toFrom :=
			decimal.NewFromInt(1).Div(fromTo)

		graph[from] = append(
			graph[from],
			conversionEdge{
				To:   to,
				Rate: fromTo,
			},
		)

		graph[to] = append(
			graph[to],
			conversionEdge{
				To:   from,
				Rate: toFrom,
			},
		)
	}

	return graph
}

func valueStateInUSDT(
	state []assetAmount,
	graph map[string][]conversionEdge,
) (decimal.Decimal, bool) {
	total := decimal.Zero

	for _, item := range state {
		quantity, err :=
			decimal.NewFromString(item.Quantity)

		if err != nil {
			return decimal.Zero, false
		}

		if quantity.IsZero() {
			continue
		}

		rate, ok :=
			conversionRateToUSDT(
				item.Asset,
				graph,
			)

		if !ok {
			return decimal.Zero, false
		}

		total =
			total.Add(quantity.Mul(rate))
	}

	return total, true
}

func conversionRateToUSDT(
	from string,
	graph map[string][]conversionEdge,
) (decimal.Decimal, bool) {
	if from == usdtAsset {
		return decimal.NewFromInt(1), true
	}

	type node struct {
		asset string
		rate  decimal.Decimal
	}

	queue := []node{
		{
			asset: from,
			rate:  decimal.NewFromInt(1),
		},
	}

	visited := map[string]bool{
		from: true,
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range graph[current.asset] {
			if visited[edge.To] {
				continue
			}

			rate :=
				current.rate.Mul(edge.Rate)

			if edge.To == usdtAsset {
				return rate, true
			}

			visited[edge.To] = true

			queue = append(
				queue,
				node{
					asset: edge.To,
					rate:  rate,
				},
			)
		}
	}

	return decimal.Zero, false
}

// =========================
// FORMAT
// =========================

func money(v decimal.Decimal) string {
	return v.Round(12).StringFixed(12)
}

func signedMoney(v decimal.Decimal) string {
	if v.IsNegative() {
		return money(v)
	}

	return "+" + money(v)
}