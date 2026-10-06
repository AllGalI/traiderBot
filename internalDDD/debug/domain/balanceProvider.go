package domain

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	balanceApplication "traiderBot/internalDDD/balance/application"

	"github.com/shopspring/decimal"
)

const balancePageSize = 10

type BalanceProvider struct {
	service *balanceApplication.BalanceService
}

func NewBalanceProvider(
	service *balanceApplication.BalanceService,
) *BalanceProvider {
	return &BalanceProvider{
		service: service,
	}
}

func (p *BalanceProvider) Name() string {
	return "BalanceProvider"
}

func (p *BalanceProvider) Snapshot() (any, error) {
	balances := p.service.Snapshot()

	result := map[string]any{
		"assets": len(balances),
	}

	assets := make([]any, 0, len(balances))

	for _, balance := range balances {
		assets = append(
			assets,
			map[string]any{
				"asset":          balance.Asset,
				"total":          balance.Total,
				"exchangeLocked": balance.ExchangeLocked,
				"reserved":       balance.Reserved,
				"available":      balance.Available,
			},
		)
	}

	result["balances"] = assets

	return result, nil
}

func (p *BalanceProvider) Metrics() (any, error) {
	balances := p.service.Snapshot()

	result := map[string]any{
		"assets": len(balances),
	}

	byAsset := make(map[string]any, len(balances))

	for _, balance := range balances {
		byAsset[balance.Asset] = map[string]any{
			"total":          balance.Total,
			"exchangeLocked": balance.ExchangeLocked,
			"reserved":       balance.Reserved,
			"available":      balance.Available,
		}
	}

	result["byAsset"] = byAsset

	return result, nil
}

func (p *BalanceProvider) Browse(
	ctx context.Context,
	reader io.Reader,
	writer io.Writer,
) error {
	input := bufio.NewReader(reader)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		balances := p.service.Snapshot()

		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "--------------------------------")
		fmt.Fprintln(writer, " Balance Assets")
		fmt.Fprintln(writer, "--------------------------------")
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "1. List assets")
		fmt.Fprintln(writer, "2. Search asset")
		fmt.Fprintln(writer, "0. Back")
		fmt.Fprintln(writer)

		fmt.Fprint(writer, "Select: ")

		command, err := readBalanceInput(input)
		if err != nil {
			return err
		}

		switch command {
		case "1":
			if err := browseBalancePages(
				ctx,
				input,
				writer,
				balances,
			); err != nil {
				return err
			}

		case "2":
			if err := searchBalance(
				ctx,
				input,
				writer,
				balances,
			); err != nil {
				return err
			}

		case "0":
			return nil

		default:
			fmt.Fprintln(writer, "Unknown command.")
		}
	}
}

func browseBalancePages(
	ctx context.Context,
	reader *bufio.Reader,
	writer io.Writer,
	balances []balanceApplication.BalanceSnapshot,
) error {
	page := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		total := len(balances)

		if total == 0 {
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, "No balances.")
			return nil
		}

		totalPages := (total + balancePageSize - 1) / balancePageSize

		if page >= totalPages {
			page = totalPages - 1
		}

		start := page * balancePageSize
		end := start + balancePageSize

		if end > total {
			end = total
		}

		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "--------------------------------")
		fmt.Fprintf(
			writer,
			" Assets %d-%d / %d\n",
			start+1,
			end,
			total,
		)
		fmt.Fprintln(writer, "--------------------------------")
		fmt.Fprintln(writer)

		for i := start; i < end; i++ {
			fmt.Fprintf(
				writer,
				"%d. %s\n",
				i-start+1,
				balances[i].Asset,
			)
		}

		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "n. Next page")
		fmt.Fprintln(writer, "p. Previous page")
		fmt.Fprintln(writer, "s. Search")
		fmt.Fprintln(writer, "0. Back")
		fmt.Fprintln(writer)

		fmt.Fprint(writer, "Select: ")

		command, err := readBalanceInput(reader)
		if err != nil {
			return err
		}

		switch strings.ToLower(command) {
		case "n":
			if page+1 < totalPages {
				page++
			} else {
				fmt.Fprintln(writer, "Already on last page.")
			}

		case "p":
			if page > 0 {
				page--
			} else {
				fmt.Fprintln(writer, "Already on first page.")
			}

		case "s":
			if err := searchBalance(
				ctx,
				reader,
				writer,
				balances,
			); err != nil {
				return err
			}

		case "0":
			return nil

		default:
			index, err := strconv.Atoi(command)
			if err != nil {
				fmt.Fprintln(writer, "Unknown command.")
				continue
			}

			if index < 1 || index > end-start {
				fmt.Fprintln(writer, "Invalid asset.")
				continue
			}

			printBalance(
				writer,
				balances[start+index-1],
			)
		}
	}
}

func searchBalance(
	ctx context.Context,
	reader *bufio.Reader,
	writer io.Writer,
	balances []balanceApplication.BalanceSnapshot,
) error {
	fmt.Fprintln(writer)
	fmt.Fprint(writer, "Search asset: ")

	query, err := readBalanceInput(reader)
	if err != nil {
		return err
	}

	query = strings.ToUpper(strings.TrimSpace(query))

	if query == "" {
		return nil
	}

	matches := make([]balanceApplication.BalanceSnapshot, 0)

	for _, balance := range balances {
		if strings.Contains(
			strings.ToUpper(balance.Asset),
			query,
		) {
			matches = append(matches, balance)
		}
	}

	fmt.Fprintln(writer)

	if len(matches) == 0 {
		fmt.Fprintf(
			writer,
			"No assets found for %q.\n",
			query,
		)
		return nil
	}

	fmt.Fprintf(
		writer,
		"Found %d asset(s):\n",
		len(matches),
	)

	for i, balance := range matches {
		fmt.Fprintf(
			writer,
			"%d. %s\n",
			i+1,
			balance.Asset,
		)
	}

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Enter asset number or 0 to back.")
	fmt.Fprint(writer, "Select: ")

	command, err := readBalanceInput(reader)
	if err != nil {
		return err
	}

	if command == "0" {
		return nil
	}

	index, err := strconv.Atoi(command)
	if err != nil {
		fmt.Fprintln(writer, "Invalid asset.")
		return nil
	}

	if index < 1 || index > len(matches) {
		fmt.Fprintln(writer, "Invalid asset.")
		return nil
	}

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	printBalance(
		writer,
		matches[index-1],
	)

	return nil
}

func printBalance(
	writer io.Writer,
	balance balanceApplication.BalanceSnapshot,
) {
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "--------------------------------")
	fmt.Fprintf(writer, " %s\n", balance.Asset)
	fmt.Fprintln(writer, "--------------------------------")
	fmt.Fprintln(writer)

	printDecimalField(
		writer,
		"available",
		balance.Available,
	)

	printDecimalField(
		writer,
		"exchangeLocked",
		balance.ExchangeLocked,
	)

	printDecimalField(
		writer,
		"reserved",
		balance.Reserved,
	)

	printDecimalField(
		writer,
		"total",
		balance.Total,
	)

	fmt.Fprintln(writer)
}

func printDecimalField(
	writer io.Writer,
	name string,
	value decimal.Decimal,
) {
	fmt.Fprintf(
		writer,
		"%-24s %s\n",
		name+":",
		value.String(),
	)
}

func readBalanceInput(
	reader *bufio.Reader,
) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
