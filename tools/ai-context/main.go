package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultRoot   = "internalDDD"
	defaultOutput = ".ai-context"
)

type FileSummary struct {
	Path       string
	Package    string
	Imports    []string
	Types      []string
	Interfaces []string
	Structs    []string
	Functions  []string
	Methods    []string
	Constants  []string
	Variables  []string
}

type BoundedContext struct {
	Name  string
	Files []FileSummary
}

func main() {
	root := defaultRoot
	output := defaultOutput

	if len(os.Args) > 1 && os.Args[1] != "" {
		root = os.Args[1]
	}

	if len(os.Args) > 2 && os.Args[2] != "" {
		output = os.Args[2]
	}

	if err := generateContext(root, output); err != nil {
		fmt.Fprintf(os.Stderr, "ai-context: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"AI context generated: %s\n",
		output,
	)
}

func generateContext(
	root string,
	output string,
) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf(
			"stat root %q: %w",
			root,
			err,
		)
	}

	if !info.IsDir() {
		return fmt.Errorf(
			"root %q is not a directory",
			root,
		)
	}

	contexts, err := collectBoundedContexts(root)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(output); err != nil {
		return fmt.Errorf(
			"remove old output: %w",
			err,
		)
	}

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf(
			"create output directory: %w",
			err,
		)
	}

	if err := writeIndex(output, contexts); err != nil {
		return err
	}

	for _, boundedContext := range contexts {
		if err := writeBoundedContext(
			output,
			boundedContext,
		); err != nil {
			return err
		}
	}

	return nil
}

func collectBoundedContexts(
	root string,
) ([]BoundedContext, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf(
			"read %q: %w",
			root,
			err,
		)
	}

	var result []BoundedContext

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if strings.HasPrefix(name, ".") {
			continue
		}

		path := filepath.Join(root, name)

		files, err := collectGoFiles(path)
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			continue
		}

		sort.Slice(
			files,
			func(i, j int) bool {
				return files[i].Path < files[j].Path
			},
		)

		result = append(
			result,
			BoundedContext{
				Name:  name,
				Files: files,
			},
		)
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].Name < result[j].Name
		},
	)

	return result, nil
}

func collectGoFiles(
	root string,
) ([]FileSummary, error) {
	var result []FileSummary

	err := filepath.Walk(
		root,
		func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if shouldSkipDir(info.Name()) {
					return filepath.SkipDir
				}

				return nil
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			if strings.HasSuffix(path, "_test.go") {
				result = append(
					result,
					summarizeGoFile(
						root,
						path,
						true,
					),
				)

				return nil
			}

			result = append(
				result,
				summarizeGoFile(
					root,
					path,
					false,
				),
			)

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf(
			"walk %q: %w",
			root,
			err,
		)
	}

	return result, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git",
		".idea",
		".vscode",
		"vendor",
		"node_modules":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func summarizeGoFile(
	root string,
	path string,
	isTest bool,
) FileSummary {
	summary := FileSummary{
		Path: filepath.ToSlash(
			path,
		),
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(
		fset,
		path,
		nil,
		parser.ParseComments,
	)

	if err != nil {
		return summary
	}

	summary.Package = file.Name.Name

	for _, imp := range file.Imports {
		if imp.Path != nil {
			summary.Imports = append(
				summary.Imports,
				strings.Trim(
					imp.Path.Value,
					`"`,
				),
			)
		}
	}

	for _, decl := range file.Decls {
		switch node := decl.(type) {

		case *ast.GenDecl:
			summarizeGenDecl(
				node,
				&summary,
			)

		case *ast.FuncDecl:
			summarizeFuncDecl(
				node,
				&summary,
			)
		}
	}

	if isTest {
		summary.Path += " [test]"
	}

	normalizeSummary(&summary)

	_ = root

	return summary
}

func summarizeGenDecl(
	decl *ast.GenDecl,
	summary *FileSummary,
) {
	switch decl.Tok {

	case token.CONST:
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				summary.Constants = append(
					summary.Constants,
					name.Name,
				)
			}
		}

	case token.VAR:
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				summary.Variables = append(
					summary.Variables,
					name.Name,
				)
			}
		}

	case token.TYPE:
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			name := typeSpec.Name.Name

			summary.Types = append(
				summary.Types,
				formatTypeSignature(typeSpec),
			)

			switch typeSpec.Type.(type) {
			case *ast.StructType:
				summary.Structs = append(
					summary.Structs,
					name,
				)

			case *ast.InterfaceType:
				summary.Interfaces = append(
					summary.Interfaces,
					name,
				)
			}
		}
	}
}

