package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	balanceApplication "traiderBot/internalDDD/balance/application"
	executionFile "traiderBot/internalDDD/execution/infrastructure/file"
	simulationFile "traiderBot/internalDDD/simulation/infrastructure/file"
)

type Bot struct {
	token  string
	chatID int64

	baseURL string

	balanceService    *balanceApplication.BalanceService
	executionStore    *executionFile.ExecutionStore
	opportunityStore  *simulationFile.OpportunityStore

	client *http.Client

	offset int64
}

func NewBot(
	balanceService *balanceApplication.BalanceService,
	executionStore *executionFile.ExecutionStore,
	opportunityStore *simulationFile.OpportunityStore,
) (*Bot, error) {
	token := strings.TrimSpace(
		os.Getenv("TG_BOT_TOKEN"),
	)

	if token == "" {
		return nil, fmt.Errorf(
			"TG_BOT_TOKEN is not configured",
		)
	}

	var chatID int64

	rawChatID := strings.TrimSpace(
		os.Getenv("TG_CHAT_ID"),
	)

	if rawChatID != "" {
		value, err := strconv.ParseInt(rawChatID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid TG_CHAT_ID: %w",
				err,
			)
		}

		chatID = value
	}

	return &Bot{
		token:  token,
		chatID: chatID,

		baseURL: "https://api.telegram.org/bot" + token,

		balanceService:   balanceService,
		executionStore:   executionStore,
		opportunityStore: opportunityStore,

		client: &http.Client{
			Timeout: 35 * time.Second,
		},
	}, nil
}

func (b *Bot) Run(
	ctx context.Context,
) error {
	log.Println("telegram bot starting...")

	if err := b.getMe(ctx); err != nil {
		return fmt.Errorf(
			"telegram getMe: %w",
			err,
		)
	}

	log.Println("telegram bot started")

	for {
		if ctx.Err() != nil {
			log.Println("telegram bot stopped")
			return nil
		}

		updates, err := b.getUpdates(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			log.Printf(
				"telegram getUpdates error: %v",
				err,
			)

			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			b.offset = update.UpdateID + 1

			if err := b.handleUpdate(ctx, update); err != nil {
				log.Printf(
					"telegram update error: %v",
					err,
				)
			}
		}
	}
}

type update struct {
	UpdateID int64 `json:"update_id"`

	Message *message `json:"message"`
}

