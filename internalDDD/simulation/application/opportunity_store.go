package application

import simulationDomain "traiderBot/internalDDD/simulation/domain"

type OpportunityStore interface {
	Save(
		opportunity *simulationDomain.Opportunity,
	) error

	Latest() (
		*simulationDomain.Opportunity,
		bool,
	)

	List(
		limit int,
	) []simulationDomain.Opportunity

	Count() int
}
