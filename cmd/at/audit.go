package main

// import (
// 	"bufio"
// 	"encoding/csv"
// 	"encoding/json"
// 	"fmt"
// 	"math"
// 	"os"
// 	"path/filepath"
// 	"sort"
// 	"strconv"
// 	"strings"
// )

// type AssetAmount struct {
// 	Asset    string `json:"Asset"`
// 	Quantity string `json:"Quantity"`
// }

// type Edge struct {
// 	From      string `json:"From"`
// 	To        string `json:"To"`
// 	Symbol    string `Symbol"`
// 	Direction int    `json:"Direction"`
// }

// type OpportunityStep struct {
// 	Edge           Edge   `json:"Edge"`
// 	AvgPrice       string `json:"AvgPrice"`
// 	Quantity       string `json:"Quantity"`
// 	QuoteQty       string `json:"QuoteQty"`
// 	RequestedInput string `json:"RequestedInput"`
// 	ExecutedInput  string `json:"ExecutedInput"`
// 	GrossOutput    string `json:"GrossOutput"`
// 	NetOutput      string `json:"NetOutput"`
// 	FillRatio      string `json:"FillRatio"`
// 	Fee            string `json:"Fee"`
// 	FeeAsset       string `json:"FeeAsset"`
// }

// type Opportunity struct {
// 	ID                  string            `json:"ID"`
// 	Steps               []OpportunityStep `json:"Steps"`
// 	StartAsset          string            `json:"StartAsset"`
// 	RequestedStartAmount string           `json:"RequestedStartAmount"`
// 	ExecutedStartAmount string            `json:"ExecutedStartAmount"`
// 	EndAmount           string            `json:"EndAmount"`
// 	FillRatio           string            `json:"FillRatio"`
// 	ProfitAbs           string            `json:"ProfitAbs"`
// 	ProfitPct           string            `json:"ProfitPct"`
// }

// type Route struct {
// 	Edges []Edge `json:"Edges"`
// 	Key   string `json:"Key"`
// }

// type OpportunityFull struct {
// 	ID                   string            `json:"ID"`
// 	Route                Route             `json:"Route"`
// 	Steps                []OpportunityStep `json:"Steps"`
// 	StartAsset           string            `json:"StartAsset"`
// 	RequestedStartAmount string            `json:"RequestedStartAmount"`
// 	ExecutedStartAmount  string            `json:"ExecutedStartAmount"`
// 	EndAmount            string            `json:"EndAmount"`
// 	FillRatio             string            `json:"FillRatio"`
// 	ProfitAbs             string            `json:"ProfitAbs"`
// 	ProfitPct             string            `json:"ProfitPct"`
// }

// type ExecutionStep struct {
// 	StepID       int         `json:"StepID"`
// 	PlannedInput AssetAmount `json:"PlannedInput"`
// 	RequestedInput AssetAmount `json:"RequestedInput"`
// 	ActualInput  AssetAmount `json:"ActualInput"`
// 	GrossOutput  AssetAmount `json:"GrossOutput"`
// 	NetOutput    AssetAmount `json:"NetOutput"`
// 	FillRatio    string      `json:"FillRatio"`
// 	Result       string      `json:"Result"`
// }

// type Execution struct {
// 	ID            string          `json:"ID"`
// 	OpportunityID string          `json:"OpportunityID"`
// 	InitialState  []AssetAmount   `json:"InitialState"`
// 	ActualState   []AssetAmount   `json:"ActualState"`
// 	Steps         []ExecutionStep `json:"Steps"`
// }

// type TradeDetail struct {
// 	ExecutionID    string
// 	OpportunityID  string

// 	ExpectedPnL    float64
// 	ActualPnL      float64
// 	PnLDeviation   float64

// 	ExpectedProfit bool
// 	ActualProfit   bool

// 	StepID int

// 	Symbol    string
// 	FromAsset string
// 	ToAsset   string

// 	ExpectedInput  float64
// 	ActualInput    float64
// 	InputDeviation float64

// 	ExpectedOutput  float64
// 	ActualOutput    float64
// 	OutputDeviation float64

// 	ExpectedPrice float64
// 	ActualPrice   float64
// 	PriceDeviationPct float64

// 	ResidualAsset string
// 	ResidualQty   float64
// }

// type ExecutionSummary struct {
// 	ExecutionID   string
// 	OpportunityID string

// 	ExpectedPnL  float64
// 	ActualPnL    float64
// 	Deviation    float64

// 	ExpectedProfit bool
// 	ActualProfit   bool
// }

