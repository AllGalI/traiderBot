package logReader

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Определение ваших структур (добавлены базовые поля для демонстрации)
type SymbolInfo struct {
	Base  string
	Quote string
	Name  string
	BoolV bool
}

type Route struct {
	Symbols []SymbolInfo
}

type StepResult struct {
	Symbol   SymbolInfo
	Side     string
	Price    float64
	Amount   float64
	Total    float64
	Result   float64
	Asset    string
	InValue  float64
	OutValue float64
}

type Opportunity struct {
	Route       Route
	Steps       []StepResult
	StartAsset  string
	StartAmount float64
	EndAmount   float64
	ProfitAbs   float64
	ProfitPct   float64
	Timestamp   time.Time
}

// ParseOpportunityLog принимает строку лога и возвращает заполненную структуру
func ParseOpportunityLog(logLine string) (*Opportunity, error) {
	// Регулярное выражение находит всё, что находится внутри крайних фигурных скобок &{...}
	re := regexp.MustCompile(`&\{(.+)\}\s*$`)
	matches := re.FindStringSubmatch(logLine)
	if len(matches) < 2 {
		return nil, fmt.Errorf("объект Opportunity не найден в строке лога")
	}

	content := matches[1]

	// Разделяем контент по срезам [...] и финальным одиночным полям
	// Срезы в логах Go выводятся как [{...} {...}]
	sliceRe := regexp.MustCompile(`\[.*?\]`)
	slices := sliceRe.FindAllString(content, -2)
	if len(slices) < 2 {
		return nil, fmt.Errorf("не удалось найти срезы Route и Steps")
	}

	opp := &Opportunity{}

	// 1. Парсим Route (первый срез)
	routeStr := strings.Trim(slices[0], "[]")
	opp.Route = parseRoute(routeStr)

	// 2. Парсим Steps (второй срез)
	stepsStr := strings.Trim(slices[1], "[]")
	opp.Steps = parseSteps(stepsStr)

	// 3. Парсим оставшиеся одиночные поля после второго среза
	// Находим индекс окончания второго среза в общей строке
	lastSliceIdx := strings.Index(content, slices[1]) + len(slices[1])
	tail := strings.TrimSpace(content[lastSliceIdx:])

	// Разделяем хвост по пробелам, учитывая, что дата содержит пробелы
	// Пример хвоста: BNB 0.15 0.150111 0.000111 0.000741 2026-07-29 22:47:50.507811335 +0300 MSK m=+6913.288950280
	parts := strings.Fields(tail)
	if len(parts) < 9 {
		return nil, fmt.Errorf("неверный формат простых полей в хвосте лога")
	}

	opp.StartAsset = parts[0]
	opp.StartAmount, _ = strconv.ParseFloat(parts[1], 64)
	opp.EndAmount, _ = strconv.ParseFloat(parts[2], 64)
	opp.ProfitAbs, _ = strconv.ParseFloat(parts[3], 64)
	opp.ProfitPct, _ = strconv.ParseFloat(parts[4], 64)

	// Собираем обратно строку времени (из-за монотонного времени m=+... берем 5 элементов)
	timeStr := strings.Join(parts[5:10], " ")
	// Отрезаем монотонную часть m=+..., так как стандартный time.Parse её не поддерживает
	if idx := strings.Index(timeStr, " m="); idx != -1 {
		timeStr = timeStr[:idx]
	}

	// Парсим время по стандартному шаблону Go
	t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга времени: %v", err)
	}
	opp.Timestamp = t

	return opp, nil
}

// Вспомогательная функция для парсинга Route
func parseRoute(str string) Route {
	var r Route
	// Находим элементы {BNB TRX TRXBNB true}
	itemRe := regexp.MustCompile(`\{([^\}]+)\}`)
	matches := itemRe.FindAllStringSubmatch(str, -1)

	for _, m := range matches {
		fields := strings.Fields(m[1])
		if len(fields) >= 4 {
			r.Symbols = append(r.Symbols, SymbolInfo{
				Base:  fields[0],
				Quote: fields[1],
				Name:  fields[2],
				BoolV: fields[3] == "true",
			})
		}
	}
	return r
}

