package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	sourceRoot = "internalDDD"
	outputRoot = ".ai/context"
)

var layers = []string{
	"domain",
	"application",
	"infrastructure",
}

type boundedContext struct {
	Name   string
	Layers map[string]*layerInfo
}

type layerInfo struct {
	Name  string
	Files []fileInfo
}

type fileInfo struct {
	Path         string
	Package      string
	Imports      []string
	Interfaces   []string
	Structs      []string
	Constructors []string
	Functions    []string
	Methods      []string
}

type dependencyInfo struct {
	BoundedContext string
	Layer          string
	Package        string
	ImportPath     string
}

func main() {
	fset := token.NewFileSet()

	contexts, dependencies, err := collectContexts(fset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect contexts: %v\n", err)
		os.Exit(1)
	}

	if err := prepareOutputDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "prepare output directory: %v\n", err)
		os.Exit(1)
	}

	if err := generateContexts(
		contexts,
		dependencies,
	); err != nil {
		fmt.Fprintf(os.Stderr, "generate contexts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"AI context generated in %s\n",
		outputRoot,
	)
}

func collectContexts(
	fset *token.FileSet,
) ([]boundedContext, []dependencyInfo, error) {
	contextMap := make(map[string]*boundedContext)
	var dependencies []dependencyInfo

	err := filepath.Walk(
		sourceRoot,
		func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			// Тесты не входят в основной AI-контекст.
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			contextName, layerName, ok := classifyPath(path)
			if !ok {
				return nil
			}

			file, err := parser.ParseFile(
				fset,
				path,
				nil,
				parser.ParseComments,
			)
			if err != nil {
				return fmt.Errorf(
					"parse %s: %w",
					path,
					err,
				)
			}

			bc, ok := contextMap[contextName]
			if !ok {
				bc = &boundedContext{
					Name:   contextName,
					Layers: make(map[string]*layerInfo),
				}

				contextMap[contextName] = bc
			}

			layer, ok := bc.Layers[layerName]
			if !ok {
				layer = &layerInfo{
					Name: layerName,
				}

				bc.Layers[layerName] = layer
			}

			fileInfo := analyzeFile(
				fset,
				file,
				path,
			)

			layer.Files = append(
				layer.Files,
				fileInfo,
			)

			for _, importPath := range fileInfo.Imports {
				dependencies = append(
					dependencies,
					dependencyInfo{
						BoundedContext: contextName,
						Layer:          layerName,
						Package:        fileInfo.Package,
						ImportPath:     importPath,
					},
				)
			}

			return nil
		},
	)
	if err != nil {
		return nil, nil, err
	}

	contexts := make(
		[]boundedContext,
		0,
		len(contextMap),
	)

	for _, bc := range contextMap {
		for _, layer := range bc.Layers {
			sort.Slice(
				layer.Files,
				func(i, j int) bool {
					return layer.Files[i].Path <
						layer.Files[j].Path
				},
			)
		}

		contexts = append(
			contexts,
			*bc,
		)
	}

	sort.Slice(
		contexts,
		func(i, j int) bool {
			return contexts[i].Name <
				contexts[j].Name
		},
	)

	sort.Slice(
		dependencies,
		func(i, j int) bool {
			if dependencies[i].BoundedContext !=
				dependencies[j].BoundedContext {
				return dependencies[i].BoundedContext <
					dependencies[j].BoundedContext
			}

			if dependencies[i].Layer !=
				dependencies[j].Layer {
				return dependencies[i].Layer <
					dependencies[j].Layer
			}

			if dependencies[i].Package !=
				dependencies[j].Package {
				return dependencies[i].Package <
					dependencies[j].Package
			}

			return dependencies[i].ImportPath <
				dependencies[j].ImportPath
		},
	)

	return contexts, dependencies, nil
}

func classifyPath(
	path string,
) (contextName string, layerName string, ok bool) {
	rel, err := filepath.Rel(
		sourceRoot,
		path,
	)
	if err != nil {
		return "", "", false
	}

	parts := strings.Split(
		filepath.ToSlash(rel),
		"/",
	)

	// Ожидаем:
	//
	// internalDDD/
	//   <bounded-context>/
	//     <layer>/
	//       ...
	//
	// Например:
	//
	// internalDDD/execution/domain/execution.go

	if len(parts) < 3 {
		return "", "", false
	}

	contextName = parts[0]
	layerName = parts[1]

	if !isKnownLayer(layerName) {
		return "", "", false
	}

	return contextName, layerName, true
}

func isKnownLayer(
	layer string,
) bool {
	for _, known := range layers {
		if layer == known {
			return true
		}
	}

	return false
}

