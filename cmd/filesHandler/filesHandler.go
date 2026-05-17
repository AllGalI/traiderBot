package filesHandler

import (
	"encoding/json"
	"os"
)

var (
	pairsFile = "pairs.json"
)

func ReadSymbols() ([]string, error) {
	file, err := os.ReadFile(pairsFile)
	if err != nil {
		return nil, err
	}

	var ticks []string
	if err := json.Unmarshal(file, &ticks); err != nil {
		return nil, err
	}

	return ticks, nil
}

func SaveSymbols(data any) error {
	pretty, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(pairsFile, pretty, 0644)
	if err != nil {
		return err
	}

	return nil
}
