package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type AssetAmount struct {
	Asset    string          `json:"Asset"`
	Quantity decimal.Decimal `json:"Quantity"`
}

type BalanceSnapshot struct {
	Asset    string          `json:"Asset"`
	Quantity decimal.Decimal `json:"Quantity"`
}

type RecoveryState map[string]decimal.Decimal

type Opportunity struct {
	ID                   string          `json:"ID"`
	RequestedStartAmount decimal.Decimal `json:"RequestedStartAmount"`
	ExecutedStartAmount  decimal.Decimal `json:"ExecutedStartAmount"`
	EndAmount            decimal.Decimal `json:"EndAmount"`
	FillRatio            decimal.Decimal `json:"FillRatio"`
	ProfitAbs            decimal.Decimal `json:"ProfitAbs"`
	ProfitPct            decimal.Decimal `json:"ProfitPct"`
	Timestamp            time.Time       `json:"Timestamp"`
}

type ExecutionStep struct {
	StepID         int             `json:"StepID"`
	PlannedInput   AssetAmount     `json:"PlannedInput"`
	RequestedInput AssetAmount     `json:"RequestedInput"`
	ActualInput    AssetAmount     `json:"ActualInput"`
	GrossOutput    AssetAmount     `json:"GrossOutput"`
	Fee            AssetAmount     `json:"Fee"`
	NetOutput      AssetAmount     `json:"NetOutput"`
	FillRatio      decimal.Decimal `json:"FillRatio"`
	Result         string          `json:"Result"`
	Error          string          `json:"Error"`
	StartedAt      time.Time       `json:"StartedAt"`
	CompletedAt    time.Time       `json:"CompletedAt"`
}

type Execution struct {
	ID            string            `json:"ID"`
	OpportunityID string            `json:"OpportunityID"`
	RecoveryID    string            `json:"RecoveryID"`
	State         string            `json:"State"`
	StartedAt     time.Time         `json:"StartedAt"`
	UpdatedAt     time.Time         `json:"UpdatedAt"`
	CompletedAt   *time.Time        `json:"CompletedAt"`
	CurrentStep   int               `json:"CurrentStep"`
	TotalSteps    int               `json:"TotalSteps"`
	ActualSteps   int               `json:"ActualSteps"`
	InitialState  []BalanceSnapshot `json:"InitialState"`
	ActualState   []BalanceSnapshot `json:"ActualState"`
	Steps         []ExecutionStep   `json:"Steps"`
	Error         string            `json:"Error"`
}

type RecoveryAttempt struct {
	Index          int           `json:"Index"`
	PlansCreated   int           `json:"PlansCreated"`
	PlansAttempted int           `json:"PlansAttempted"`
	PlansSucceeded int           `json:"PlansSucceeded"`
	PlansFailed    int           `json:"PlansFailed"`
	StateBefore    RecoveryState `json:"StateBefore"`
	StateAfter     RecoveryState `json:"StateAfter"`
	Changed        bool          `json:"Changed"`
	StartedAt      time.Time     `json:"StartedAt"`
	CompletedAt    *time.Time    `json:"CompletedAt"`
	Error          string        `json:"Error"`
}

type Recovery struct {
	ID            string            `json:"ID"`
	ExecutionID   string            `json:"ExecutionID"`
	ProcessState  string            `json:"ProcessState"`
	Reason        string            `json:"Reason"`
	InitialState  RecoveryState     `json:"InitialState"`
	ActualState   RecoveryState     `json:"ActualState"`
	StartedAt     time.Time         `json:"StartedAt"`
	UpdatedAt     time.Time         `json:"UpdatedAt"`
	CompletedAt   *time.Time        `json:"CompletedAt"`
	Attempts      []RecoveryAttempt `json:"Attempts"`
	PlansCreated  int               `json:"PlansCreated"`
	PlansExecuted int               `json:"PlansExecuted"`
	Error         string            `json:"Error"`
}

