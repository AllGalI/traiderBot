package binance

import (
	"encoding/json"
	"fmt"
)

func decodeJSON[T any](
	body []byte,
) (T, error) {
	var result T

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf(
			"decode binance response: %w",
			err,
		)
	}

	return result, nil
}
