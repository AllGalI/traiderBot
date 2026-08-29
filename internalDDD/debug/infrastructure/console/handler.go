package console

import (
	"fmt"
	"io"
	"sort"
	"strings"

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

	sort.Strings(providers)

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
	fmt.Fprintln(h.writer, "Metrics")
	fmt.Fprintln(h.writer, "-------")

	printMetrics(h.writer, metrics, 0)
}

func printMetrics(
	writer io.Writer,
	value any,
	indent int,
) {
	prefix := strings.Repeat("  ", indent)

	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))

		for key := range value {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			current := value[key]

			if isNestedMetric(current) {
				fmt.Fprintf(
					writer,
					"%s%s:\n",
					prefix,
					key,
				)

				printMetrics(
					writer,
					current,
					indent+1,
				)

				continue
			}

			fmt.Fprintf(
				writer,
				"%s%-24s %v\n",
				prefix,
				key+":",
				current,
			)
		}

	case map[string]int:
		keys := make([]string, 0, len(value))

		for key := range value {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			fmt.Fprintf(
				writer,
				"%s%-24s %d\n",
				prefix,
				key+":",
				value[key],
			)
		}

	default:
		fmt.Fprintf(
			writer,
			"%s%v\n",
			prefix,
			value,
		)
	}
}

func isNestedMetric(value any) bool {
	switch value.(type) {
	case map[string]any:
		return true
	case map[string]int:
		return true
	default:
		return false
	}
}