type Dataset struct {
	Opportunities map[string]Opportunity
	Executions    map[string]Execution
	Recoveries    map[string]Recovery
}

func main() {
	dataDir := "data"
	outputFile := filepath.Join(dataDir, "trade-report.txt")

	data, err := loadDataset(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := writeReport(outputFile, data); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Trade report written to %s\n", outputFile)
	fmt.Printf(
		"Loaded: opportunities=%d executions=%d recoveries=%d\n",
		len(data.Opportunities),
		len(data.Executions),
		len(data.Recoveries),
	)
}

func loadDataset(root string) (Dataset, error) {
	data := Dataset{
		Opportunities: make(map[string]Opportunity),
		Executions:    make(map[string]Execution),
		Recoveries:    make(map[string]Recovery),
	}

	err := filepath.WalkDir(root, func(
		path string,
		entry fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		dir := filepath.Base(filepath.Dir(path))

		switch dir {
		case "opportunities":
			if err := readJSONL(path, func(raw []byte) error {
				var value Opportunity
				if err := json.Unmarshal(raw, &value); err != nil {
					return err
				}

				if value.ID != "" {
					data.Opportunities[value.ID] = value
				}

				return nil
			}); err != nil {
				return fmt.Errorf("read opportunities %s: %w", path, err)
			}

		case "executions":
			if err := readJSONL(path, func(raw []byte) error {
				var value Execution
				if err := json.Unmarshal(raw, &value); err != nil {
					return err
				}

				if value.ID != "" {
					data.Executions[value.ID] = value
				}

				return nil
			}); err != nil {
				return fmt.Errorf("read executions %s: %w", path, err)
			}

		case "recoveries":
			if err := readJSONL(path, func(raw []byte) error {
				var value Recovery
				if err := json.Unmarshal(raw, &value); err != nil {
					return err
				}

				if value.ID != "" {
					data.Recoveries[value.ID] = value
				}

				return nil
			}); err != nil {
				return fmt.Errorf("read recoveries %s: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return Dataset{}, err
	}

	return data, nil
}

func readJSONL(path string, fn func([]byte) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// JSON objects can become large.
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	line := 0

	for scanner.Scan() {
		line++

		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}

		if err := fn(raw); err != nil {
			return fmt.Errorf(
				"%s:%d: %w",
				path,
				line,
				err,
			)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func writeReport(path string, data Dataset) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	executions := make([]Execution, 0, len(data.Executions))

	for _, execution := range data.Executions {
		executions = append(executions, execution)
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdatedAt.Before(executions[j].UpdatedAt)
	})

	fmt.Fprintln(writer, "TRADE POST-PROCESSING REPORT")
	fmt.Fprintln(writer, strings.Repeat("=", 80))
	fmt.Fprintf(writer, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "Opportunities: %d\n", len(data.Opportunities))
	fmt.Fprintf(writer, "Executions:    %d\n", len(data.Executions))
	fmt.Fprintf(writer, "Recoveries:    %d\n", len(data.Recoveries))
	fmt.Fprintln(writer)

	for i, execution := range executions {
		if i > 0 {
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, strings.Repeat("#", 80))
			fmt.Fprintln(writer)
		}

		if err := writeExecutionReport(
			writer,
			execution,
			data,
		); err != nil {
			return err
		}
	}

	return nil
}

func writeExecutionReport(
	writer *bufio.Writer,
	execution Execution,
	data Dataset,
) error {
	opportunity, hasOpportunity := data.Opportunities[execution.OpportunityID]

	recovery, hasRecovery := findRecovery(
		execution,
		data.Recoveries,
	)

	fmt.Fprintf(writer, "EXECUTION %s\n", execution.ID)
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	fmt.Fprintf(writer, "Opportunity: %s\n", execution.OpportunityID)
	fmt.Fprintf(writer, "Recovery:    %s\n", execution.RecoveryID)
	fmt.Fprintf(writer, "State:       %s\n", execution.State)
	fmt.Fprintf(writer, "ActualSteps: %d / %d\n",
		execution.ActualSteps,
		execution.TotalSteps,
	)

	if execution.StartedAt.IsZero() {
		fmt.Fprintln(writer, "StartedAt:   INVALID / ZERO")
	} else {
		fmt.Fprintf(writer, "StartedAt:   %s\n",
			execution.StartedAt.Format(time.RFC3339Nano))
	}

	if execution.CompletedAt != nil {
		fmt.Fprintf(writer, "CompletedAt: %s\n",
			execution.CompletedAt.Format(time.RFC3339Nano))
	}

	fmt.Fprintln(writer)

	// -----------------------------------------------------------------
	// OPPORTUNITY
	// -----------------------------------------------------------------

	if hasOpportunity {
		fmt.Fprintln(writer, "SIMULATION")
		fmt.Fprintln(writer, strings.Repeat("-", 80))

		fmt.Fprintf(writer,
			"RequestedStartAmount: %s\n",
			decimalString(opportunity.RequestedStartAmount),
		)

		fmt.Fprintf(writer,
			"ExecutedStartAmount:  %s\n",
			decimalString(opportunity.ExecutedStartAmount),
		)

		fmt.Fprintf(writer,
			"EndAmount:            %s\n",
			decimalString(opportunity.EndAmount),
		)

		fmt.Fprintf(writer,
			"FillRatio:            %s\n",
			decimalString(opportunity.FillRatio),
		)

		fmt.Fprintf(writer,
			"ProfitAbs:            %s\n",
			decimalString(opportunity.ProfitAbs),
		)

		fmt.Fprintf(writer,
			"ProfitPct:            %s\n",
			decimalString(opportunity.ProfitPct),
		)

		fmt.Fprintln(writer)
	}

	// -----------------------------------------------------------------
	// INITIAL STATE
	// -----------------------------------------------------------------

	fmt.Fprintln(writer, "EXECUTION INITIAL STATE")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	writeBalances(writer, execution.InitialState)

	startUSDT := balanceOf(
		execution.InitialState,
		"USDT",
	)

	fmt.Fprintf(writer,
		"Initial USDT: %s\n",
		decimalString(startUSDT),
	)

	fmt.Fprintln(writer)

	// -----------------------------------------------------------------
	// EXECUTION STEPS
	// -----------------------------------------------------------------

	fmt.Fprintln(writer, "EXECUTION STEPS")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	var actuallyUsedStartAsset decimal.Decimal

	for _, step := range execution.Steps {
		fmt.Fprintf(writer, "Step %d\n", step.StepID)

		fmt.Fprintf(writer,
			"  Planned input:   %s %s\n",
			decimalString(step.PlannedInput.Quantity),
			step.PlannedInput.Asset,
		)

		fmt.Fprintf(writer,
			"  Requested input: %s %s\n",
			decimalString(step.RequestedInput.Quantity),
			step.RequestedInput.Asset,
		)

		fmt.Fprintf(writer,
			"  Actual input:    %s %s\n",
			decimalString(step.ActualInput.Quantity),
			step.ActualInput.Asset,
		)

		fmt.Fprintf(writer,
			"  Gross output:    %s %s\n",
			decimalString(step.GrossOutput.Quantity),
			step.GrossOutput.Asset,
		)

		fmt.Fprintf(writer,
			"  Fee:             %s %s\n",
			decimalString(step.Fee.Quantity),
			step.Fee.Asset,
		)

		fmt.Fprintf(writer,
			"  Net output:      %s %s\n",
			decimalString(step.NetOutput.Quantity),
			step.NetOutput.Asset,
		)

		fmt.Fprintf(writer,
			"  Fill ratio:      %s\n",
			decimalString(step.FillRatio),
		)

		fmt.Fprintf(writer,
			"  Result:          %s\n",
			step.Result,
		)

		if step.Error != "" {
			fmt.Fprintf(writer,
				"  Error:           %s\n",
				step.Error,
			)
		}

		if !step.StartedAt.IsZero() &&
			!step.CompletedAt.IsZero() {
			duration := step.CompletedAt.Sub(step.StartedAt)

			fmt.Fprintf(writer,
				"  Duration:        %s\n",
				duration,
			)
		} else {
			fmt.Fprintln(writer,
				"  Duration:        INVALID / MISSING TIMESTAMP",
			)
		}

		// The first actual input is the real amount of the starting
		// asset that was sent into the first execution order.
		if step.StepID == 0 {
			actuallyUsedStartAsset = step.ActualInput.Quantity
		}

		fmt.Fprintln(writer)
	}

	fmt.Fprintln(writer, "ACTUAL START-ASSET USAGE")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	fmt.Fprintf(writer,
		"Initial USDT:        %s\n",
		decimalString(startUSDT),
	)

	fmt.Fprintf(writer,
		"Actually used USDT:  %s\n",
		decimalString(actuallyUsedStartAsset),
	)

	remainingUSDT := startUSDT.Sub(actuallyUsedStartAsset)

	fmt.Fprintf(writer,
		"USDT not sent:       %s\n",
		decimalString(remainingUSDT),
	)

	if !startUSDT.IsZero() {
		usedPct := actuallyUsedStartAsset.
			Div(startUSDT).
			Mul(decimal.NewFromInt(100))

		fmt.Fprintf(writer,
			"Used:                %s%%\n",
			decimalString(usedPct),
		)
	}

	fmt.Fprintln(writer)

	// -----------------------------------------------------------------
	// RECOVERY
	// -----------------------------------------------------------------

	if hasRecovery {
		writeRecoveryReport(writer, recovery)
	} else {
		fmt.Fprintln(writer, "RECOVERY")
		fmt.Fprintln(writer, strings.Repeat("-", 80))
		fmt.Fprintln(writer, "No recovery record found.")
		fmt.Fprintln(writer)
	}

	// -----------------------------------------------------------------
	// FINAL STATE
	// -----------------------------------------------------------------

	fmt.Fprintln(writer, "FINAL EXECUTION STATE")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	writeBalances(writer, execution.ActualState)

	finalUSDT := balanceOf(
		execution.ActualState,
		"USDT",
	)

	fmt.Fprintln(writer)
	fmt.Fprintf(writer,
		"Initial USDT: %s\n",
		decimalString(startUSDT),
	)

	fmt.Fprintf(writer,
		"Final USDT:   %s\n",
		decimalString(finalUSDT),
	)

	usdtPnL := finalUSDT.Sub(startUSDT)

	fmt.Fprintf(writer,
		"USDT PnL:     %s\n",
		decimalString(usdtPnL),
	)

	if !startUSDT.IsZero() {
		usdtPnLPct := usdtPnL.
			Div(startUSDT).
			Mul(decimal.NewFromInt(100))

		fmt.Fprintf(writer,
			"USDT PnL %%:   %s%%\n",
			decimalString(usdtPnLPct),
		)
	}

	fmt.Fprintln(writer)

	// -----------------------------------------------------------------
	// FEES
	// -----------------------------------------------------------------

	writeFees(writer, execution)

	// -----------------------------------------------------------------
	// DIAGNOSTICS
	// -----------------------------------------------------------------

	fmt.Fprintln(writer, "DIAGNOSTICS")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	if execution.StartedAt.IsZero() {
		fmt.Fprintln(writer,
			"WARNING: Execution.StartedAt is zero.",
		)
	}

	if hasRecovery &&
		execution.RecoveryID != "" &&
		execution.RecoveryID != recovery.ID {
		fmt.Fprintf(writer,
			"WARNING: Execution RecoveryID=%s, but linked Recovery ID=%s.\n",
			execution.RecoveryID,
			recovery.ID,
		)
	}

	if hasRecovery {
		if recovery.PlansCreated != recovery.PlansExecuted {
			fmt.Fprintf(writer,
				"WARNING: Recovery plans created=%d, executed=%d.\n",
				recovery.PlansCreated,
				recovery.PlansExecuted,
			)
		}

		if recovery.ProcessState != "Completed" {
			fmt.Fprintf(writer,
				"Recovery final state: %s\n",
				recovery.ProcessState,
			)
		}
	}

	if execution.Error != "" {
		fmt.Fprintf(writer,
			"Execution error: %s\n",
			execution.Error,
		)
	}

	fmt.Fprintln(writer)

	return nil
}

func writeRecoveryReport(
	writer *bufio.Writer,
	recovery Recovery,
) {
	fmt.Fprintln(writer, "RECOVERY")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	fmt.Fprintf(writer, "ID:            %s\n", recovery.ID)
	fmt.Fprintf(writer, "ExecutionID:   %s\n", recovery.ExecutionID)
	fmt.Fprintf(writer, "ProcessState:  %s\n", recovery.ProcessState)
	fmt.Fprintf(writer, "Reason:        %s\n", recovery.Reason)
	fmt.Fprintf(writer, "PlansCreated:  %d\n", recovery.PlansCreated)
	fmt.Fprintf(writer, "PlansExecuted: %d\n", recovery.PlansExecuted)

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Recovery initial state:")

	writeRecoveryBalances(writer, recovery.InitialState)

	fmt.Fprintln(writer)

	for _, attempt := range recovery.Attempts {
		fmt.Fprintf(writer,
			"Attempt %d\n",
			attempt.Index,
		)

		fmt.Fprintf(writer,
			"  Plans created:   %d\n",
			attempt.PlansCreated,
		)

		fmt.Fprintf(writer,
			"  Plans attempted: %d\n",
			attempt.PlansAttempted,
		)

		fmt.Fprintf(writer,
			"  Plans succeeded: %d\n",
			attempt.PlansSucceeded,
		)

		fmt.Fprintf(writer,
			"  Plans failed:    %d\n",
			attempt.PlansFailed,
		)

		fmt.Fprintf(writer,
			"  Changed:         %t\n",
			attempt.Changed,
		)

		if !attempt.StartedAt.IsZero() &&
			attempt.CompletedAt != nil {
			fmt.Fprintf(writer,
				"  Duration:        %s\n",
				attempt.CompletedAt.Sub(attempt.StartedAt),
			)
		} else {
			fmt.Fprintln(writer,
				"  Duration:        INVALID / MISSING TIMESTAMP",
			)
		}

		fmt.Fprintln(writer, "  State before:")
		writeRecoveryBalancesIndented(writer, attempt.StateBefore, "    ")

		fmt.Fprintln(writer, "  State after:")
		writeRecoveryBalancesIndented(writer, attempt.StateAfter, "    ")
		if attempt.Error != "" {
			fmt.Fprintf(writer,
				"  Error: %s\n",
				attempt.Error,
			)
		}

		fmt.Fprintln(writer)
	}

	fmt.Fprintln(writer, "Recovery final state:")
	writeRecoveryBalances(writer, recovery.ActualState)

	if recovery.Error != "" {
		fmt.Fprintf(writer,
			"Recovery error: %s\n",
			recovery.Error,
		)
	}

	fmt.Fprintln(writer)
}

func writeFees(
	writer *bufio.Writer,
	execution Execution,
) {
	fmt.Fprintln(writer, "FEES")
	fmt.Fprintln(writer, strings.Repeat("-", 80))

	fees := make(map[string]decimal.Decimal)

	for _, step := range execution.Steps {
		if step.Fee.Asset == "" {
			continue
		}

		fees[step.Fee.Asset] =
			fees[step.Fee.Asset].Add(step.Fee.Quantity)
	}

	if len(fees) == 0 {
		fmt.Fprintln(writer, "No execution fees recorded.")
		fmt.Fprintln(writer)
		return
	}

	assets := make([]string, 0, len(fees))

	for asset := range fees {
		assets = append(assets, asset)
	}

	sort.Strings(assets)

	for _, asset := range assets {
		fmt.Fprintf(writer,
			"%s: %s\n",
			asset,
			decimalString(fees[asset]),
		)
	}

	fmt.Fprintln(writer)
	fmt.Fprintln(writer,
		"NOTE: Fees in non-USDT assets are not converted to USDT.",
	)
	fmt.Fprintln(writer,
		"A price at execution time is required for an exact USDT equivalent.",
	)
	fmt.Fprintln(writer)
}

func writeBalances(
	writer *bufio.Writer,
	balances []BalanceSnapshot,
) {
	sorted := make([]BalanceSnapshot, len(balances))
	copy(sorted, balances)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Asset < sorted[j].Asset
	})

	for _, balance := range sorted {
		fmt.Fprintf(
			writer,
			"%s: %s\n",
			balance.Asset,
			decimalString(balance.Quantity),
		)
	}
}

func writeRecoveryBalances(
	writer *bufio.Writer,
	balances RecoveryState,
) {
	assets := make([]string, 0, len(balances))

	for asset := range balances {
		assets = append(assets, asset)
	}

	sort.Strings(assets)

	for _, asset := range assets {
		fmt.Fprintf(
			writer,
			"%s: %s\n",
			asset,
			decimalString(balances[asset]),
		)
	}
}

func writeBalancesIndented(
	writer *bufio.Writer,
	balances []BalanceSnapshot,
	indent string,
) {
	sorted := make([]BalanceSnapshot, len(balances))
	copy(sorted, balances)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Asset < sorted[j].Asset
	})

	for _, balance := range sorted {
		fmt.Fprintf(writer,
			"%s%s: %s\n",
			indent,
			balance.Asset,
			decimalString(balance.Quantity),
		)
	}
}

