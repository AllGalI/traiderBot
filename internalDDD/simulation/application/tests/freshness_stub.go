package tests

import (
	"time"
)

type alwaysFreshPolicy struct{}

func (alwaysFreshPolicy) IsFresh(
	symbol string,
	updatedAt time.Time,
	now time.Time,
) bool {
	return true
}