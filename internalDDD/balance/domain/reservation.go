package domain

import (
	"github.com/shopspring/decimal"
)


type Reservation struct {

	ID string

	ExecutionID string

	Asset string

	Amount decimal.Decimal

	Status ReservationStatus
}


type ReservationStatus string


const (
	ReservationActive ReservationStatus = "Active"

	ReservationReleased ReservationStatus = "Released"

	ReservationCompleted ReservationStatus = "Completed"
)