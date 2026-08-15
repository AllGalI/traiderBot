package freshness

import (
	"math"
	"sort"
	"sync"
	"time"
)

// =====================================================================
// СТРУКТУРЫ (декларации)
// =====================================================================

// UpdateEvent — событие обновления стакана по символу.
// Вызывающая сторона передаёт его при каждом апдейте книги (даже если
// цена не изменилась — важен сам факт "тика" для расчёта интервалов).
type UpdateEvent struct {
	Symbol    string
	Timestamp time.Time
	SeqID     uint64 // sequence/update id биржи, 0 если не поддерживается
}

// HeartbeatEvent — сигнал живости соединения по символу: любое сообщение
// от биржи (ping/pong, любой другой канал этого же коннекшна и т.п.).
// Нужен, чтобы отличать "рынок молчит" от "сокет умер".
type HeartbeatEvent struct {
	Symbol    string
	Timestamp time.Time
}

// FreshnessConfig — параметры порогов и поведения трекера.
type FreshnessConfig struct {
	WindowSize  int // сколько последних интервалов хранить на символ
	RecalcEvery int // пересчитывать перцентили раз в N апдейтов (экономия CPU)

	MinFloor    time.Duration // абсолютный нижний предел порога свежести
	PercentileK float64       // множитель над p99 интервала (k * p99)

	MaxHeartbeatGap time.Duration // макс. тишина соединения, после которой считаем его мёртвым

	MinSamples       int           // мин. число сэмплов, после которого доверяем адаптивному порогу
	DefaultThreshold time.Duration // порог-заглушка, пока сэмплов недостаточно

	EnableSeqGapCheck bool   // проверять разрывы sequence id
	SeqIncrement      uint64 // ожидаемый шаг seq id (обычно 1)

	// MaxZSpread — максимально допустимая разница между z-score ног
	// одного маршрута (max(z) - min(z)). Если 0 — проверка отключена
	// и остаётся только сравнение каждой ноги с maxZ по отдельности.
	MaxZSpread float64
}

// FreshnessResult — результат проверки свежести одного символа.
type FreshnessResult struct {
	Fresh          bool
	Reason         string
	Age            time.Duration
	Threshold      time.Duration
	ZScore         float64 // age / typical(p99) интервал — для кросс-ноговой нормализации
	ConnectionDead bool
	SeqGapDetected bool
	SampleCount    int
}

// ringBuffer — кольцевой буфер интервалов между апдейтами (в мс),
// на основе которого считаются перцентили.
type ringBuffer struct {
	data []float64
	pos  int
	full bool
}

// SymbolStats — статистика и состояние по одному символу.
type SymbolStats struct {
	mu sync.Mutex

	lastUpdateTime time.Time
	lastSeqID      uint64
	seqInitialized bool
	seqGapDetected bool // латчится при обнаружении разрыва, снимается только через MarkResynced

	lastHeartbeat time.Time

	intervals *ringBuffer

	cachedP95          float64
	cachedP99          float64
	cacheDirty         bool
	updatesSinceRecalc int
	recalcEvery        int

	// Поля для холодного старта из исторической статистики (см. SeedFromHistory).
	seeded      bool
	seedSamples int
}

// FreshnessTracker — трекер по всем символам.
type FreshnessTracker struct {
	mu      sync.RWMutex
	symbols map[string]*SymbolStats
	cfg     FreshnessConfig
}

// =====================================================================
// ЛОГИКА: сбор статистики
// =====================================================================

func newRingBuffer(size int) *ringBuffer {
	if size <= 0 {
		size = 1
	}
	return &ringBuffer{data: make([]float64, size)}
}

func (r *ringBuffer) push(v float64) {
	r.data[r.pos] = v
	r.pos = (r.pos + 1) % len(r.data)
	if r.pos == 0 {
		r.full = true
	}
}

