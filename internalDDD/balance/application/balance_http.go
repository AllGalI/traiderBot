package application

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type balanceHTTPResponse struct {
	Asset   string                 `json:"asset"`
	Current string                 `json:"current"`
	History []BalanceHistoryPoint `json:"history"`
}

func (s *BalanceService) BalanceHTTPHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	history := s.USDTHistory()

	if hoursValue := r.URL.Query().Get("hours"); hoursValue != "" {
		hours, err := strconv.Atoi(hoursValue)
		if err != nil || hours <= 0 {
			http.Error(
				w,
				"invalid hours",
				http.StatusBadRequest,
			)

			return
		}

		history = s.USDTHistorySince(
			time.Now().Add(
				-time.Duration(hours)*time.Hour,
			),
		)
	}

	response := balanceHTTPResponse{
		Asset:   "USDT",
		Current: s.USDTString(),
		History: history,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}