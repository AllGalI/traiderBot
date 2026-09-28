package batchjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const defaultBatchSize = 1000

type Writer struct {
	mu sync.Mutex

	dir       string
	batchSize int

	batchNumber int
	recordCount int
}

func NewWriter(dir string, batchSize int) (*Writer, error) {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create batch directory %s: %w", dir, err)
	}

	writer := &Writer{
		dir:         dir,
		batchSize:   batchSize,
		batchNumber: 1,
	}

	if err := writer.loadLatestBatch(); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *Writer) Append(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal batch record: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.recordCount >= w.batchSize {
		w.batchNumber++
		w.recordCount = 0
	}

	path := filepath.Join(
		w.dir,
		fmt.Sprintf("%06d.json", w.batchNumber),
	)

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open batch %s: %w", path, err)
	}

	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append batch record %s: %w", path, err)
	}

	w.recordCount++

	return nil
}

func (w *Writer) loadLatestBatch() error {
	files, err := os.ReadDir(w.dir)
	if err != nil {
		return fmt.Errorf("read batch directory %s: %w", w.dir, err)
	}

	var batchNumbers []int

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		if filepath.Ext(name) != ".json" {
			continue
		}

		base := strings.TrimSuffix(name, ".json")

		number, err := strconv.Atoi(base)
		if err != nil {
			continue
		}

		batchNumbers = append(batchNumbers, number)
	}

	if len(batchNumbers) == 0 {
		return nil
	}

	sort.Ints(batchNumbers)

	latestNumber := batchNumbers[len(batchNumbers)-1]

	path := filepath.Join(
		w.dir,
		fmt.Sprintf("%06d.json", latestNumber),
	)

	count, err := countRecords(path)
	if err != nil {
		return fmt.Errorf("count records in batch %s: %w", path, err)
	}

	w.batchNumber = latestNumber
	w.recordCount = count

	if w.recordCount >= w.batchSize {
		w.batchNumber++
		w.recordCount = 0
	}

	return nil
}

func countRecords(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Opportunity может содержать достаточно большие Steps.
	// Увеличиваем лимит одной строки.
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	count := 0

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}