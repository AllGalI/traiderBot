package application

import recoveryDomain "traiderBot/internalDDD/recovery/domain"

func CloneState(
	state recoveryDomain.State,
) recoveryDomain.State {
	result := make(
		recoveryDomain.State,
		len(state),
	)

	for asset, amount := range state {
		result[asset] = amount
	}

	return result
}