// type StepAggregate struct {
// 	StepID int

// 	Executions int

// 	ExpectedOutput float64
// 	ActualOutput   float64
// 	OutputDeviation float64

// 	ExpectedInput float64
// 	ActualInput   float64
// 	InputDeviation float64

// 	PriceDeviationPct float64
// 	PriceSamples      int

// 	WorseCount int
// 	BetterCount int
// }

// type SymbolAggregate struct {
// 	Symbol string

// 	Executions int

// 	OutputDeviation float64
// 	InputDeviation  float64

// 	PriceDeviationPct float64
// 	PriceSamples      int

// 	WorseCount  int
// 	BetterCount int
// }

// type AuditReport struct {
// 	Executions int
// 	Matched    int

// 	Profitable int
// 	Losing     int
// 	ZeroPnL    int

// 	ExpectedProfitable int
// 	ExpectedLosses     int

// 	ExpectedPositiveActualPositive int
// 	ExpectedPositiveActualNegative int
// 	ExpectedNegativeActualPositive int

// 	InitialUSDT float64
// 	FinalUSDT   float64
// 	USDTDelta   float64

// 	InitialPortfolio float64
// 	FinalPortfolio   float64
// 	RealizedPnL      float64

// 	ExpectedPnL float64
// 	Deviation   float64

// 	ExecutionSummaries []ExecutionSummary
// 	TradeDetails       []TradeDetail

// 	StepsByID map[int]*StepAggregate
// 	Symbols   map[string]*SymbolAggregate

// 	Residuals map[string]float64
// }

// func parseFloat(s string) float64 {
// 	if s == "" {
// 		return 0
// 	}

// 	v, err := strconv.ParseFloat(s, 64)
// 	if err != nil {
// 		return 0
// 	}

// 	return v
// }

// func almostEqual(a, b float64) bool {
// 	return math.Abs(a-b) < 1e-12
// }

// func loadJSONLines[T any](path string) ([]T, error) {
// 	file, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	var result []T

// 	scanner := bufio.NewScanner(file)
// 	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

// 	line := 0

// 	for scanner.Scan() {
// 		line++

// 		data := strings.TrimSpace(scanner.Text())
// 		if data == "" {
// 			continue
// 		}

// 		var item T
// 		if err := json.Unmarshal([]byte(data), &item); err != nil {
// 			return nil, fmt.Errorf(
// 				"parse %s line %d: %w",
// 				path,
// 				line,
// 				err,
// 			)
// 		}

// 		result = append(result, item)
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	return result, nil
// }

// func loadAllJSONLines[T any](dir string) ([]T, error) {
// 	entries, err := os.ReadDir(dir)
// 	if err != nil {
// 		return nil, err
// 	}

// 	sort.Slice(entries, func(i, j int) bool {
// 		return entries[i].Name() < entries[j].Name()
// 	})

// 	var all []T

// 	for _, entry := range entries {
// 		if entry.IsDir() {
// 			continue
// 		}

// 		path := filepath.Join(dir, entry.Name())

// 		items, err := loadJSONLines[T](path)
// 		if err != nil {
// 			return nil, err
// 		}

// 		all = append(all, items...)
// 	}

// 	return all, nil
// }

// func findOpportunity(
// 	opportunities []OpportunityFull,
// 	id string,
// ) *OpportunityFull {
// 	for i := range opportunities {
// 		if opportunities[i].ID == id {
// 			return &opportunities[i]
// 		}
// 	}

// 	return nil
// }

// func findExecutionStep(
// 	execution Execution,
// 	opportunityStep OpportunityStep,
// 	stepIndex int,
// ) *ExecutionStep {
// 	for i := range execution.Steps {
// 		step := &execution.Steps[i]

// 		if step.StepID != stepIndex {
// 			continue
// 		}

// 		if step.ActualInput.Asset == opportunityStep.Edge.From &&
// 			step.NetOutput.Asset == opportunityStep.Edge.To {
// 			return step
// 		}
// 	}

// 	for i := range execution.Steps {
// 		step := &execution.Steps[i]

// 		if step.StepID != stepIndex {
// 			continue
// 		}

// 		return step
// 	}

// 	return nil
// }

// func getAssetBalance(
// 	state []AssetAmount,
// 	asset string,
// ) float64 {
// 	for _, item := range state {
// 		if item.Asset == asset {
// 			return parseFloat(item.Quantity)
// 		}
// 	}

// 	return 0
// }

