package console

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	debugApplication "traiderBot/internalDDD/debug/application"
	debugDomain "traiderBot/internalDDD/debug/domain"
)

type Handler struct {
	debugService *debugApplication.DebugService
	writer       io.Writer
}

func NewHandler(
	debugService *debugApplication.DebugService,
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

	if _, ok := provider.(debugDomain.BrowserProvider); ok {
		fmt.Fprintln(h.writer, "3. Browse")
	}

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
	fmt.Fprintln(h.writer, "Snapshot")
	fmt.Fprintln(h.writer, "--------")

	printValue(h.writer, snapshot, 0)
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

	printValue(h.writer, metrics, 0)
}

func (h *Handler) BrowseProvider(
	ctx context.Context,
	name string,
	reader io.Reader,
) error {
	provider, err := h.debugService.Provider(name)
	if err != nil {
		return err
	}

	browser, ok := provider.(debugDomain.BrowserProvider)
	if !ok {
		return fmt.Errorf(
			"provider %q does not support browsing",
			name,
		)
	}

	return browser.Browse(
		ctx,
		reader,
		h.writer,
	)
}

func printValue(
	writer io.Writer,
	value any,
	indent int,
) {
	prefix := strings.Repeat("  ", indent)

	if value == nil {
		fmt.Fprintf(writer, "%s-\n", prefix)
		return
	}

	switch value := value.(type) {
	case map[string]any:
		printMap(writer, value, indent)
		return

	case map[string]string:
		printStringMap(writer, value, indent)
		return

	case map[string]int:
		printIntMap(writer, value, indent)
		return

	case []any:
		printSlice(writer, value, indent)
		return

	case []string:
		for _, item := range value {
			fmt.Fprintf(writer, "%s- %s\n", prefix, item)
		}
		return

	default:
		if printReflectValue(writer, value, indent) {
			return
		}

		fmt.Fprintf(writer, "%s%v\n", prefix, value)
	}
}

func printMap(
	writer io.Writer,
	value map[string]any,
	indent int,
) {
	keys := make([]string, 0, len(value))

	for key := range value {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		current := value[key]

		if isContainer(current) {
			fmt.Fprintf(
				writer,
				"%s%s:\n",
				strings.Repeat("  ", indent),
				key,
			)

			printValue(writer, current, indent+1)
			continue
		}

		fmt.Fprintf(
			writer,
			"%s%-24s %v\n",
			strings.Repeat("  ", indent),
			key+":",
			current,
		)
	}
}

func printStringMap(
	writer io.Writer,
	value map[string]string,
	indent int,
) {
	keys := make([]string, 0, len(value))

	for key := range value {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	prefix := strings.Repeat("  ", indent)

	for _, key := range keys {
		fmt.Fprintf(
			writer,
			"%s%-24s %s\n",
			prefix,
			key+":",
			value[key],
		)
	}
}

func printIntMap(
	writer io.Writer,
	value map[string]int,
	indent int,
) {
	keys := make([]string, 0, len(value))

	for key := range value {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	prefix := strings.Repeat("  ", indent)

	for _, key := range keys {
		fmt.Fprintf(
			writer,
			"%s%-24s %d\n",
			prefix,
			key+":",
			value[key],
		)
	}
}

func printSlice(
	writer io.Writer,
	value []any,
	indent int,
) {
	prefix := strings.Repeat("  ", indent)

	for _, item := range value {
		if isContainer(item) {
			printValue(writer, item, indent)
			continue
		}

		fmt.Fprintf(
			writer,
			"%s- %v\n",
			prefix,
			item,
		)
	}
}

func printReflectValue(
	writer io.Writer,
	value any,
	indent int,
) bool {
	reflected := reflect.ValueOf(value)

	if !reflected.IsValid() {
		return false
	}

	switch reflected.Kind() {
	case reflect.Slice, reflect.Array:
		length := reflected.Len()

		for index := 0; index < length; index++ {
			item := reflected.Index(index).Interface()
			printValue(writer, item, indent)
		}

		return true

	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return false
		}

		keys := reflected.MapKeys()

		sort.Slice(
			keys,
			func(i, j int) bool {
				return keys[i].String() < keys[j].String()
			},
		)

		for _, key := range keys {
			item := reflected.MapIndex(key).Interface()

			if isContainer(item) {
				fmt.Fprintf(
					writer,
					"%s%s:\n",
					strings.Repeat("  ", indent),
					key.String(),
				)

				printValue(writer, item, indent+1)
				continue
			}

			fmt.Fprintf(
				writer,
				"%s%-24s %v\n",
				strings.Repeat("  ", indent),
				key.String()+":",
				item,
			)
		}

		return true
	}

	return false
}

func isContainer(value any) bool {
	if value == nil {
		return false
	}

	switch value.(type) {
	case map[string]any:
		return true

	case map[string]string:
		return true

	case map[string]int:
		return true

	case []any:
		return true

	case []string:
		return true
	}

	valueType := reflect.TypeOf(value)

	if valueType == nil {
		return false
	}

	return valueType.Kind() == reflect.Map ||
		valueType.Kind() == reflect.Slice ||
		valueType.Kind() == reflect.Array
}
