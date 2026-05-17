package arbitrage

import (
	"errors"
	"traiderBot/cmd/domain"
)

// План расчёта

// 1. Получаем
// 	bids(запросы на покупку/мы им сможем продать)
// 	баланс на покупку основной крипты

// 	asks(запросы на продажу/мы у них можем купить)
// 	баланс на продажу основной крипты
// 2.
// 	узнаём сколько валюты у нас на счетах usdt на покупку, btc на продажу.
// 	сравниваем максимальный bid с минимальным ack(они отсортированны должны быть уже)
// 	Если разница меньше дельта 0.3% пропускаем пару.
// 	Если больше в зависимости от знака будем выбирать биржу где продать, где купить.

// 	После

// 	2 варианта
// 		мы можем делать сделки глубиной в 1 слой и собирать лучшую маржу

// 		или мы можем пытаться весь спред между биржами с учётом комиссии

// 		Тогда сначала будем пробовать получить всю возможную глубину
// 		с учётом комиссии ------------------------------------------------------- ok

/* Теперь думаю о том, что нужно как то выбирать, какая биржа будет для продажи
а какая для покупки. Думаю сделать ещё одну функцию, которая будет делать
быструю проверку. Сравнивать первые значения bids & asks на бирже. Чтобы
уменьшить количество циклов, думаю сделать так, чтобы при проверке учиты-
вался модуль разницы, в зависимости от знака будет выбранна последовательность
при которой будут передаваться acks, bids разных бирж.

Сейчас пользуюсь binance и bybit. Е меня есть информация о биржах, парах, стаканах
комиссии пока что захардкожены. По поводу пар, думаю нужно сохранять их так, чтобы
там лежала инфа не только о названии пары, но и про то какой токен основной, а ка-
кой нет. Нужно будет переделать сохранение пар токенов, а пока что захардкожу в
json, чтобы время не тратить и добавлять буду вручную.

Как будет работать, у меня есть слайс из OrderBook, binance и bybit - последова-
тельность не меняется.

type OrderBook struct {
	Exchange string
	Symbols  []Symbol
}

type Symbol struct {
	Name string
	Data
}

type Data struct {
	LastUpdateId   int64              `json:"lastUpdateId"`
	Bids           map[string]float64 `json:"-"` // price -> quantity
	Asks           map[string]float64 `json:"-"` // price -> quantity
	LastUpdateTime int64              `json:"-"` // Unix Milli
}




backlog

	Так же если бирж будет больше 2, тогда нужно сохранять экстремальные значения
цены покупки и продажи. Чтобы выбрать лучшую возможность арбитража. Но там может
быть зависимость от объема. Пока что забью на это, буду пробовать на 2 биржах.

 Нужно будет переделать сохранение пар токенов, а пока что захардкожу в
json, чтобы время не тратить и добавлять буду вручную.

*/

func Analize(asks, bids [][]float64) (*domain.ArbitrageResult, error) {
	if len(asks) == 0 || len(bids) == 0 {
		return nil, errors.New("empty orderbook")
	}

	const minSpread = 0.003

	i := 0 // ask index
	j := 0 // bid index

	totalVolume := 0.0

	minAskPrice := asks[0][0]
	maxBidPrice := bids[0][0]

	// остатки объёмов текущих уровней
	askVolumeLeft := asks[0][1]
	bidVolumeLeft := bids[0][1]

	found := false

	for i < len(asks) && j < len(bids) {
		askPrice := asks[i][0]
		bidPrice := bids[j][0]

		// проверяем спред
		diff := (bidPrice - askPrice) / askPrice

		if diff < minSpread {
			break
		}

		found = true

		// сколько можем сматчить на этих уровнях
		matchedVolume := min(askVolumeLeft, bidVolumeLeft)

		totalVolume += matchedVolume

		// обновляем экстремальные цены
		if askPrice < minAskPrice {
			minAskPrice = askPrice
		}

		if bidPrice > maxBidPrice {
			maxBidPrice = bidPrice
		}

		// уменьшаем остатки
		askVolumeLeft -= matchedVolume
		bidVolumeLeft -= matchedVolume

		// если ask уровень закончился → следующий ask
		if askVolumeLeft == 0 {
			i++

			if i < len(asks) {
				askVolumeLeft = asks[i][1]
			}
		}

		// если bid уровень закончился → следующий bid
		if bidVolumeLeft == 0 {
			j++

			if j < len(bids) {
				bidVolumeLeft = bids[j][1]
			}
		}
	}

	if !found {
		return nil, errors.New("no arbitrage")
	}

	return &domain.ArbitrageResult{
		Volume:      totalVolume,
		MinAskPrice: minAskPrice,
		MaxBidPrice: maxBidPrice,
	}, nil
}
