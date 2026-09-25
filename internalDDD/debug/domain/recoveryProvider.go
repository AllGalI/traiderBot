package domain

import (
	"fmt"
	"strings"

	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"

	"github.com/shopspring/decimal"
)

type RecoveryProvider struct {
	store recoveryApplication.RecoveryStore
}

func NewRecoveryProvider(
	store recoveryApplication.RecoveryStore,
) *RecoveryProvider {
	return &RecoveryProvider{
		store: store,
	}
}

func (p *RecoveryProvider) Name() string {
	return "RecoveryProvider"
}

func (p *RecoveryProvider) Snapshot() (string, error) {

	recovery, ok := p.store.Latest()

	if !ok {
		return "No recoveries\n", nil
	}

	var builder strings.Builder

	fmt.Fprintln(
		&builder,
		"Latest Recovery",
	)

	fmt.Fprintln(
		&builder,
		"---------------",
	)

	fmt.Fprintf(
		&builder,
		"ID: %s\n",
		recovery.ID,
	)

	fmt.Fprintf(
		&builder,
		"Execution: %s\n",
		recovery.ExecutionID,
	)

	fmt.Fprintf(
		&builder,
		"State: %s\n",
		recovery.ProcessState,
	)

	fmt.Fprintf(
		&builder,
		"Attempts: %d\n",
		recovery.Attempts,
	)

	fmt.Fprintf(
		&builder,
		"Plans created: %d\n",
		recovery.PlansCreated,
	)

	fmt.Fprintf(
		&builder,
		"Plans executed: %d\n",
		recovery.PlansExecuted,
	)

	fmt.Fprintln(
		&builder,
		"",
	)

	fmt.Fprintln(
		&builder,
		"Initial State:",
	)

	printState(
		&builder,
		recovery.InitialState,
	)


	fmt.Fprintln(
		&builder,
		"Actual State:",
	)

	printState(
		&builder,
		recovery.ActualState,
	)


	fmt.Fprintf(
		&builder,
		"Started: %s\n",
		recovery.StartedAt,
	)

	fmt.Fprintf(
		&builder,
		"Updated: %s\n",
		recovery.UpdatedAt,
	)


	if recovery.CompletedAt != nil {

		fmt.Fprintf(
			&builder,
			"Completed: %s\n",
			*recovery.CompletedAt,
		)
	}


	if recovery.Error != "" {

		fmt.Fprintf(
			&builder,
			"Error: %s\n",
			recovery.Error,
		)
	}


	return builder.String(), nil
}


func (p *RecoveryProvider) Metrics() (any, error) {

	recoveries := p.store.List(1000)

	response := map[string]any{
		"stored": len(recoveries),
	}


	if len(recoveries) == 0 {
		return response, nil
	}


	active := 0
	completed := 0
	failed := 0


	var totalAttempts decimal.Decimal


	for _, recovery := range recoveries {

		totalAttempts = totalAttempts.Add(
			decimal.NewFromInt(
				int64(recovery.Attempts),
			),
		)


		switch recovery.ProcessState {

		case recoveryDomain.RecoveryStarted,
			recoveryDomain.RecoveryPlanning,
			recoveryDomain.RecoveryExecuting:

			active++


		case recoveryDomain.RecoveryCompleted:

			completed++


		case recoveryDomain.RecoveryFailed:

			failed++
		}
	}


	response["active"] = active
	response["completed"] = completed
	response["failed"] = failed


	response["averageAttempts"] =
		totalAttempts.Div(
			decimal.NewFromInt(
				int64(len(recoveries)),
			),
		)


	latest := recoveries[len(recoveries)-1]


	response["latestID"] =
		latest.ID


	response["latestState"] =
		latest.ProcessState


	return response, nil
}


func printState(
	builder *strings.Builder,
	state recoveryDomain.State,
) {

	if len(state) == 0 {
		fmt.Fprintln(
			builder,
			"  empty",
		)

		return
	}


	for asset, amount := range state {

		fmt.Fprintf(
			builder,
			"  %s: %s\n",
			asset,
			amount,
		)
	}
}