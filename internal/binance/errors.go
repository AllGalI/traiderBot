package binance

import "fmt"

type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e APIError) Error() string {
	return fmt.Sprintf(
		"binance error %d: %s",
		e.Code,
		e.Msg,
	)
}