// func calculateInitialUSDT(execution Execution) float64 {
// 	return getAssetBalance(execution.InitialState, "USDT")
// }

// func calculateFinalUSDT(execution Execution) float64 {
// 	return getAssetBalance(execution.ActualState, "USDT")
// }

// func calculateResiduals(
// 	executions []Execution,
// ) map[string]float64 {
// 	residuals := make(map[string]float64)

// 	for _, execution := range executions {
// 		for _, item := range execution.ActualState {
// 			if item.Asset == "USDT" {
// 				continue
// 			}

// 			qty := parseFloat(item.Quantity)

// 			if math.Abs(qty) < 1e-15 {
// 				continue
// 			}

// 			residuals[item.Asset] += qty
// 		}
// 	}

// 	return residuals
// }

// func directResidualValue(
// 	asset string,
// 	qty float64,
// 	opp OpportunityFull,
// ) float64 {
// 	if asset == "USDT" {
// 		return qty
// 	}

// 	// First try direct edge asset -> USDT.
// 	for _, step := range opp.Steps {
// 		if step.Edge.From == asset && step.Edge.To == "USDT" {
// 			price := parseFloat(step.AvgPrice)

// 			if price > 0 {
// 				// SELL:
// 				// asset quantity * price = USDT
// 				if step.Edge.Direction == 1 {
// 					return qty * price
// 				}
// 			}
// 		}
// 	}

// 	// Then try direct USDT -> asset.
// 	for _, step := range opp.Steps {
// 		if step.Edge.From == "USDT" && step.Edge.To == asset {
// 			price := parseFloat(step.AvgPrice)

// 			if price > 0 {
// 				// BUY:
// 				// asset quantity / price = USDT
// 				if step.Edge.Direction == 0 {
// 					return qty * price
// 				}
// 			}
// 		}
// 	}

// 	return 0
// }

// func calculateResidualValue(
// 	residuals map[string]float64,
// 	opportunities []OpportunityFull,
// ) float64 {
// 	total := 0.0

// 	for asset, qty := range residuals {
// 		if asset == "USDT" {
// 			continue
// 		}

// 		// Use the most recent opportunity containing a direct
// 		// conversion between the asset and USDT.
// 		for i := len(opportunities) - 1; i >= 0; i-- {
// 			value := directResidualValue(asset, qty, opportunities[i])

// 			if value > 0 {
// 				total += value
// 				break
// 			}
// 		}
// 	}

// 	return total
// }

// func calculateActualPnL(
// 	execution Execution,
// 	opportunity OpportunityFull,
// ) float64 {
// 	initial := calculateInitialUSDT(execution)
// 	final := calculateFinalUSDT(execution)

// 	// Start with USDT difference.
// 	pnl := final - initial

// 	// Add residual non-USDT assets using prices from this opportunity.
// 	for _, item := range execution.ActualState {
// 		if item.Asset == "USDT" {
// 			continue
// 		}

// 		qty := parseFloat(item.Quantity)

// 		if math.Abs(qty) < 1e-15 {
// 			continue
// 		}

// 		value := directResidualValue(item.Asset, qty, opportunity)

// 		pnl += value
// 	}

// 	return pnl
// }

// func buildReport(
// 	opportunities []OpportunityFull,
// 	executions []Execution,
// ) AuditReport {
// 	report := AuditReport{
// 		StepsByID: make(map[int]*StepAggregate),
// 		Symbols:   make(map[string]*SymbolAggregate),
// 		Residuals: make(map[string]float64),
// 	}

// 	report.Executions = len(executions)
// 	report.Residuals = calculateResiduals(executions)

// 	for _, execution := range executions {
// 		opp := findOpportunity(opportunities, execution.OpportunityID)

// 		if opp == nil {
// 			continue
// 		}

// 		report.Matched++

// 		expectedPnL := parseFloat(opp.ProfitAbs)
// 		actualPnL := calculateActualPnL(execution, *opp)
// 		deviation := actualPnL - expectedPnL

// 		expectedProfit := expectedPnL > 0
// 		actualProfit := actualPnL > 0

// 		summary := ExecutionSummary{
// 			ExecutionID:   execution.ID,
// 			OpportunityID: execution.OpportunityID,
// 			ExpectedPnL:   expectedPnL,
// 			ActualPnL:     actualPnL,
// 			Deviation:     deviation,
// 			ExpectedProfit: expectedProfit,
// 			ActualProfit:   actualProfit,
// 		}

// 		report.ExecutionSummaries = append(
// 			report.ExecutionSummaries,
// 			summary,
// 		)

