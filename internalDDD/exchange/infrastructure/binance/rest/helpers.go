package rest

import (
	"encoding/json"
)


func stringArrayToJSON(
	values []string,
) string {

	data, _ := json.Marshal(values)

	return string(data)
}