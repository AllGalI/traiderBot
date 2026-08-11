package stats

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

func AppendRowToCSV(pathToFolder, filename string, values ...any) error {
	if err := os.MkdirAll(pathToFolder, 0755); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	filePath := filepath.Join(pathToFolder, filename)

	file, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat csv file: %w", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Если файл новый — записываем header.
	if info.Size() == 0 {
		header := []string{
			"time",
			"symbol_checks_counter",
			"old_symbols_count",
			"total_simulations_started",
			"dubs_check_counter",
			"unfresh_routes_counter",
			"fresh_routes_counter",
			"wrong_start_amount",
			"ok_start_amount_counter",
			"total_simulations",
			"top_percent",
			"average_percent",
		}

		if err := writer.Write(header); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
	}

	// Преобразуем любые типы в string.
	dataToWrite := make([]string, 0, len(values))

	for _, value := range values {
		dataToWrite = append(dataToWrite, fmt.Sprint(value))
	}

	if err := writer.Write(dataToWrite); err != nil {
		return fmt.Errorf("write csv row: %w", err)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}