// 		report.ExpectedPnL += expectedPnL
// 		report.Deviation += deviation

// 		switch {
// 		case actualPnL > 1e-12:
// 			report.Profitable++
// 		case actualPnL < -1e-12:
// 			report.Losing++
// 		default:
// 			report.ZeroPnL++
// 		}

// 		if expectedProfit {
// 			report.ExpectedProfitable++

// 			if actualProfit {
// 				report.ExpectedPositiveActualPositive++
// 			} else if actualPnL < -1e-12 {
// 				report.ExpectedPositiveActualNegative++
// 			}
// 		} else {
// 			report.ExpectedLosses++

// 			if actualProfit {
// 				report.ExpectedNegativeActualPositive++
// 			}
// 		}

// 		initialUSDT := calculateInitialUSDT(execution)
// 		finalUSDT := calculateFinalUSDT(execution)

// 		if report.Matched == 1 {
// 			report.InitialUSDT = initialUSDT
// 			report.FinalUSDT = finalUSDT
// 		} else {
// 			report.FinalUSDT = finalUSDT
// 		}

// 		for stepIndex, opportunityStep := range opp.Steps {
// 			executionStep := findExecutionStep(
// 				execution,
// 				opportunityStep,
// 				stepIndex,
// 			)

// 			if executionStep == nil {
// 				continue
// 			}

// 			expectedInput := parseFloat(opportunityStep.ExecutedInput)
// 			actualInput := parseFloat(executionStep.ActualInput.Quantity)

// 			expectedOutput := parseFloat(opportunityStep.NetOutput)
// 			actualOutput := parseFloat(executionStep.NetOutput.Quantity)

// 			inputDeviation := actualInput - expectedInput
// 			outputDeviation := actualOutput - expectedOutput

// 			expectedPrice := parseFloat(opportunityStep.AvgPrice)

// 			actualPrice := 0.0

// 			if actualInput > 0 && actualOutput > 0 {
// 				// Effective price depends on direction.
// 				if opportunityStep.Edge.Direction == 0 {
// 					// BUY:
// 					// output / input = base asset per quote.
// 					actualPrice = actualInput / actualOutput
// 				} else {
// 					// SELL:
// 					// output / input = quote per base.
// 					actualPrice = actualOutput / actualInput
// 				}
// 			}

// 			priceDeviationPct := 0.0

// 			if expectedPrice > 0 && actualPrice > 0 {
// 				priceDeviationPct =
// 					(actualPrice - expectedPrice) /
// 						expectedPrice *
// 						100

// 				if opportunityStep.Edge.Direction == 0 {
// 					// For BUY lower price is better.
// 					priceDeviationPct =
// 						(expectedPrice - actualPrice) /
// 							expectedPrice *
// 							100
// 				}
// 			}

// 			detail := TradeDetail{
// 				ExecutionID:   execution.ID,
// 				OpportunityID: execution.OpportunityID,

// 				ExpectedPnL:  expectedPnL,
// 				ActualPnL:    actualPnL,
// 				PnLDeviation: deviation,

// 				ExpectedProfit: expectedProfit,
// 				ActualProfit:   actualProfit,

// 				StepID: stepIndex,

// 				Symbol:    opportunityStep.Edge.Symbol,
// 				FromAsset: opportunityStep.Edge.From,
// 				ToAsset:   opportunityStep.Edge.To,

// 				ExpectedInput:  expectedInput,
// 				ActualInput:    actualInput,
// 				InputDeviation: inputDeviation,

// 				ExpectedOutput:  expectedOutput,
// 				ActualOutput:   actualOutput,
// 				OutputDeviation: outputDeviation,

// 				ExpectedPrice:     expectedPrice,
// 				ActualPrice:       actualPrice,
// 				PriceDeviationPct: priceDeviationPct,
// 			}

// 			report.TradeDetails = append(
// 				report.TradeDetails,
// 				detail,
// 			)

// 			stepAgg := report.StepsByID[stepIndex]

// 			if stepAgg == nil {
// 				stepAgg = &StepAggregate{
// 					StepID: stepIndex,
// 				}

// 				report.StepsByID[stepIndex] = stepAgg
// 			}

// 			stepAgg.Executions++

// 			stepAgg.ExpectedInput += expectedInput
// 			stepAgg.ActualInput += actualInput
// 			stepAgg.InputDeviation += inputDeviation

// 			stepAgg.ExpectedOutput += expectedOutput
// 			stepAgg.ActualOutput += actualOutput
// 			stepAgg.OutputDeviation += outputDeviation

