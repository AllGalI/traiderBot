package application

import executionDomain "traiderBot/internalDDD/execution/domain"

type ExecutionStore interface {
	Create(
		execution *executionDomain.ExecutionSnapshot,
	) error

	Complete(
		execution *executionDomain.ExecutionSnapshot,
	) error

	Update(
		execution *executionDomain.ExecutionSnapshot,
	) error

	Active() []executionDomain.ExecutionSnapshot

	Latest() (
		*executionDomain.ExecutionSnapshot,
		bool,
	)

	List(
		limit int,
	) []executionDomain.ExecutionSnapshot
}