// Вспомогательная функция для парсинга Steps
func parseSteps(str string) []StepResult {
	var steps []StepResult
	// Вложенная структура имеет вид: {{BNB TRX TRXBNB true} BUY 0.00057 262 ...}
	itemRe := regexp.MustCompile(`\{\{([^\}]+)\}\s+([^\}]+)\}`)
	matches := itemRe.FindAllStringSubmatch(str, -1)

	for _, m := range matches {
		// Парсим внутренний SymbolInfo
		symFields := strings.Fields(m[1])
		if len(symFields) < 4 {
			continue
		}
		sym := SymbolInfo{
			Base:  symFields[0],
			Quote: symFields[1],
			Name:  symFields[2],
			BoolV: symFields[3] == "true",
		}

		// Парсим остальные поля шага
		stepFields := strings.Fields(m[2])
		if len(stepFields) < 8 {
			continue
		}

		price, _ := strconv.ParseFloat(stepFields[1], 64)
		amount, _ := strconv.ParseFloat(stepFields[2], 64)
		total, _ := strconv.ParseFloat(stepFields[3], 64)
		result, _ := strconv.ParseFloat(stepFields[4], 64)
		inVal, _ := strconv.ParseFloat(stepFields[6], 64)
		outVal, _ := strconv.ParseFloat(stepFields[7], 64)

		steps = append(steps, StepResult{
			Symbol:   sym,
			Side:     stepFields[0],
			Price:    price,
			Amount:   amount,
			Total:    total,
			Result:   result,
			Asset:    stepFields[5],
			InValue:  inVal,
			OutValue: outVal,
		})
	}
	return steps
}

// ReadLogFileByLines открывает .log файл и построчно парсит объекты Opportunity
func ReadLogFileByLines(filePath string) ([]*Opportunity, error) {
	// 1. Открываем файл для чтения
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}
	// Обязательно закрываем файл после завершения работы функции
	defer file.Close()

	var opportunities []*Opportunity
	lineCount := 0

	// 2. Создаем сканер для построчного чтения
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Пропускаем пустые строки, если они есть в логах
		if len(line) == 0 {
			continue
		}

		// 3. Вызываем функцию парсинга для текущей строки
		opp, err := ParseOpportunityLog(line)
		if err != nil {
			// Выводим ошибку в консоль, но продолжаем читать файл дальше
			fmt.Printf("[Строка %d] Ошибка парсинга: %v\n", lineCount, err)
			continue
		}

		// Если строка успешно распарсена, добавляем объект в итоговый срез
		opportunities = append(opportunities, opp)
	}

	// 4. Проверяем, не возникло ли ошибок во время самого сканирования файла
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла на строке %d: %w", lineCount, err)
	}

	return opportunities, nil
}

// CalculateCumulativeProfit принимает список возможностей и суммирует доход по каждой стартовой монете
func CalculateCumulativeProfit(opportunities []*Opportunity) map[string]float64 {
	// Инициализируем мапу для хранения совокупного дохода
	// Ключ: название монеты (например, "BNB"), Значение: сумма ProfitAbs
	profitMap := make(map[string]float64)

	for _, opp := range opportunities {
		// Игнорируем nil-объекты, если они попали в срез
		if opp == nil {
			continue
		}

		// Пропускаем операции, если название монеты пустое
		if opp.StartAsset == "" {
			continue
		}

		// Прибавляем абсолютный профит к текущему значению в мапе для этой монеты.
		// В Go, если ключа еще нет в мапе, вернется дефолтный 0.0, так что проверка не нужна.
		profitMap[opp.StartAsset] += opp.ProfitAbs
	}

	return profitMap
}

func CheckCummulativeProfit(logFilePath string) {
	// Путь к вашему файлу логов
	if logFilePath == "" {
		logFilePath = "app.log"
	}

	fmt.Printf("Начинаем чтение файла: %s\n", logFilePath)

	mockResults, err := ReadLogFileByLines(logFilePath)
	fmt.Println(len(mockResults))
	if err != nil {
		fmt.Printf("Критическая ошибка: %v\n", err)
		return
	}

	totalProfit := CalculateCumulativeProfit(mockResults)

	fmt.Println("\n--- Совокупный доход по монетам ---")
	for asset, profit := range totalProfit {
		// Форматируем вывод флагом %f для сохранения точности крипты
		fmt.Printf("Монета: %-6s | Общий профит: %.8f\n", asset, profit)
	}

}
