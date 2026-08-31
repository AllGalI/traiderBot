# debug — infrastructure

Source: `internalDDD/debug/infrastructure/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/debug/infrastructure/console/comand_handler.go

Package: `console`

### Structs

```go
CommandHandler struct {
	handler *Handler
	reader  *bufio.Reader
	writer  io.Writer
}
```

### Constructors

```go
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
```

### Methods

```go
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
			fmt.Fprintln(
				h.writer,
				"Unknown command.",
			)
		}
	}
}
```

```go
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
```

```go
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

		case "0":
			return nil

		default:
			fmt.Fprintln(
				h.writer,
				"Unknown command.",
			)
		}

		fmt.Fprintln(h.writer)
	}
}
```

```go
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
			fmt.Fprintln(
				h.writer,
				"Enter a number.",
			)
			continue
		}

		if index < 1 || index > len(providers) {
			fmt.Fprintln(
				h.writer,
				"Invalid provider.",
			)
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
```

```go
func (h *CommandHandler) readLine() (string, error) {
	line, err := h.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}
```

## internalDDD/debug/infrastructure/console/handler.go

Package: `console`

### Structs

```go
Handler struct {
	debugService *application.DebugService
	writer       io.Writer
}
```

### Constructors

```go
func NewHandler(
	debugService *application.DebugService,
	writer io.Writer,
) *Handler {
	return &Handler{
		debugService: debugService,
		writer:       writer,
	}
}
```

### Functions

```go
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
```

```go
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
```

### Methods

```go
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
```

```go
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
```

```go
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
```

```go
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
```

