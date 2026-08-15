package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// SaveToJSON сериализует любой объект в JSON и сохраняет в файл,
// полностью перезаписывая его, если он уже существует.
type LogData struct {
	Data any       `json:"data"`
	Time time.Time `json:"time"`
}

type FileStructure struct {
	Logs []LogData `json:"logs"`
}

func SaveLog(path string, data any) error {
	var fileStructure FileStructure

	// Пытаемся прочитать существующий файл.
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Файла нет — начинаем с пустой структуры.
		fileStructure = FileStructure{
			Logs: make([]LogData, 0),
		}
	} else {
		// Файл существует — десериализуем его.
		if len(content) > 0 {
			if err := json.Unmarshal(content, &fileStructure); err != nil {
				return fmt.Errorf("failed to decode json from %s: %w", path, err)
			}
		}
	}

	// Добавляем новый лог.
	fileStructure.Logs = append(fileStructure.Logs, LogData{
		Data: data,
		Time: time.Now(),
	})

	// Создаём файл, если его не было, и перезаписываем содержимое.
	f, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	if err := enc.Encode(fileStructure); err != nil {
		return fmt.Errorf("failed to encode json to %s: %w", path, err)
	}

	return nil
}