// 			if math.Abs(priceDeviationPct) > 0 {
// 				stepAgg.PriceDeviationPct += priceDeviationPct
// 				stepAgg.PriceSamples++
// 			}

// 			if priceDeviationPct < -1e-9 {
// 				stepAgg.WorseCount++
// 			} else if priceDeviationPct > 1e-9 {
// 				stepAgg.BetterCount++
// 			}

// 			symbolAgg := report.Symbols[opportunityStep.Edge.Symbol]

// 			if symbolAgg == nil {
// 				symbolAgg = &SymbolAggregate{
// 					Symbol: opportunityStep.Edge.Symbol,
// 				}

// 				report.Symbols[opportunityStep.Edge.Symbol] = symbolAgg
// 			}

// 			symbolAgg.Executions++

// 			symbolAgg.InputDeviation += inputDeviation
// 			symbolAgg.OutputDeviation += outputDeviation

// 			if math.Abs(priceDeviationPct) > 0 {
// 				symbolAgg.PriceDeviationPct += priceDeviationPct
// 				symbolAgg.PriceSamples++
// 			}

// 			if priceDeviationPct < -1e-9 {
// 				symbolAgg.WorseCount++
// 			} else if priceDeviationPct > 1e-9 {
// 				symbolAgg.BetterCount++
// 			}
// 		}
// 	}

// 	report.USDTDelta = report.FinalUSDT - report.InitialUSDT

// 	report.InitialPortfolio = report.InitialUSDT

// 	residualValue := calculateResidualValue(
// 		report.Residuals,
// 		opportunities,
// 	)

// 	report.FinalPortfolio =
// 		report.FinalUSDT + residualValue

// 	report.RealizedPnL =
// 		report.FinalPortfolio - report.InitialPortfolio

// 	return report
// }

// func writeTradeCSV(
// 	path string,
// 	details []TradeDetail,
// ) error {
// 	if err := os.MkdirAll(
// 		filepath.Dir(path),
// 		0755,
// 	); err != nil {
// 		return err
// 	}

// 	file, err := os.Create(path)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	writer := csv.NewWriter(file)
// 	defer writer.Flush()

// 	header := []string{
// 		"execution_id",
// 		"opportunity_id",

// 		"expected_pnl",
// 		"actual_pnl",
// 		"pnl_deviation",

// 		"expected_profitable",
// 		"actual_profitable",

// 		"step_id",
// 		"symbol",
// 		"from_asset",
// 		"to_asset",

// 		"expected_input",
// 		"actual_input",
// 		"input_deviation",

// 		"expected_output",
// 		"actual_output",
// 		"output_deviation",

// 		"expected_price",
// 		"actual_price",
// 		"price_deviation_pct",
// 	}

// 	if err := writer.Write(header); err != nil {
// 		return err
// 	}

// 	for _, d := range details {
// 		row := []string{
// 			d.ExecutionID,
// 			d.OpportunityID,

// 			formatFloat(d.ExpectedPnL),
// 			formatFloat(d.ActualPnL),
// 			formatFloat(d.PnLDeviation),

// 			strconv.FormatBool(d.ExpectedProfit),
// 			strconv.FormatBool(d.ActualProfit),

// 			strconv.Itoa(d.StepID),
// 			d.Symbol,
// 			d.FromAsset,
// 			d.ToAsset,

// 			formatFloat(d.ExpectedInput),
// 			formatFloat(d.ActualInput),
// 			formatFloat(d.InputDeviation),

// 			formatFloat(d.ExpectedOutput),
// 			formatFloat(d.ActualOutput),
// 			formatFloat(d.OutputDeviation),

// 			formatFloat(d.ExpectedPrice),
// 			formatFloat(d.ActualPrice),
// 			formatFloat(d.PriceDeviationPct),
// 		}

// 		if err := writer.Write(row); err != nil {
// 			return err
// 		}
// 	}

// 	return writer.Error()
// }

// func formatFloat(v float64) string {
// 	return strconv.FormatFloat(v, 'f', 12, 64)
// }

// func printSeparator() {
// 	fmt.Println("------------------------------------------------")
// }

// func PrintReport(
// 	report AuditReport,
// 	opportunities []OpportunityFull,
// ) {
// 	fmt.Println("==============================================")
// 	fmt.Println("              TRADE ECONOMIC AUDIT")
// 	fmt.Println("==============================================")