type message struct {
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`

	Text string `json:"text"`
}

type updatesResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

type getMeResponse struct {
	OK     bool `json:"ok"`

	Result struct {
		Username string `json:"username"`
	} `json:"result"`
}

func (b *Bot) getMe(
	ctx context.Context,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		b.baseURL+"/getMe",
		nil,
	)
	if err != nil {
		return err
	}

	response, err := b.client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	var result getMeResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf(
			"telegram getMe returned not ok",
		)
	}

	log.Printf(
		"telegram bot: @%s",
		result.Result.Username,
	)

	return nil
}

func (b *Bot) getUpdates(
	ctx context.Context,
) ([]update, error) {
	params := url.Values{}

	params.Set(
		"timeout",
		"30",
	)

	params.Set(
		"offset",
		strconv.FormatInt(
			b.offset,
			10,
		),
	)

	requestURL :=
		b.baseURL +
			"/getUpdates?" +
			params.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL,
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"telegram getUpdates status: %s",
			response.Status,
		)
	}

	var result updatesResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf(
			"telegram getUpdates returned not ok",
		)
	}

	return result.Result, nil
}

func (b *Bot) handleUpdate(
	ctx context.Context,
	update update,
) error {
	if update.Message == nil {
		return nil
	}

	chatID := update.Message.Chat.ID

	text := strings.TrimSpace(
		update.Message.Text,
	)

	log.Printf(
		"telegram message: chat=%d text=%q",
		chatID,
		text,
	)

	if b.chatID == 0 {
		log.Printf(
			"telegram: set TG_CHAT_ID=%d in .env",
			chatID,
		)

		return nil
	}

	if chatID != b.chatID {
		log.Printf(
			"telegram: unauthorized chat=%d",
			chatID,
		)

		return nil
	}

	switch text {
	case "/start":
		return b.sendMessage(
			ctx,
			chatID,
			"🤖 traiderBot online\n\n"+
				"/status — статус\n"+
				"/balance — текущий USDT\n"+
				"/history — история USDT\n"+
				"/executions — последние executions\n"+
				"/opportunities — последние opportunities",
		)

	case "/status":
		return b.sendStatus(
			ctx,
			chatID,
		)

	case "/balance":
		return b.sendBalance(
			ctx,
			chatID,
		)

	case "/history":
		return b.sendHistory(
			ctx,
			chatID,
		)

	case "/executions":
		return b.sendExecutions(
			ctx,
			chatID,
		)

	case "/opportunities":
		return b.sendOpportunities(
			ctx,
			chatID,
		)

	default:
		return b.sendMessage(
			ctx,
			chatID,
			"Неизвестная команда.\n\n"+
				"/status\n"+
				"/balance\n"+
				"/history\n"+
				"/executions\n"+
				"/opportunities",
		)
	}
}

func (b *Bot) sendStatus(
	ctx context.Context,
	chatID int64,
) error {
	activeExecutions :=
		len(b.executionStore.Active())

	opportunities :=
		b.opportunityStore.Count()

	return b.sendMessage(
		ctx,
		chatID,
		fmt.Sprintf(
			"🟢 traiderBot работает\n\n"+
				"Executions active: %d\n"+
				"Opportunities stored: %d",
			activeExecutions,
			opportunities,
		),
	)
}

func (b *Bot) sendBalance(
	ctx context.Context,
	chatID int64,
) error {
	usdt := b.balanceService.USDT()

	return b.sendMessage(
		ctx,
		chatID,
		fmt.Sprintf(
			"💰 USDT available: %s",
			usdt.String(),
		),
	)
}

func (b *Bot) sendHistory(
	ctx context.Context,
	chatID int64,
) error {
	history :=
		b.balanceService.USDTHistorySince(
			time.Now().Add(-2 * time.Hour),
		)

	if len(history) == 0 {
		return b.sendMessage(
			ctx,
			chatID,
			"История USDT пока пустая.",
		)
	}

	var builder strings.Builder

	builder.WriteString(
		"📊 USDT за последние 2 часа\n\n",
	)

	start := 0

	if len(history) > 24 {
		start = len(history) - 24
	}

	for _, point := range history[start:] {
		builder.WriteString(
			point.Time.Format("15:04:05"),
		)

		builder.WriteString("  ")

		builder.WriteString(
			point.USDT.String(),
		)

		if !point.Delta.IsZero() {
			builder.WriteString("  Δ ")
			builder.WriteString(
				point.Delta.String(),
			)
		}

		builder.WriteString("\n")
	}

	return b.sendMessage(
		ctx,
		chatID,
		builder.String(),
	)
}

func (b *Bot) sendExecutions(
	ctx context.Context,
	chatID int64,
) error {
	executions :=
		b.executionStore.List(10)

	if len(executions) == 0 {
		return b.sendMessage(
			ctx,
			chatID,
			"⚡ Executions пока нет.",
		)
	}

	var builder strings.Builder

	builder.WriteString(
		"⚡ Последние executions\n\n",
	)

	for i, execution := range executions {
		data, err := json.MarshalIndent(
			execution,
			"",
			"  ",
		)
		if err != nil {
			continue
		}

		builder.WriteString(
			fmt.Sprintf(
				"#%d\n%s\n\n",
				i+1,
				string(data),
			),
		)
	}

	return b.sendLongMessage(
		ctx,
		chatID,
		builder.String(),
	)
}

func (b *Bot) sendOpportunities(
	ctx context.Context,
	chatID int64,
) error {
	opportunities :=
		b.opportunityStore.List(10)

	if len(opportunities) == 0 {
		return b.sendMessage(
			ctx,
			chatID,
			"🎯 Opportunities пока нет.",
		)
	}

	var builder strings.Builder

	builder.WriteString(
		"🎯 Последние opportunities\n\n",
	)

	for i, opportunity := range opportunities {
		data, err := json.MarshalIndent(
			opportunity,
			"",
			"  ",
		)
		if err != nil {
			continue
		}

		builder.WriteString(
			fmt.Sprintf(
				"#%d\n%s\n\n",
				i+1,
				string(data),
			),
		)
	}

	return b.sendLongMessage(
		ctx,
		chatID,
		builder.String(),
	)
}

func (b *Bot) sendLongMessage(
	ctx context.Context,
	chatID int64,
	text string,
) error {
	const maxMessageLength = 3900

	for len(text) > 0 {
		part := text

		if len(part) > maxMessageLength {
			part = text[:maxMessageLength]

			if index := strings.LastIndex(
				part,
				"\n",
			); index > 0 {
				part = part[:index]
			}
		}

		if err := b.sendMessage(
			ctx,
			chatID,
			part,
		); err != nil {
			return err
		}

		text = strings.TrimPrefix(
			text[len(part):],
			"\n",
		)
	}

	return nil
}

func (b *Bot) sendMessage(
	ctx context.Context,
	chatID int64,
	text string,
) error {
	params := url.Values{}

	params.Set(
		"chat_id",
		strconv.FormatInt(
			chatID,
			10,
		),
	)

	params.Set(
		"text",
		text,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		b.baseURL+"/sendMessage",
		strings.NewReader(
			params.Encode(),
		),
	)
	if err != nil {
		return err
	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	response, err := b.client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"telegram sendMessage status: %s",
			response.Status,
		)
	}

	return nil
}