func analyzeFile(
	fset *token.FileSet,
	file *ast.File,
	path string,
) fileInfo {
	info := fileInfo{
		Path:    filepath.ToSlash(path),
		Package: file.Name.Name,
	}

	for _, importSpec := range file.Imports {
		importPath := strings.Trim(
			importSpec.Path.Value,
			`"`,
		)

		if importPath == "" {
			continue
		}

		info.Imports = append(
			info.Imports,
			importPath,
		)
	}

	sort.Strings(info.Imports)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}

			for _, spec := range d.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				switch typeSpec.Type.(type) {
				case *ast.InterfaceType:
					info.Interfaces = append(
						info.Interfaces,
						formatNode(
							fset,
							typeSpec,
						),
					)

				case *ast.StructType:
					info.Structs = append(
						info.Structs,
						formatNode(
							fset,
							typeSpec,
						),
					)
				}
			}

		case *ast.FuncDecl:
			signature := formatNode(
				fset,
				d,
			)

			if signature == "" {
				continue
			}

			if d.Recv == nil {
				if strings.HasPrefix(
					d.Name.Name,
					"New",
				) {
					info.Constructors = append(
						info.Constructors,
						signature,
					)
				} else {
					info.Functions = append(
						info.Functions,
						signature,
					)
				}

				continue
			}

			info.Methods = append(
				info.Methods,
				signature,
			)
		}
	}

	sort.Strings(info.Interfaces)
	sort.Strings(info.Structs)
	sort.Strings(info.Constructors)
	sort.Strings(info.Functions)
	sort.Strings(info.Methods)

	return info
}