// 	fmt.Println()
// 	fmt.Println("==============================================")
// 	fmt.Println("                  SUMMARY")
// 	fmt.Println("==============================================")

// 	fmt.Printf(
// 		"Executions:                 %d\n",
// 		report.Executions,
// 	)

// 	fmt.Printf(
// 		"Matched opportunities:      %d\n",
// 		report.Matched,
// 	)

// 	fmt.Println()

// 	fmt.Printf(
// 		"Profitable executions:      %d\n",
// 		report.Profitable,
// 	)

// 	fmt.Printf(
// 		"Losing executions:          %d\n",
// 		report.Losing,
// 	)

// 	fmt.Printf(
// 		"Zero PnL executions:        %d\n",
// 		report.ZeroPnL,
// 	)

// 	fmt.Println()

// 	fmt.Printf(
// 		"Expected profitable:        %d\n",
// 		report.ExpectedProfitable,
// 	)

// 	fmt.Printf(
// 		"Expected losses:            %d\n",
// 		report.ExpectedLosses,
// 	)

// 	fmt.Printf(
// 		"Expected + Actual +:        %d\n",
// 		report.ExpectedPositiveActualPositive,
// 	)

// 	fmt.Printf(
// 		"Expected + Actual -:        %d\n",
// 		report.ExpectedPositiveActualNegative,
// 	)

// 	fmt.Printf(
// 		"Expected - Actual +:        %d\n",
// 		report.ExpectedNegativeActualPositive,
// 	)

// 	fmt.Println()

// 	fmt.Printf(
// 		"Initial USDT:               %.12f\n",
// 		report.InitialUSDT,
// 	)

// 	fmt.Printf(
// 		"Final USDT:                 %.12f\n",
// 		report.FinalUSDT,
// 	)

// 	fmt.Printf(
// 		"USDT delta:                 %+.12f\n",
// 		report.USDTDelta,
// 	)

// 	fmt.Println()

// 	fmt.Printf(
// 		"Initial portfolio value:    %.12f USDT\n",
// 		report.InitialPortfolio,
// 	)

// 	fmt.Printf(
// 		"Final portfolio value:      %.12f USDT\n",
// 		report.FinalPortfolio,
// 	)

// 	fmt.Printf(
// 		"REALIZED PnL:               %+.12f USDT\n",
// 		report.RealizedPnL,
// 	)

// 	fmt.Printf(
// 		"Expected PnL:               %+.12f USDT\n",
// 		report.ExpectedPnL,
// 	)

// 	fmt.Printf(
// 		"Actual - expected:          %+.12f USDT\n",
// 		report.Deviation,
// 	)

// 	if report.Matched > 0 {
// 		fmt.Printf(
// 			"Average PnL/execution:      %+.12f USDT\n",
// 			report.RealizedPnL/float64(report.Matched),
// 		)

// 		fmt.Printf(
// 			"Average expected:           %+.12f USDT\n",
// 			report.ExpectedPnL/float64(report.Matched),
// 		)

// 		fmt.Printf(
// 			"Average deviation:          %+.12f USDT\n",
// 			report.Deviation/float64(report.Matched),
// 		)
// 	}

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("              EXPECTED vs ACTUAL")
// 	fmt.Println("==============================================")

// 	if report.ExpectedProfitable > 0 {
// 		fmt.Printf(
// 			"Positive opportunities:     %d\n",
// 			report.ExpectedProfitable,
// 		)

// 		fmt.Printf(
// 			"Stayed positive:            %d (%.2f%%)\n",
// 			report.ExpectedPositiveActualPositive,
// 			100*float64(report.ExpectedPositiveActualPositive)/
// 				float64(report.ExpectedProfitable),
// 		)

// 		fmt.Printf(
// 			"Turned negative:            %d (%.2f%%)\n",
// 			report.ExpectedPositiveActualNegative,
// 			100*float64(report.ExpectedPositiveActualNegative)/
// 				float64(report.ExpectedProfitable),
// 		)
// 	}

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("                STEP ANALYSIS")
// 	fmt.Println("==============================================")

// 	stepIDs := make([]int, 0, len(report.StepsByID))

// 	for stepID := range report.StepsByID {
// 		stepIDs = append(stepIDs, stepID)
// 	}

// 	sort.Ints(stepIDs)

// 	for _, stepID := range stepIDs {
// 		agg := report.StepsByID[stepID]

// 		fmt.Printf(
// 			"STEP %d   executions=%d\n",
// 			agg.StepID,
// 			agg.Executions,
// 		)

