package console

import (
	"fmt"
	"io"

	"traiderBot/internalDDD/debug/application"
)

type Handler struct {
	debugService *application.DebugService
	writer       io.Writer
}

func NewHandler(
	debugService *application.DebugService,
	writer io.Writer,
) *Handler {
	return &Handler{
		debugService: debugService,
		writer:       writer,
	}
}

func (h *Handler) PrintProviders() []string {
	providers := h.debugService.Providers()

	fmt.Fprintln(h.writer)
	fmt.Fprintln(h.writer, "Providers:")

	if len(providers) == 0 {
		fmt.Fprintln(h.writer, "  no providers registered")
		return providers
	}

	for i, name := range providers {
		fmt.Fprintf(
			h.writer,
			"%d. %s\n",
			i+1,
			name,
		)
	}

	return providers
}

func (h *Handler) PrintProvider(name string) {
	provider, err := h.debugService.Provider(name)
	if err != nil {
		fmt.Fprintf(
			h.writer,
			"error: %v\n",
			err,
		)
		return
	}

	fmt.Fprintln(h.writer)
	fmt.Fprintln(h.writer, "--------------------------------")
	fmt.Fprintf(h.writer, " %s\n", provider.Name())
	fmt.Fprintln(h.writer, "--------------------------------")
	fmt.Fprintln(h.writer)
	fmt.Fprintln(h.writer, "1. Snapshot")
	fmt.Fprintln(h.writer, "2. Metrics")
	fmt.Fprintln(h.writer, "0. Back")
}

func (h *Handler) PrintSnapshot(name string) {
	provider, err := h.debugService.Provider(name)
	if err != nil {
		fmt.Fprintf(
			h.writer,
			"error: %v\n",
			err,
		)
		return
	}

	snapshot, err := provider.Snapshot()
	if err != nil {
		fmt.Fprintf(
			h.writer,
			"error: %v\n",
			err,
		)
		return
	}

	fmt.Fprintln(h.writer)
	fmt.Fprintln(h.writer, snapshot)
}

func (h *Handler) PrintMetrics(name string) {
	provider, err := h.debugService.Provider(name)
	if err != nil {
		fmt.Fprintf(
			h.writer,
			"error: %v\n",
			err,
		)
		return
	}

	metrics, err := provider.Metrics()
	if err != nil {
		fmt.Fprintf(
			h.writer,
			"error: %v\n",
			err,
		)
		return
	}

	fmt.Fprintln(h.writer)
	fmt.Fprintf(h.writer, "%v\n", metrics)
}