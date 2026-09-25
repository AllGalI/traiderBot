package domain

import (
	"fmt"
	"strings"

	executionApplication "traiderBot/internalDDD/execution/application"
)

type ExecutionProvider struct {
	store executionApplication.ExecutionStore
}

func NewExecutionProvider(
	store executionApplication.ExecutionStore,
) *ExecutionProvider {

	return &ExecutionProvider{
		store: store,
	}
}

func (p *ExecutionProvider) Name() string {
	return "ExecutionProvider"
}

func (p *ExecutionProvider) Snapshot() (string, error) {

	active := p.store.Active()

	var builder strings.Builder

	if len(active) == 0 {

		return "No active executions\n", nil
	}

	fmt.Fprintln(
		&builder,
		"Active Executions",
	)

	fmt.Fprintln(
		&builder,
		"-----------------",
	)

	for _, execution := range active {

		fmt.Fprintf(
			&builder,
			"ID: %s\n",
			execution.ID,
		)

		fmt.Fprintf(
			&builder,
			"Opportunity: %s\n",
			execution.OpportunityID,
		)

		fmt.Fprintf(
			&builder,
			"State: %s\n",
			execution.State,
		)

		fmt.Fprintf(
			&builder,
			"Progress: %d/%d\n",
			execution.CurrentStep,
			execution.TotalSteps,
		)

		fmt.Fprintf(
			&builder,
			"Updated: %s\n",
			execution.UpdatedAt,
		)

		if execution.Error != "" {

			fmt.Fprintf(
				&builder,
				"Error: %s\n",
				execution.Error,
			)
		}

		fmt.Fprintln(
			&builder,
		)
	}

	return builder.String(), nil
}

func (p *ExecutionProvider) Metrics() (any, error) {

	active := p.store.Active()

	history := p.store.List(1000)

	response := map[string]any{

		"active": len(active),

		"stored": len(history),
	}

	if len(history) == 0 {
		return response, nil
	}

	states := make(
		map[string]int,
	)

	var completed int
	var failed int

	for _, execution := range history {

		states[string(execution.State)]++

		if execution.Error != "" {
			failed++
		}

		if execution.CompletedAt != nil {
			completed++
		}
	}

	response["states"] = states

	response["completed"] = completed

	response["failed"] = failed

	latest, ok := p.store.Latest()

	if ok {

		response["latestID"] =
			latest.ID

		response["latestState"] =
			latest.State
	}

	return response, nil
}