// 		fmt.Printf(
// 			"  Input deviation:         %+.12f\n",
// 			agg.InputDeviation,
// 		)

// 		fmt.Printf(
// 			"  Output deviation:        %+.12f\n",
// 			agg.OutputDeviation,
// 		)

// 		if agg.PriceSamples > 0 {
// 			avgPriceDeviation :=
// 				agg.PriceDeviationPct /
// 					float64(agg.PriceSamples)

// 			fmt.Printf(
// 				"  Avg price deviation:     %+.6f%%\n",
// 				avgPriceDeviation,
// 			)
// 		}

// 		fmt.Printf(
// 			"  Worse price executions:  %d\n",
// 			agg.WorseCount,
// 		)

// 		fmt.Printf(
// 			"  Better price executions: %d\n",
// 			agg.BetterCount,
// 		)

// 		fmt.Println()
// 	}

// 	fmt.Println("==============================================")
// 	fmt.Println("               SYMBOL ANALYSIS")
// 	fmt.Println("==============================================")

// 	symbols := make([]*SymbolAggregate, 0, len(report.Symbols))

// 	for _, agg := range report.Symbols {
// 		symbols = append(symbols, agg)
// 	}

// 	sort.Slice(symbols, func(i, j int) bool {
// 		return math.Abs(symbols[i].OutputDeviation) >
// 			math.Abs(symbols[j].OutputDeviation)
// 	})

// 	for _, agg := range symbols {
// 		fmt.Printf(
// 			"%-12s executions=%d\n",
// 			agg.Symbol,
// 			agg.Executions,
// 		)

// 		fmt.Printf(
// 			"  Input deviation:         %+.12f\n",
// 			agg.InputDeviation,
// 		)

// 		fmt.Printf(
// 			"  Output deviation:        %+.12f\n",
// 			agg.OutputDeviation,
// 		)

// 		if agg.PriceSamples > 0 {
// 			avgPriceDeviation :=
// 				agg.PriceDeviationPct /
// 					float64(agg.PriceSamples)

// 			fmt.Printf(
// 				"  Avg price deviation:     %+.6f%%\n",
// 				avgPriceDeviation,
// 			)
// 		}

// 		fmt.Printf(
// 			"  Worse: %d   Better: %d\n",
// 			agg.WorseCount,
// 			agg.BetterCount,
// 		)

// 		fmt.Println()
// 	}

// 	fmt.Println("==============================================")
// 	fmt.Println("              WORST EXECUTIONS")
// 	fmt.Println("==============================================")

// 	summaries := append(
// 		[]ExecutionSummary(nil),
// 		report.ExecutionSummaries...,
// 	)

// 	sort.Slice(
// 		summaries,
// 		func(i, j int) bool {
// 			return summaries[i].Deviation <
// 				summaries[j].Deviation
// 		},
// 	)

// 	limit := 10

// 	if len(summaries) < limit {
// 		limit = len(summaries)
// 	}

// 	for i := 0; i < limit; i++ {
// 		item := summaries[i]

// 		fmt.Printf(
// 			"%2d. %-16s expected=%+.6f actual=%+.6f deviation=%+.6f\n",
// 			i+1,
// 			item.ExecutionID,
// 			item.ExpectedPnL,
// 			item.ActualPnL,
// 			item.Deviation,
// 		)
// 	}

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("              BEST EXECUTIONS")
// 	fmt.Println("==============================================")

// 	sort.Slice(
// 		summaries,
// 		func(i, j int) bool {
// 			return summaries[i].Deviation >
// 				summaries[j].Deviation
// 		},
// 	)

// 	for i := 0; i < limit; i++ {
// 		item := summaries[i]

// 		fmt.Printf(
// 			"%2d. %-16s expected=%+.6f actual=%+.6f deviation=%+.6f\n",
// 			i+1,
// 			item.ExecutionID,
// 			item.ExpectedPnL,
// 			item.ActualPnL,
// 			item.Deviation,
// 		)
// 	}

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("             RESIDUAL ASSETS")
// 	fmt.Println("==============================================")

// 	type residualItem struct {
// 		asset string
// 		qty   float64
// 	}

// 	residualList := make([]residualItem, 0, len(report.Residuals))

// 	for asset, qty := range report.Residuals {
// 		residualList = append(
// 			residualList,
// 			residualItem{
// 				asset: asset,
// 				qty:   qty,
// 			},
// 		)
// 	}

// 	sort.Slice(
// 		residualList,
// 		func(i, j int) bool {
// 			return residualList[i].asset <
// 				residualList[j].asset
// 		},
// 	)