func prepareOutputDirectory() error {
	if err := os.MkdirAll(
		outputRoot,
		0o755,
	); err != nil {
		return err
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		return err
	}

	// Полностью очищаем старый сгенерированный контекст.
	for _, entry := range entries {
		path := filepath.Join(
			outputRoot,
			entry.Name(),
		)

		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

func generateContexts(
	contexts []boundedContext,
	dependencies []dependencyInfo,
) error {
	for _, bc := range contexts {
		if err := generateBoundedContext(
			bc,
			dependencies,
		); err != nil {
			return err
		}
	}

	return nil
}

func generateBoundedContext(
	bc boundedContext,
	dependencies []dependencyInfo,
) error {
	bcDir := filepath.Join(
		outputRoot,
		bc.Name,
	)

	if err := os.MkdirAll(
		bcDir,
		0o755,
	); err != nil {
		return err
	}

	if err := writeReadme(
		bcDir,
		bc,
	); err != nil {
		return err
	}

	for _, layerName := range layers {
		layer, ok := bc.Layers[layerName]
		if !ok {
			continue
		}

		path := filepath.Join(
			bcDir,
			layerName+".md",
		)

		if err := writeLayer(
			path,
			bc,
			*layer,
		); err != nil {
			return err
		}
	}

	dependenciesPath := filepath.Join(
		bcDir,
		"dependencies.md",
	)

	return writeDependencies(
		dependenciesPath,
		bc,
		dependencies,
	)
}

func writeReadme(
	bcDir string,
	bc boundedContext,
) error {
	var out bytes.Buffer

	fmt.Fprintf(
		&out,
		"# Bounded Context: %s\n\n",
		bc.Name,
	)

	fmt.Fprintf(
		&out,
		"Generated: %s\n\n",
		time.Now().Format(
			"2006-01-02 15:04:05",
		),
	)

	fmt.Fprintln(
		&out,
		"This file is generated automatically from the Go source code.",
	)
	fmt.Fprintln(
		&out,
		"Do not edit manually.",
	)
	fmt.Fprintln(&out)

	fmt.Fprintln(
		&out,
		"## Layers",
	)
	fmt.Fprintln(&out)

	for _, layerName := range layers {
		layer, ok := bc.Layers[layerName]
		if !ok {
			continue
		}

		fmt.Fprintf(
			&out,
			"- `%s` — %d source files\n",
			layerName,
			len(layer.Files),
		)
	}

	fmt.Fprintln(&out)

	fmt.Fprintln(
		&out,
		"## Source",
	)
	fmt.Fprintln(&out)

	fmt.Fprintf(
		&out,
		"`internalDDD/%s/`\n",
		bc.Name,
	)

	path := filepath.Join(
		bcDir,
		"README.md",
	)

	return os.WriteFile(
		path,
		out.Bytes(),
		0o644,
	)
}

func writeLayer(
	path string,
	bc boundedContext,
	layer layerInfo,
) error {
	var out bytes.Buffer

	fmt.Fprintf(
		&out,
		"# %s — %s\n\n",
		bc.Name,
		layer.Name,
	)

	fmt.Fprintf(
		&out,
		"Source: `internalDDD/%s/%s/`\n\n",
		bc.Name,
		layer.Name,
	)

	fmt.Fprintf(
		&out,
		"Generated: %s\n\n",
		time.Now().Format(
			"2006-01-02 15:04:05",
		),
	)

	fmt.Fprintln(
		&out,
		"This file is generated automatically from the Go source code.",
	)
	fmt.Fprintln(
		&out,
		"Do not edit manually.",
	)
	fmt.Fprintln(&out)

	if len(layer.Files) == 0 {
		fmt.Fprintln(
			&out,
			"No source files.",
		)

		return os.WriteFile(
			path,
			out.Bytes(),
			0o644,
		)
	}

	for _, file := range layer.Files {
		writeFile(
			&out,
			file,
		)
	}

	return os.WriteFile(
		path,
		out.Bytes(),
		0o644,
	)
}

func writeFile(
	out *bytes.Buffer,
	file fileInfo,
) {
	fmt.Fprintf(
		out,
		"## %s\n\n",
		file.Path,
	)

	fmt.Fprintf(
		out,
		"Package: `%s`\n\n",
		file.Package,
	)

	writeNames(
		out,
		"Interfaces",
		file.Interfaces,
	)

	writeNames(
		out,
		"Structs",
		file.Structs,
	)

	writeSignatures(
		out,
		"Constructors",
		file.Constructors,
	)

	writeSignatures(
		out,
		"Functions",
		file.Functions,
	)

	writeSignatures(
		out,
		"Methods",
		file.Methods,
	)
}

func writeNames(
	out *bytes.Buffer,
	title string,
	names []string,
) {
	if len(names) == 0 {
		return
	}

	fmt.Fprintf(
		out,
		"### %s\n\n",
		title,
	)

	for _, name := range names {
		fmt.Fprintln(
			out,
			"```go",
		)

		fmt.Fprintln(
			out,
			name,
		)

		fmt.Fprintln(
			out,
			"```",
		)

		fmt.Fprintln(out)
	}
}

func writeSignatures(
	out *bytes.Buffer,
	title string,
	signatures []string,
) {
	if len(signatures) == 0 {
		return
	}

	fmt.Fprintf(
		out,
		"### %s\n\n",
		title,
	)

	for _, signature := range signatures {
		fmt.Fprintln(
			out,
			"```go",
		)

		fmt.Fprintln(
			out,
			signature,
		)

		fmt.Fprintln(
			out,
			"```",
		)

		fmt.Fprintln(out)
	}
}

func writeDependencies(
	path string,
	bc boundedContext,
	allDependencies []dependencyInfo,
) error {
	var out bytes.Buffer

	fmt.Fprintf(
		&out,
		"# Dependencies: %s\n\n",
		bc.Name,
	)

	fmt.Fprintf(
		&out,
		"Generated: %s\n\n",
		time.Now().Format(
			"2006-01-02 15:04:05",
		),
	)

	fmt.Fprintln(
		&out,
		"This file is generated automatically from Go imports.",
	)
	fmt.Fprintln(
		&out,
		"Do not edit manually.",
	)
	fmt.Fprintln(&out)

	internalPrefix := sourceRoot + "/"

	var internal []dependencyInfo
	var external []dependencyInfo

	for _, dep := range allDependencies {
		if dep.BoundedContext != bc.Name {
			continue
		}

		if strings.HasPrefix(
			dep.ImportPath,
			internalPrefix,
		) {
			internal = append(
				internal,
				dep,
			)
		} else {
			external = append(
				external,
				dep,
			)
		}
	}

	fmt.Fprintln(
		&out,
		"## Internal dependencies",
	)
	fmt.Fprintln(&out)

	writeInternalDependencies(
		&out,
		bc,
		internal,
	)

	fmt.Fprintln(
		&out,
		"## Bounded Context dependencies",
	)
	fmt.Fprintln(&out)

	writeBoundedContextDependencies(
		&out,
		bc,
		internal,
	)

	fmt.Fprintln(
		&out,
		"## External dependencies",
	)
	fmt.Fprintln(&out)

	writeExternalDependencies(
		&out,
		external,
	)

	return os.WriteFile(
		path,
		out.Bytes(),
		0o644,
	)
}

func writeInternalDependencies(
	out *bytes.Buffer,
	bc boundedContext,
	dependencies []dependencyInfo,
) {
	if len(dependencies) == 0 {
		fmt.Fprintln(
			out,
			"No internal dependencies.",
		)
		fmt.Fprintln(out)
		return
	}

	type layerKey struct {
		layer   string
		pkg     string
		importP string
	}

	unique := make(
		map[layerKey]struct{},
	)

	for _, dep := range dependencies {
		key := layerKey{
			layer:   dep.Layer,
			pkg:     dep.Package,
			importP: dep.ImportPath,
		}

		unique[key] = struct{}{}
	}

	keys := make(
		[]layerKey,
		0,
		len(unique),
	)

	for key := range unique {
		keys = append(keys, key)
	}

	sort.Slice(
		keys,
		func(i, j int) bool {
			if keys[i].layer != keys[j].layer {
				return keys[i].layer < keys[j].layer
			}

			if keys[i].pkg != keys[j].pkg {
				return keys[i].pkg < keys[j].pkg
			}

			return keys[i].importP < keys[j].importP
		},
	)

	currentLayer := ""

	for _, key := range keys {
		if key.layer != currentLayer {
			currentLayer = key.layer

			fmt.Fprintf(
				out,
				"### %s\n\n",
				currentLayer,
			)
		}

		fmt.Fprintf(
			out,
			"- `%s` → `%s`\n",
			key.pkg,
			key.importP,
		)
	}

	fmt.Fprintln(out)

	_ = bc
}

func writeBoundedContextDependencies(
	out *bytes.Buffer,
	bc boundedContext,
	dependencies []dependencyInfo,
) {
	type contextKey struct {
		context string
		layer   string
	}

	unique := make(
		map[contextKey]struct{},
	)

	prefix := sourceRoot + "/"

	for _, dep := range dependencies {
		relative := strings.TrimPrefix(
			dep.ImportPath,
			prefix,
		)

		parts := strings.Split(
			relative,
			"/",
		)

		if len(parts) < 2 {
			continue
		}

		contextName := parts[0]

		if contextName == bc.Name {
			continue
		}

		layerName := parts[1]

		unique[contextKey{
			context: contextName,
			layer:   layerName,
		}] = struct{}{}
	}

	if len(unique) == 0 {
		fmt.Fprintln(
			out,
			"No dependencies on other Bounded Contexts.",
		)
		fmt.Fprintln(out)
		return
	}

	keys := make(
		[]contextKey,
		0,
		len(unique),
	)

	for key := range unique {
		keys = append(keys, key)
	}

	sort.Slice(
		keys,
		func(i, j int) bool {
			if keys[i].context != keys[j].context {
				return keys[i].context < keys[j].context
			}

			return keys[i].layer < keys[j].layer
		},
	)

	currentContext := ""

	for _, key := range keys {
		if key.context != currentContext {
			currentContext = key.context

			fmt.Fprintf(
				out,
				"### %s\n\n",
				currentContext,
			)
		}

		fmt.Fprintf(
			out,
			"- `%s`\n",
			key.layer,
		)
	}

	fmt.Fprintln(out)
}

func writeExternalDependencies(
	out *bytes.Buffer,
	dependencies []dependencyInfo,
) {
	if len(dependencies) == 0 {
		fmt.Fprintln(
			out,
			"No external dependencies.",
		)
		fmt.Fprintln(out)
		return
	}

	type packageKey struct {
		layer   string
		pkg     string
		importP string
	}

	unique := make(
		map[packageKey]struct{},
	)

	for _, dep := range dependencies {
		unique[packageKey{
			layer:   dep.Layer,
			pkg:     dep.Package,
			importP: dep.ImportPath,
		}] = struct{}{}
	}

	keys := make(
		[]packageKey,
		0,
		len(unique),
	)

	for key := range unique {
		keys = append(keys, key)
	}

	sort.Slice(
		keys,
		func(i, j int) bool {
			if keys[i].layer != keys[j].layer {
				return keys[i].layer < keys[j].layer
			}

			if keys[i].importP != keys[j].importP {
				return keys[i].importP < keys[j].importP
			}

			return keys[i].pkg < keys[j].pkg
		},
	)

	currentLayer := ""

	for _, key := range keys {
		if key.layer != currentLayer {
			currentLayer = key.layer

			fmt.Fprintf(
				out,
				"### %s\n\n",
				currentLayer,
			)
		}

		fmt.Fprintf(
			out,
			"- `%s`",
			key.importP,
		)

		if key.pkg != "" {
			fmt.Fprintf(
				out,
				" — package `%s`",
				key.pkg,
			)
		}

		fmt.Fprintln(out)
	}

	fmt.Fprintln(out)
}

func formatNode(
	fset *token.FileSet,
	node ast.Node,
) string {
	var buf bytes.Buffer

	if err := format.Node(
		&buf,
		fset,
		node,
	); err != nil {
		return ""
	}

	return buf.String()
}
