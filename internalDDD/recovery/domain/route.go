package domain

import (
	"strings"

	marketDomain "traiderBot/internalDDD/market/domain"
)

type RecoveryRoute struct {
	Edges []marketDomain.GraphEdge
	Key   string
}

func NewRecoveryRoute(
	edges []marketDomain.GraphEdge,
) RecoveryRoute {

	return RecoveryRoute{
		Edges: edges,
		Key:   buildKey(edges),
	}
}


func (r RecoveryRoute) Symbols() []string {

	result := make([]string, 0, len(r.Edges))

	for _, edge := range r.Edges {
		result = append(
			result,
			edge.Symbol,
		)
	}

	return result
}


func (r RecoveryRoute) Steps() int {
	return len(r.Edges)
}


func buildKey(
	edges []marketDomain.GraphEdge,
) string {

	if len(edges) == 0 {
		return ""
	}


	parts := make([]string, 0, len(edges))


	for _, edge := range edges {

		direction := "SELL"

		if edge.Direction == marketDomain.Buy {
			direction = "BUY"
		}


		parts = append(
			parts,
			edge.From+
				":"+
				direction+
				":"+
				edge.To,
		)
	}


	return strings.Join(parts, "|")
}