// 	for _, item := range residualList {
// 		fmt.Printf(
// 			"%-10s quantity=%-20.12f\n",
// 			item.asset,
// 			item.qty,
// 		)
// 	}

// 	residualValue :=
// 		report.FinalPortfolio -
// 			report.FinalUSDT

// 	fmt.Println()

// 	fmt.Printf(
// 		"Total residual value:      %.12f USDT\n",
// 		residualValue,
// 	)

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("                  WARNINGS")
// 	fmt.Println("==============================================")

// 	fmt.Printf(
// 		"USDT down, portfolio up:   %d\n",
// 		countUSDTDownPortfolioUp(report),
// 	)

// 	fmt.Printf(
// 		"Real portfolio losses:     %d\n",
// 		report.Losing,
// 	)

// 	fmt.Printf(
// 		"PnL mismatches:             %d\n",
// 		countPnLMismatches(report),
// 	)

// 	fmt.Printf(
// 		"Missing opportunities:      %d\n",
// 		report.Executions-report.Matched,
// 	)

// 	fmt.Println()

// 	if len(opportunities) > 0 {
// 		expectedReturn := 0.0

// 		for _, opp := range opportunities {
// 			start := parseFloat(opp.RequestedStartAmount)

// 			if start > 0 {
// 				expectedReturn +=
// 					parseFloat(opp.ProfitAbs) /
// 						start
// 			}
// 		}

// 		fmt.Printf(
// 			"Expected return sum:        %.6f %%\n",
// 			expectedReturn*100,
// 		)
// 	}

// 	if report.InitialPortfolio > 0 {
// 		fmt.Printf(
// 			"Realized return:            %.6f %%\n",
// 			report.RealizedPnL/
// 				report.InitialPortfolio*
// 				100,
// 		)
// 	}

// 	if report.InitialPortfolio > 0 {
// 		fmt.Printf(
// 			"Expected return:            %.6f %%\n",
// 			report.ExpectedPnL/
// 				report.InitialPortfolio*
// 				100,
// 		)
// 	}

// 	fmt.Println()

// 	fmt.Println("==============================================")
// 	fmt.Println("                   OUTPUT")
// 	fmt.Println("==============================================")

// 	fmt.Println(
// 		"Detailed CSV: data/audit/trade-detail.csv",
// 	)

// 	fmt.Println("==============================================")
// }

// func countPnLMismatches(
// 	report AuditReport,
// ) int {
// 	count := 0

// 	for _, item := range report.ExecutionSummaries {
// 		if math.Abs(item.Deviation) > 1e-12 {
// 			count++
// 		}
// 	}

// 	return count
// }

// func countUSDTDownPortfolioUp(
// 	report AuditReport,
// ) int {
// 	// This metric is calculated from the complete execution
// 	// history in the original report, so here we use the
// 	// presence of residual assets as the portfolio-level reason.
// 	//
// 	// Per-execution exact accounting is intentionally kept
// 	// in the detailed CSV.

// 	if report.FinalPortfolio > report.InitialPortfolio &&
// 		report.FinalUSDT < report.InitialUSDT {
// 		return 1
// 	}

// 	return 0
// }

// func RunAudit(dataDir string) (AuditReport, error) {
// 	opportunityDir := filepath.Join(
// 		dataDir,
// 		"opportunities",
// 	)

// 	executionDir := filepath.Join(
// 		dataDir,
// 		"executions",
// 	)

// 	opportunities, err :=
// 		loadAllJSONLines[OpportunityFull](
// 			opportunityDir,
// 		)

// 	if err != nil {
// 		return AuditReport{}, fmt.Errorf(
// 			"load opportunities: %w",
// 			err,
// 		)
// 	}

// 	executions, err :=
// 		loadAllJSONLines[Execution](
// 			executionDir,
// 		)

// 	if err != nil {
// 		return AuditReport{}, fmt.Errorf(
// 			"load executions: %w",
// 			err,
// 		)
// 	}

// 	report := buildReport(
// 		opportunities,
// 		executions,
// 	)

// 	csvPath := filepath.Join(
// 		dataDir,
// 		"audit",
// 		"trade-detail.csv",
// 	)

// 	if err := writeTradeCSV(
// 		csvPath,
// 		report.TradeDetails,
// 	); err != nil {
// 		return AuditReport{}, fmt.Errorf(
// 			"write trade csv: %w",
// 			err,
// 		)
// 	}

// 	return report, nil
// }