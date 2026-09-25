package application

import recoveryDomain "traiderBot/internalDDD/recovery/domain"

type RecoveryStore interface {

	Create(
		recovery *recoveryDomain.RecoverySnapshot,
	) error


	Update(
		recovery *recoveryDomain.RecoverySnapshot,
	) error


	Complete(
		recovery *recoveryDomain.RecoverySnapshot,
	) error


	Active() []recoveryDomain.RecoverySnapshot


	Latest() (
		*recoveryDomain.RecoverySnapshot,
		bool,
	)


	List(
		limit int,
	) []recoveryDomain.RecoverySnapshot
}