func (r *ringBuffer) snapshot() []float64 {
	if r.full {
		out := make([]float64, len(r.data))
		copy(out, r.data)
		return out
	}
	out := make([]float64, r.pos)
	copy(out, r.data[:r.pos])
	return out
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := p / 100 * float64(len(sorted)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func newSymbolStats(windowSize, recalcEvery int) *SymbolStats {
	if recalcEvery <= 0 {
		recalcEvery = 20
	}
	return &SymbolStats{
		intervals:   newRingBuffer(windowSize),
		cacheDirty:  true,
		recalcEvery: recalcEvery,
	}
}

// recalcLocked пересчитывает p95/p99 по текущему окну.
// Вызывается под уже захваченным s.mu.
func (s *SymbolStats) recalcLocked() {
	data := s.intervals.snapshot()
	if len(data) == 0 {
		return
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	s.cachedP95 = percentile(sorted, 95)
	s.cachedP99 = percentile(sorted, 99)
	s.cacheDirty = false
	s.updatesSinceRecalc = 0
}

func (s *SymbolStats) sampleCount() int {
	n := len(s.intervals.snapshot())
	// Пока реальных интервалов меньше, чем "посеяно" из истории — считаем,
	// что сэмплов достаточно (seedSamples), чтобы не проваливаться в
	// DefaultThreshold на каждом рестарте процесса.
	if s.seeded && n < s.seedSamples {
		return s.seedSamples
	}
	return n
}

// NewFreshnessTracker создаёт трекер с заданной конфигурацией.
func NewFreshnessTracker(cfg FreshnessConfig) *FreshnessTracker {
	if cfg.SeqIncrement == 0 {
		cfg.SeqIncrement = 1
	}
	return &FreshnessTracker{
		symbols: make(map[string]*SymbolStats),
		cfg:     cfg,
	}
}

func (t *FreshnessTracker) getOrCreate(symbol string) *SymbolStats {
	t.mu.RLock()
	s, ok := t.symbols[symbol]
	t.mu.RUnlock()
	if ok {
		return s
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok := t.symbols[symbol]; ok {
		return s
	}
	s = newSymbolStats(t.cfg.WindowSize, t.cfg.RecalcEvery)
	t.symbols[symbol] = s
	return s
}

// RecordUpdate вызывается на каждый апдейт стакана.
// Копит интервалы между апдейтами и (опционально) проверяет разрывы seq id.
func (t *FreshnessTracker) RecordUpdate(ev UpdateEvent) {
	s := t.getOrCreate(ev.Symbol)
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.lastUpdateTime.IsZero() {
		dt := ev.Timestamp.Sub(s.lastUpdateTime).Seconds() * 1000 // мс
		if dt > 0 {
			s.intervals.push(dt)
			s.cacheDirty = true
			s.updatesSinceRecalc++
		}
	}
	s.lastUpdateTime = ev.Timestamp

	// --- Проверка sequence id ---
	// Важно: lastSeqID НЕ откатывается назад на дубликатах/реордеринге,
	// а seqGapDetected латчится (не сбрасывается сам на следующем "ровном" апдейте) —
	// снять флаг можно только явным вызовом MarkResynced после переполучения снапшота.
	if t.cfg.EnableSeqGapCheck && ev.SeqID != 0 {
		if s.seqInitialized {
			switch {
			case ev.SeqID == s.lastSeqID:
				// дубликат, игнорируем полностью
			case ev.SeqID < s.lastSeqID:
				// сообщение "из прошлого" (reorder на сети), состояние не трогаем
			case ev.SeqID != s.lastSeqID+t.cfg.SeqIncrement:
				// реальный разрыв — латчим флаг и продвигаем последовательность вперёд
				s.seqGapDetected = true
				s.lastSeqID = ev.SeqID
			default:
				s.lastSeqID = ev.SeqID
			}
		} else {
			s.lastSeqID = ev.SeqID
			s.seqInitialized = true
		}
	}

	if s.cacheDirty && s.updatesSinceRecalc >= s.recalcEvery {
		s.recalcLocked()
	}
}

// RecordHeartbeat вызывается на ЛЮБОЕ сообщение от соединения,
// несущего данный символ (не только апдейты книги).
func (t *FreshnessTracker) RecordHeartbeat(ev HeartbeatEvent) {
	s := t.getOrCreate(ev.Symbol)
	s.mu.Lock()
	s.lastHeartbeat = ev.Timestamp
	s.mu.Unlock()
}

// MarkResynced снимает латч seqGapDetected для символа. Вызывайте после того,
// как заново получили полный снапшот стакана (REST snapshot / переподписка)
// и восстановили консистентное состояние книги.
func (t *FreshnessTracker) MarkResynced(symbol string) {
	s := t.getOrCreate(symbol)
	s.mu.Lock()
	s.seqGapDetected = false
	s.mu.Unlock()
}

func (t *FreshnessTracker) ResyncConnection() {
	for symbol, _ := range t.symbols {
		t.MarkResynced(symbol)
	}
}

// SeedFromHistory бутстрапит адаптивный порог символа из ранее собранной
// офлайн-статистики (например, вашего дампа p95/p99 за прошлые сутки),
// чтобы сразу после рестарта процесса символ не падал на DefaultThreshold,
// одинаковый для всех пар. Разброс p99 между парами в вашем дампе — от
// 175мс (BTCUSDT) до 48с (ETCBTC), поэтому единый DefaultThreshold — это
// всегда компромисс, а seed решает проблему для каждого символа отдельно.
//
// sampleCount ограничивается cfg.MinSamples — этого достаточно, чтобы сразу
// пройти порог доверия адаптивному порогу. Посев замещается реальными
// данными естественным образом: как только кольцевой буфер наберёт
// RecalcEvery новых интервалов, cachedP95/P99 пересчитаются уже по живым
// данным и посеянные числа больше не будут использоваться напрямую (см.
// recalcLocked — она перезаписывает кэш, если в буфере есть данные).
func (t *FreshnessTracker) SeedFromHistory(symbol string, p95Ms, p99Ms float64, sampleCount int) {
	s := t.getOrCreate(symbol)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cachedP95 = p95Ms
	s.cachedP99 = p99Ms
	s.cacheDirty = false

	if sampleCount > t.cfg.MinSamples {
		sampleCount = t.cfg.MinSamples
	}
	if sampleCount < 0 {
		sampleCount = 0
	}
	s.seedSamples = sampleCount
	s.seeded = true
}

// =====================================================================
// ЛОГИКА: проверка свежести
// =====================================================================

// CheckFreshness проверяет, можно ли доверять текущему состоянию стакана
// символа в момент времени now.
func (t *FreshnessTracker) CheckFreshness(symbol string, now time.Time) FreshnessResult {
	s := t.getOrCreate(symbol)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastUpdateTime.IsZero() {
		return FreshnessResult{Fresh: false, Reason: "нет данных по символу"}
	}

	age := now.Sub(s.lastUpdateTime)

	// 1. Здоровье соединения: если heartbeat давно не приходил — считаем канал мёртвым,
	//    независимо от того, что говорит адаптивный порог.
	if !s.lastHeartbeat.IsZero() {
		gap := now.Sub(s.lastHeartbeat)
		if gap > t.cfg.MaxHeartbeatGap {
			return FreshnessResult{
				Fresh:          false,
				Reason:         "соединение мертво: нет heartbeat",
				Age:            age,
				ConnectionDead: true,
				SeqGapDetected: s.seqGapDetected,
			}
		}
	}

	// 2. Разрыв sequence id — верный признак реальной рассинхронизации,
	//    а не просто редких апдейтов. Флаг залатчен, пока не вызовут MarkResynced.
	if t.cfg.EnableSeqGapCheck && s.seqGapDetected {
		return FreshnessResult{
			Fresh:          false,
			Reason:         "обнаружен разрыв sequence id (требуется ресинк)",
			Age:            age,
			SeqGapDetected: true,
		}
	}

	samples := s.sampleCount()

	// 3. Адаптивный порог: k * p99 собственных интервалов символа, с нижним полом.
	var threshold time.Duration
	if samples < t.cfg.MinSamples {
		threshold = t.cfg.DefaultThreshold
	} else {
		p99 := time.Duration(s.cachedP99 * float64(time.Millisecond))
		threshold = time.Duration(float64(p99) * t.cfg.PercentileK)
		if threshold < t.cfg.MinFloor {
			threshold = t.cfg.MinFloor
		}
	}

	var z float64
	if samples >= t.cfg.MinSamples && s.cachedP99 > 0 {
		z = float64(age.Milliseconds()) / s.cachedP99
	}

	fresh := age <= threshold
	reason := "свежий"
	if !fresh {
		reason = "возраст превышает адаптивный порог"
	}

	return FreshnessResult{
		Fresh:       fresh,
		Reason:      reason,
		Age:         age,
		Threshold:   threshold,
		ZScore:      z,
		SampleCount: samples,
	}
}

// ZScoreForRoute проверяет весь маршрут треугольного арбитража целиком.
//
// Делает два независимых по смыслу вида проверок:
//  1. Абсолютная (по сути дублирует Fresh, но плюс heartbeat/seq-gap) —
//     каждая нога должна быть Fresh, соединение живо, seq без разрывов.
//  2. Относительная — сравнивает z-score ног между собой. Даже если каждая
//     нога по отдельности "свежая" относительно своей истории, одна из них
//     может быть заметно "старее" других прямо сейчас — это и есть тот самый
//     дисбаланс, из-за которого треугольник может закрыться на неактуальной цене.
//     Проверяется через max(z) - min(z) <= cfg.MaxZSpread (если MaxZSpread > 0).
func (t *FreshnessTracker) ZScoreForRoute(symbols []string, now time.Time, maxZ float64) (bool, map[string]FreshnessResult) {
	results := make(map[string]FreshnessResult, len(symbols))
	ok := true

	var minZ, maxZSeen float64
	haveZ := false

	for _, sym := range symbols {
		r := t.CheckFreshness(sym, now)
		results[sym] = r
		if !r.Fresh || r.ConnectionDead || r.SeqGapDetected {
			ok = false
			continue
		}
		if r.SampleCount > 0 {
			if r.ZScore > maxZ {
				ok = false
			}
			if !haveZ {
				minZ, maxZSeen = r.ZScore, r.ZScore
				haveZ = true
			} else {
				if r.ZScore < minZ {
					minZ = r.ZScore
				}
				if r.ZScore > maxZSeen {
					maxZSeen = r.ZScore
				}
			}
		}
	}

	// Относительная проверка: разброс z-score между ногами маршрута.
	if ok && t.cfg.MaxZSpread > 0 && haveZ {
		if (maxZSeen - minZ) > t.cfg.MaxZSpread {
			ok = false
		}
	}

	return ok, results
}