func summarizeFuncDecl(
	decl *ast.FuncDecl,
	summary *FileSummary,
) {
	signature := formatFuncSignature(decl)

	if decl.Recv == nil {
		summary.Functions = append(
			summary.Functions,
			signature,
		)

		return
	}

	summary.Methods = append(
		summary.Methods,
		signature,
	)
}

func formatTypeSignature(
	spec *ast.TypeSpec,
) string {
	switch typeExpr := spec.Type.(type) {

	case *ast.StructType:
		return formatStructSignature(
			spec.Name.Name,
			typeExpr,
		)

	case *ast.InterfaceType:
		return formatInterfaceSignature(
			spec.Name.Name,
			typeExpr,
		)

	default:
		return fmt.Sprintf(
			"type %s %s",
			spec.Name.Name,
			exprString(typeExpr),
		)
	}
}

func formatStructSignature(
	name string,
	structType *ast.StructType,
) string {
	var fields []string

	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldType := exprString(field.Type)

			if len(field.Names) == 0 {
				fields = append(
					fields,
					fieldType,
				)

				continue
			}

			for _, fieldName := range field.Names {
				fields = append(
					fields,
					fmt.Sprintf(
						"%s %s",
						fieldName.Name,
						fieldType,
					),
				)
			}
		}
	}

	return fmt.Sprintf(
		"type %s struct { %s }",
		name,
		strings.Join(fields, "; "),
	)
}

func formatInterfaceSignature(
	name string,
	interfaceType *ast.InterfaceType,
) string {
	var methods []string

	if interfaceType.Methods != nil {
		for _, field := range interfaceType.Methods.List {
			switch typeExpr := field.Type.(type) {

			case *ast.FuncType:
				methodName := ""

				if len(field.Names) > 0 {
					methodName = field.Names[0].Name
				}

				methods = append(
					methods,
					fmt.Sprintf(
						"%s%s",
						methodName,
						formatFuncType(typeExpr),
					),
				)

			default:
				methods = append(
					methods,
					exprString(typeExpr),
				)
			}
		}
	}

	return fmt.Sprintf(
		"type %s interface { %s }",
		name,
		strings.Join(methods, "; "),
	)
}

func formatFuncSignature(
	decl *ast.FuncDecl,
) string {
	var receiver string

	if decl.Recv != nil &&
		len(decl.Recv.List) > 0 {
		receiver = fmt.Sprintf(
			"(%s) ",
			exprString(
				decl.Recv.List[0].Type,
			),
		)
	}

	return fmt.Sprintf(
		"func %s%s%s",
		receiver,
		decl.Name.Name,
		formatFuncType(decl.Type),
	)
}

func formatFuncType(
	funcType *ast.FuncType,
) string {
	var params []string
	var results []string

	if funcType.Params != nil {
		params = formatFieldList(
			funcType.Params,
		)
	}

	if funcType.Results != nil {
		results = formatFieldList(
			funcType.Results,
		)
	}

	signature := "(" +
		strings.Join(params, ", ") +
		")"

	switch len(results) {
	case 0:
		return signature

	case 1:
		return signature +
			" " +
			results[0]

	default:
		return signature +
			" (" +
			strings.Join(results, ", ") +
			")"
	}
}

func formatFieldList(
	fields *ast.FieldList,
) []string {
	var result []string

	for _, field := range fields.List {
		fieldType := exprString(field.Type)

		if len(field.Names) == 0 {
			result = append(
				result,
				fieldType,
			)

			continue
		}

		for _, name := range field.Names {
			result = append(
				result,
				name.Name+" "+fieldType,
			)
		}
	}

	return result
}

func exprString(
	expr ast.Expr,
) string {
	switch node := expr.(type) {

	case *ast.Ident:
		return node.Name

	case *ast.SelectorExpr:
		return exprString(node.X) +
			"." +
			node.Sel.Name

	case *ast.StarExpr:
		return "*" +
			exprString(node.X)

	case *ast.ArrayType:
		return "[]" +
			exprString(node.Elt)

	case *ast.MapType:
		return "map[" +
			exprString(node.Key) +
			"]" +
			exprString(node.Value)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.StructType:
		return "struct{...}"

	case *ast.FuncType:
		return "func" +
			formatFuncType(node)

	case *ast.Ellipsis:
		return "..." +
			exprString(node.Elt)

	case *ast.ChanType:
		return "chan " +
			exprString(node.Value)

	case *ast.ParenExpr:
		return "(" +
			exprString(node.X) +
			")"

	case *ast.BasicLit:
		return node.Value

	case *ast.IndexExpr:
		return exprString(node.X) +
			"[" +
			exprString(node.Index) +
			"]"

	case *ast.IndexListExpr:
		var indices []string

		for _, index := range node.Indices {
			indices = append(
				indices,
				exprString(index),
			)
		}

		return exprString(node.X) +
			"[" +
			strings.Join(indices, ", ") +
			"]"

	default:
		return "..."
	}
}