func writeRecoveryBalancesIndented(
	writer *bufio.Writer,
	balances RecoveryState,
	indent string,
) {
	assets := make([]string, 0, len(balances))

	for asset := range balances {
		assets = append(assets, asset)
	}

	sort.Strings(assets)

	for _, asset := range assets {
		fmt.Fprintf(
			writer,
			"%s%s: %s\n",
			indent,
			asset,
			decimalString(balances[asset]),
		)
	}
}

func balanceOf(
	balances []BalanceSnapshot,
	asset string,
) decimal.Decimal {
	for _, balance := range balances {
		if balance.Asset == asset {
			return balance.Quantity
		}
	}

	return decimal.Zero
}

func findRecovery(
	execution Execution,
	recoveries map[string]Recovery,
) (Recovery, bool) {
	// Prefer ExecutionID because the provided data already showed
	// that RecoveryID can become inconsistent.
	for _, recovery := range recoveries {
		if recovery.ExecutionID == execution.ID {
			return recovery, true
		}
	}

	if execution.RecoveryID != "" {
		recovery, ok := recoveries[execution.RecoveryID]
		if ok {
			return recovery, true
		}
	}

	return Recovery{}, false
}

func decimalString(value decimal.Decimal) string {
	if value.IsZero() {
		return "0"
	}

	// Avoid scientific notation in the report.
	value = value.Truncate(18)

	result := value.String()

	// Defensive cleanup.
	if strings.Contains(result, "e") ||
		strings.Contains(result, "E") {
		return fmt.Sprintf("%.18f", value.InexactFloat64())
	}

	// Remove insignificant trailing zeroes.
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")

	if result == "-0" {
		return "0"
	}

	return result
}

// Keep math imported intentionally for future aggregate statistics.
// This also provides a simple sanity helper if the analyzer grows.
func _unusedMathReference() float64 {
	return math.Abs(0)
}
