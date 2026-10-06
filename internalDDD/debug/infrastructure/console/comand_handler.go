package console

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type CommandHandler struct {
	handler *Handler
	reader  *bufio.Reader
	writer  io.Writer
}

func NewCommandHandler(
	handler *Handler,
	reader io.Reader,
	writer io.Writer,
) *CommandHandler {
	return &CommandHandler{
		handler: handler,
		reader:  bufio.NewReader(reader),
		writer:  writer,
	}
}

func (h *CommandHandler) Run(ctx context.Context) error {
	for {
		if err := h.printMainMenu(); err != nil {
			return err
		}

		fmt.Fprint(h.writer, "Select: ")

		input, err := h.readLine()
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		switch input {
		case "1":
			if err := h.providersMenu(ctx); err != nil {
				return err
			}

		case "0":
			return nil

		default:
			fmt.Fprintln(h.writer, "Unknown command.")
		}
	}
}

func (h *CommandHandler) printMainMenu() error {
	fmt.Fprintln(h.writer)
	fmt.Fprintln(h.writer, "================================")
	fmt.Fprintln(h.writer, " Debug Console")
	fmt.Fprintln(h.writer, "================================")
	fmt.Fprintln(h.writer)

	fmt.Fprintln(h.writer, "1. Providers")
	fmt.Fprintln(h.writer, "0. Exit")
	fmt.Fprintln(h.writer)

	return nil
}

func (h *CommandHandler) providersMenu(
	ctx context.Context,
) error {
	for {
		providers := h.handler.PrintProviders()

		fmt.Fprintln(h.writer)
		fmt.Fprintln(h.writer, "0. Back")
		fmt.Fprintln(h.writer)

		fmt.Fprint(h.writer, "Select: ")

		input, err := h.readLine()
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if input == "0" {
			return nil
		}

		index, err := strconv.Atoi(input)
		if err != nil {
			fmt.Fprintln(h.writer, "Enter a number.")
			continue
		}

		if index < 1 || index > len(providers) {
			fmt.Fprintln(h.writer, "Invalid provider.")
			continue
		}

		providerName := providers[index-1]

		if err := h.providerMenu(
			ctx,
			providerName,
		); err != nil {
			return err
		}
	}
}

func (h *CommandHandler) providerMenu(
	ctx context.Context,
	providerName string,
) error {
	for {
		h.handler.PrintProvider(providerName)

		fmt.Fprint(h.writer, "Select: ")

		input, err := h.readLine()
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		switch input {
		case "1":
			h.handler.PrintSnapshot(providerName)

		case "2":
			h.handler.PrintMetrics(providerName)

		case "3":
			if err := h.handler.BrowseProvider(
				ctx,
				providerName,
				h.reader,
			); err != nil {
				fmt.Fprintf(
					h.writer,
					"error: %v\n",
					err,
				)
			}

		case "0":
			return nil

		default:
			fmt.Fprintln(h.writer, "Unknown command.")
		}

		fmt.Fprintln(h.writer)
	}
}

func (h *CommandHandler) readLine() (string, error) {
	line, err := h.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