func normalizeSummary(
	summary *FileSummary,
) {
	sort.Strings(summary.Imports)
	sort.Strings(summary.Types)
	sort.Strings(summary.Interfaces)
	sort.Strings(summary.Structs)
	sort.Strings(summary.Functions)
	sort.Strings(summary.Methods)
	sort.Strings(summary.Constants)
	sort.Strings(summary.Variables)
}

func writeIndex(
	output string,
	contexts []BoundedContext,
) error {
	path := filepath.Join(
		output,
		"README.md",
	)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"create %q: %w",
			path,
			err,
		)
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	fmt.Fprintln(
		writer,
		"# AI Context",
	)
	fmt.Fprintln(writer)

	fmt.Fprintln(
		writer,
		"This directory contains architecture summaries generated from `internalDDD`.",
	)
	fmt.Fprintln(
		writer,
		"Source files are intentionally not copied here.",
	)
	fmt.Fprintln(
		writer,
		"Use the summaries to navigate the project; request the actual source file when implementation details are required.",
	)
	fmt.Fprintln(writer)

	fmt.Fprintln(
		writer,
		"## Bounded Contexts",
	)
	fmt.Fprintln(writer)

	for _, boundedContext := range contexts {
		fmt.Fprintf(
			writer,
			"- [%s](%s/README.md)\n",
			boundedContext.Name,
			boundedContext.Name,
		)
	}

	return writer.Flush()
}

func writeBoundedContext(
	output string,
	context BoundedContext,
) error {
	dir := filepath.Join(
		output,
		context.Name,
	)

	if err := os.MkdirAll(
		dir,
		0o755,
	); err != nil {
		return fmt.Errorf(
			"create bounded context %q: %w",
			context.Name,
			err,
		)
	}

	readmePath := filepath.Join(
		dir,
		"README.md",
	)

	file, err := os.Create(readmePath)
	if err != nil {
		return fmt.Errorf(
			"create %q: %w",
			readmePath,
			err,
		)
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	fmt.Fprintf(
		writer,
		"# Bounded Context: %s\n\n",
		context.Name,
	)

	fmt.Fprintln(
		writer,
		"Only public architecture/signature information is included here.",
	)
	fmt.Fprintln(
		writer,
		"Implementation bodies are intentionally omitted.",
	)
	fmt.Fprintln(writer)

	for _, fileSummary := range context.Files {
		writeFileSummary(
			writer,
			fileSummary,
		)
	}

	return writer.Flush()
}

func writeFileSummary(
	writer *bufio.Writer,
	summary FileSummary,
) {
	fmt.Fprintf(
		writer,
		"## `%s`\n\n",
		summary.Path,
	)

	if summary.Package != "" {
		fmt.Fprintf(
			writer,
			"**Package:** `%s`\n\n",
			summary.Package,
		)
	}

	if len(summary.Imports) > 0 {
		fmt.Fprintln(
			writer,
			"### Imports",
		)

		for _, item := range summary.Imports {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Interfaces) > 0 {
		fmt.Fprintln(
			writer,
			"### Interfaces",
		)

		for _, item := range summary.Interfaces {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Structs) > 0 {
		fmt.Fprintln(
			writer,
			"### Structs",
		)

		for _, item := range summary.Structs {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Types) > 0 {
		fmt.Fprintln(
			writer,
			"### Types",
		)

		for _, item := range summary.Types {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Functions) > 0 {
		fmt.Fprintln(
			writer,
			"### Functions",
		)

		for _, item := range summary.Functions {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Methods) > 0 {
		fmt.Fprintln(
			writer,
			"### Methods",
		)

		for _, item := range summary.Methods {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Constants) > 0 {
		fmt.Fprintln(
			writer,
			"### Constants",
		)

		for _, item := range summary.Constants {
			fmt.Fprintf(
				writer,
				"- `%s`\n",
				item,
			)
		}

		fmt.Fprintln(writer)
	}

	if len(summary.Variables) > 0 {
		fmt.Fprintln(
			writer,
			"### Variables",
		)

		for _, item := range summary.Variables {
			fmt.Fprintf(
				writer,
				"- `%s`\n",	
				item,
			)
		}

		fmt.Fprintln(writer)
	}
}
