package domain

import (
	"fmt"
	"strings"
	"time"

	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
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
		"Reason: %s\n",
		recovery.Reason,
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

	fmt.Fprintln(&builder)

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

		fmt.Fprintf(
			&builder,
			"Duration: %s\n",
			recovery.CompletedAt.Sub(
				recovery.StartedAt,
			),
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

	var (
		completed int
		failed    int

		totalAttempts int
		totalDuration time.Duration

		totalPlansCreated   int
		totalPlansAttempted int
		totalPlansSucceeded int
		totalPlansFailed    int

		noProgressAttempts int

		reasons = map[string]int{}
	)

	for _, recovery := range recoveries {

		reasons[string(recovery.Reason)]++

		totalAttempts += len(recovery.Attempts)

		switch recovery.ProcessState {

		case recoveryDomain.RecoveryCompleted:
			completed++

		case recoveryDomain.RecoveryFailed:
			failed++
		}

		if recovery.CompletedAt != nil {

			totalDuration +=
				recovery.CompletedAt.Sub(
					recovery.StartedAt,
				)
		}

		for _, attempt := range recovery.Attempts {

			totalPlansCreated +=
				attempt.PlansCreated

			totalPlansAttempted +=
				attempt.PlansAttempted

			totalPlansSucceeded +=
				attempt.PlansSucceeded

			totalPlansFailed +=
				attempt.PlansFailed

			if !attempt.Changed {
				noProgressAttempts++
			}
		}
	}

	response["completed"] = completed
	response["failed"] = failed

	response["averageAttempts"] =
		float64(totalAttempts) /
			float64(len(recoveries))

	response["averageDuration"] =
		totalDuration /
			time.Duration(len(recoveries))

	response["plansCreated"] =
		totalPlansCreated

	response["plansAttempted"] =
		totalPlansAttempted

	response["plansSucceeded"] =
		totalPlansSucceeded

	response["plansFailed"] =
		totalPlansFailed

	response["noProgressAttempts"] =
		noProgressAttempts

	response["reasons"] = reasons

	latest := recoveries[len(recoveries)-1]

	response["latestID"] = latest.ID
	response["latestState"] = latest.ProcessState
	response["latestReason"] = latest.Reason

	response["latestAttempts"] = len(
		latest.Attempts,
	)

	response["latestPlansCreated"] =
		latest.PlansCreated

	response["latestPlansExecuted"] =
		latest.PlansExecuted

	return response, nil
}

func printAttempt(
	builder *strings.Builder,
	attempt recoveryDomain.RecoveryAttemptSnapshot,
) {

	fmt.Fprintf(
		builder,
		"\nAttempt %d\n",
		attempt.Index,
	)

	fmt.Fprintln(
		builder,
		"---------",
	)

	fmt.Fprintf(
		builder,
		"Plans created: %d\n",
		attempt.PlansCreated,
	)

	fmt.Fprintf(
		builder,
		"Plans attempted: %d\n",
		attempt.PlansAttempted,
	)

	fmt.Fprintf(
		builder,
		"Plans succeeded: %d\n",
		attempt.PlansSucceeded,
	)

	fmt.Fprintf(
		builder,
		"Plans failed: %d\n",
		attempt.PlansFailed,
	)

	fmt.Fprintf(
		builder,
		"Changed: %t\n",
		attempt.Changed,
	)

	if !attempt.StartedAt.IsZero() {

		fmt.Fprintf(
			builder,
			"Started: %s\n",
			attempt.StartedAt,
		)
	}

	if attempt.CompletedAt != nil {

		fmt.Fprintf(
			builder,
			"Completed: %s\n",
			*attempt.CompletedAt,
		)

		fmt.Fprintf(
			builder,
			"Duration: %s\n",
			attempt.CompletedAt.Sub(
				attempt.StartedAt,
			),
		)
	}

	fmt.Fprintln(
		builder,
		"State Before:",
	)

	printState(
		builder,
		attempt.StateBefore,
	)

	fmt.Fprintln(
		builder,
		"State After:",
	)

	printState(
		builder,
		attempt.StateAfter,
	)

	if attempt.Error != "" {

		fmt.Fprintf(
			builder,
			"Error: %s\n",
			attempt.Error,
		)
	}
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