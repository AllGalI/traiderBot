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

type Visibility string

const (
	VisibilityExported   Visibility = "exported"
	VisibilityUnexported Visibility = "unexported"
)

type Field struct {
	Name       string
	Type       string
	Visibility Visibility
	Embedded   bool
	Comment    string
}

type Method struct {
	Name       string
	Receiver   string
	Signature  string
	Visibility Visibility
	Comment    string
}

type Function struct {
	Name       string
	Signature  string
	Visibility Visibility
	Comment    string
}

type Struct struct {
	Name       string
	Visibility Visibility
	Comment    string
	Fields     []Field
	Package    string
	File       string
}

type Interface struct {
	Name       string
	Visibility Visibility
	Comment    string
	Methods    []Method
	Package    string
	File       string
}

type DefinedType struct {
	Name       string
	Type       string
	Visibility Visibility
	Comment    string
	Package    string
	File       string
}

type Import struct {
	Path  string
	Alias string
}

type Package struct {
	Name         string
	Path         string
	Files        []string
	Dependencies []string
}

type Relation struct {
	FromPackage string
	FromType    string
	ToPackage   string
	ToType      string
	Kind        string
	Field       string
}

type Dependency struct {
	FromPackage string
	ImportPath  string
	Kind        string
	Target      string
}

type File struct {
	Path         string
	Package      string
	Imports      []Import
	Structs      []Struct
	Interfaces   []Interface
	Functions    []Function
	Methods      []Method
	DefinedTypes []DefinedType
	Constants    []string
	Variables    []string
	IsTest       bool
}

type BoundedContext struct {
	Name         string
	Files        []File
	Packages     []Package
	Relations    []Relation
	Dependencies []Dependency
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

		context := BoundedContext{Name: name, Files: files}
		buildContextModel(&context, root)
		result = append(result, context)
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
) ([]File, error) {
	var result []File

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

func buildContextModel(context *BoundedContext, root string) {
	packageSet := make(map[string]struct{})
	for _, f := range context.Files {
		if f.IsTest {
			continue
		}
		pkgPath := filepath.ToSlash(filepath.Dir(f.Path))
		packageSet[pkgPath] = struct{}{}
		for _, imp := range f.Imports {
			kind, target := classifyDependency(imp.Path, root, context.Name)
			context.Dependencies = append(context.Dependencies, Dependency{FromPackage: f.Package, ImportPath: imp.Path, Kind: kind, Target: target})
		}
	}
	for path := range packageSet {
		name := ""
		for _, f := range context.Files {
			if filepath.ToSlash(filepath.Dir(f.Path)) == path {
				name = f.Package
				break
			}
		}
		p := Package{Name: name, Path: path}
		dependencySet := map[string]struct{}{}
		for _, f := range context.Files {
			if f.IsTest || filepath.ToSlash(filepath.Dir(f.Path)) != path {
				continue
			}
			p.Files = append(p.Files, f.Path)
			for _, dep := range context.Dependencies {
				if dep.FromPackage == f.Package {
					dependencySet[dep.ImportPath] = struct{}{}
				}
			}
		}
		for dep := range dependencySet {
			p.Dependencies = append(p.Dependencies, dep)
		}
		sort.Strings(p.Files)
		sort.Strings(p.Dependencies)
		context.Packages = append(context.Packages, p)
	}
	sort.Slice(context.Packages, func(i, j int) bool { return context.Packages[i].Path < context.Packages[j].Path })
	context.Dependencies = uniqueDependencies(context.Dependencies)
	context.Relations = collectRelations(context, root)
}

func classifyDependency(importPath, root, contextName string) (string, string) {
	marker := filepath.ToSlash(filepath.Base(root)) + "/" + contextName + "/"
	normalized := strings.ReplaceAll(importPath, `\`, "/")
	if strings.Contains(normalized, "/internalDDD/"+contextName+"/") || strings.HasPrefix(normalized, "internalDDD/"+contextName+"/") || strings.Contains(normalized, marker) {
		return "internal package", contextName
	}
	if idx := strings.Index(normalized, "/internalDDD/"); idx >= 0 {
		rest := normalized[idx+len("/internalDDD/"):]
		if slash := strings.IndexByte(rest, '/'); slash > 0 {
			return "other bounded context", rest[:slash]
		}
	}
	if strings.HasPrefix(normalized, "internalDDD/") {
		rest := strings.TrimPrefix(normalized, "internalDDD/")
		if slash := strings.IndexByte(rest, '/'); slash > 0 {
			return "other bounded context", rest[:slash]
		}
	}
	return "external module or standard library", ""
}

func uniqueDependencies(items []Dependency) []Dependency {
	seen := map[string]struct{}{}
	result := make([]Dependency, 0, len(items))
	for _, d := range items {
		key := d.FromPackage + "\x00" + d.ImportPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].FromPackage == result[j].FromPackage {
			return result[i].ImportPath < result[j].ImportPath
		}
		return result[i].FromPackage < result[j].FromPackage
	})
	return result
}

func collectRelations(context *BoundedContext, root string) []Relation {
	typeInfo := map[string]map[string]struct{}{}
	packageAliases := map[string]string{}
	for _, f := range context.Files {
		if f.IsTest {
			continue
		}
		pkgPath := filepath.ToSlash(filepath.Dir(f.Path))
		if typeInfo[pkgPath] == nil {
			typeInfo[pkgPath] = map[string]struct{}{}
		}
		for _, s := range f.Structs {
			typeInfo[pkgPath][s.Name] = struct{}{}
		}
		for _, i := range f.Interfaces {
			typeInfo[pkgPath][i.Name] = struct{}{}
		}
		for _, t := range f.DefinedTypes {
			typeInfo[pkgPath][t.Name] = struct{}{}
		}
		for _, imp := range f.Imports {
			if imp.Alias != "" && imp.Alias != "." && imp.Alias != "_" {
				packageAliases[f.Package+"|"+imp.Alias] = imp.Path
				continue
			}
			if imp.Alias == "" {
				parts := strings.Split(strings.TrimRight(imp.Path, "/"), "/")
				if len(parts) > 0 {
					packageAliases[f.Package+"|"+parts[len(parts)-1]] = imp.Path
				}
			}
		}
	}
	var relations []Relation
	for _, f := range context.Files {
		if f.IsTest {
			continue
		}
		fromPath := filepath.ToSlash(filepath.Dir(f.Path))
		for _, s := range f.Structs {
			for _, field := range s.Fields {
				if target, ok := localTypeReference(field.Type, f, typeInfo, packageAliases); ok {
					relations = append(relations, Relation{FromPackage: fromPath, FromType: s.Name, ToPackage: target[0], ToType: target[1], Kind: "field type", Field: field.Name})
				}
			}
		}
	}
	seen := map[string]struct{}{}
	result := make([]Relation, 0, len(relations))
	for _, r := range relations {
		key := r.FromPackage + "|" + r.FromType + "|" + r.ToPackage + "|" + r.ToType + "|" + r.Field
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		return a.FromPackage+a.FromType+a.Field < b.FromPackage+b.FromType+b.Field
	})
	_ = root
	return result
}

func localTypeReference(typeExpr string, file File, types map[string]map[string]struct{}, aliases map[string]string) ([2]string, bool) {
	clean := strings.TrimLeft(typeExpr, "*[]")
	for strings.HasPrefix(clean, "map[") {
		return [2]string{}, false
	}
	clean = strings.TrimSpace(clean)
	if dot := strings.IndexByte(clean, '.'); dot > 0 {
		alias, name := clean[:dot], clean[dot+1:]
		path, ok := aliases[file.Package+"|"+alias]
		if !ok {
			return [2]string{}, false
		}
		for pkgPath, names := range types {
			if pathMatchesPackage(pkgPath, path) {
				if _, ok := names[name]; ok {
					return [2]string{pkgPath, name}, true
				}
			}
		}
		return [2]string{}, false
	}
	if _, ok := types[filepath.ToSlash(filepath.Dir(file.Path))][clean]; ok {
		return [2]string{filepath.ToSlash(filepath.Dir(file.Path)), clean}, true
	}
	return [2]string{}, false
}

func pathMatchesPackage(pkgPath, importPath string) bool {
	base := filepath.ToSlash(filepath.Base(pkgPath))
	return strings.HasSuffix(importPath, "/"+base) || strings.HasSuffix(importPath, base)
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
) File {
	summary := File{
		Path:   filepath.ToSlash(path),
		IsTest: isTest,
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
		if imp.Path == nil {
			continue
		}
		item := Import{Path: strings.Trim(imp.Path.Value, `"`)}
		if imp.Name != nil {
			item.Alias = imp.Name.Name
		}
		summary.Imports = append(summary.Imports, item)
	}

	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			summarizeGenDecl(node, &summary)
		case *ast.FuncDecl:
			summarizeFuncDecl(node, &summary)
		}
	}

	normalizeSummary(&summary)
	_ = root
	return summary
}

func summarizeGenDecl(decl *ast.GenDecl, summary *File) {
	for _, spec := range decl.Specs {
		switch decl.Tok {
		case token.CONST, token.VAR:
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if decl.Tok == token.CONST {
					summary.Constants = append(summary.Constants, name.Name)
				} else {
					summary.Variables = append(summary.Variables, name.Name)
				}
			}
		case token.TYPE:
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			name := typeSpec.Name.Name
			comment := docText(typeSpec.Doc)
			if comment == "" {
				comment = docText(decl.Doc)
			}
			visibility := symbolVisibility(name)
			switch typeExpr := typeSpec.Type.(type) {
			case *ast.StructType:
				summary.Structs = append(summary.Structs, Struct{
					Name: name, Visibility: visibility, Comment: comment,
					Fields: formatStructFields(typeExpr), Package: summary.Package, File: summary.Path,
				})
			case *ast.InterfaceType:
				summary.Interfaces = append(summary.Interfaces, Interface{
					Name: name, Visibility: visibility, Comment: comment,
					Methods: formatInterfaceMethods(typeExpr), Package: summary.Package, File: summary.Path,
				})
			default:
				summary.DefinedTypes = append(summary.DefinedTypes, DefinedType{
					Name: name, Type: exprString(typeExpr), Visibility: visibility, Comment: comment,
					Package: summary.Package, File: summary.Path,
				})
			}
		}
	}
}

func summarizeFuncDecl(decl *ast.FuncDecl, summary *File) {
	item := Method{
		Name:       decl.Name.Name,
		Signature:  formatFuncSignature(decl),
		Visibility: symbolVisibility(decl.Name.Name),
		Comment:    docText(decl.Doc),
	}
	if decl.Recv == nil {
		summary.Functions = append(summary.Functions, Function{
			Name: item.Name, Signature: item.Signature, Visibility: item.Visibility, Comment: item.Comment,
		})
		return
	}
	if len(decl.Recv.List) > 0 {
		item.Receiver = exprString(decl.Recv.List[0].Type)
	}
	summary.Methods = append(summary.Methods, item)
}

func formatStructFields(structType *ast.StructType) []Field {
	if structType.Fields == nil {
		return nil
	}
	var result []Field
	for _, field := range structType.Fields.List {
		comment := docText(field.Doc)
		if comment == "" {
			comment = docText(field.Comment)
		}
		if len(field.Names) == 0 {
			name := embeddedFieldName(field.Type)
			result = append(result, Field{Name: name, Type: exprString(field.Type), Visibility: symbolVisibility(name), Embedded: true, Comment: comment})
			continue
		}
		for _, name := range field.Names {
			result = append(result, Field{Name: name.Name, Type: exprString(field.Type), Visibility: symbolVisibility(name.Name), Comment: comment})
		}
	}
	return result
}

func formatInterfaceMethods(interfaceType *ast.InterfaceType) []Method {
	if interfaceType.Methods == nil {
		return nil
	}
	var result []Method
	for _, field := range interfaceType.Methods.List {
		comment := docText(field.Doc)
		if comment == "" {
			comment = docText(field.Comment)
		}
		switch typeExpr := field.Type.(type) {
		case *ast.FuncType:
			if len(field.Names) == 0 {
				result = append(result, Method{Name: exprString(typeExpr), Signature: exprString(typeExpr), Visibility: VisibilityExported, Comment: comment})
				continue
			}
			for _, name := range field.Names {
				result = append(result, Method{Name: name.Name, Signature: name.Name + formatFuncType(typeExpr), Visibility: symbolVisibility(name.Name), Comment: comment})
			}
		default:
			result = append(result, Method{Name: exprString(typeExpr), Signature: exprString(typeExpr), Visibility: VisibilityExported, Comment: comment})
		}
	}
	return result
}

func embeddedFieldName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return embeddedFieldName(node.X)
	case *ast.SelectorExpr:
		return node.Sel.Name
	default:
		return exprString(expr)
	}
}

func symbolVisibility(name string) Visibility {
	if name != "" && ast.IsExported(name) {
		return VisibilityExported
	}
	return VisibilityUnexported
}

func docText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return strings.TrimSpace(group.Text())
}

func normalizeSummary(summary *File) {
	sort.Slice(summary.Imports, func(i, j int) bool { return summary.Imports[i].Path < summary.Imports[j].Path })
	sort.Slice(summary.Structs, func(i, j int) bool { return summary.Structs[i].Name < summary.Structs[j].Name })
	sort.Slice(summary.Interfaces, func(i, j int) bool { return summary.Interfaces[i].Name < summary.Interfaces[j].Name })
	sort.Slice(summary.DefinedTypes, func(i, j int) bool { return summary.DefinedTypes[i].Name < summary.DefinedTypes[j].Name })
	sort.Slice(summary.Functions, func(i, j int) bool { return summary.Functions[i].Name < summary.Functions[j].Name })
	sort.Slice(summary.Methods, func(i, j int) bool {
		if summary.Methods[i].Receiver == summary.Methods[j].Receiver {
			return summary.Methods[i].Name < summary.Methods[j].Name
		}
		return summary.Methods[i].Receiver < summary.Methods[j].Receiver
	})
	sort.Strings(summary.Constants)
	sort.Strings(summary.Variables)
}

func formatFuncSignature(decl *ast.FuncDecl) string {
	receiver := ""
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		receiver = fmt.Sprintf("(%s) ", exprString(decl.Recv.List[0].Type))
	}
	return fmt.Sprintf("func %s%s%s", receiver, decl.Name.Name, formatFuncType(decl.Type))
}

func formatFuncType(funcType *ast.FuncType) string {
	var params []string
	var results []string
	if funcType.Params != nil {
		params = formatFieldList(funcType.Params)
	}
	if funcType.Results != nil {
		results = formatFieldList(funcType.Results)
	}
	signature := "(" + strings.Join(params, ", ") + ")"
	switch len(results) {
	case 0:
		return signature
	case 1:
		return signature + " " + results[0]
	default:
		return signature + " (" + strings.Join(results, ", ") + ")"
	}
}

func formatFieldList(fields *ast.FieldList) []string {
	var result []string
	for _, field := range fields.List {
		fieldType := exprString(field.Type)
		if len(field.Names) == 0 {
			result = append(result, fieldType)
			continue
		}
		for _, name := range field.Names {
			result = append(result, name.Name+" "+fieldType)
		}
	}
	return result
}

func exprString(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return exprString(node.X) + "." + node.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(node.X)
	case *ast.ArrayType:
		if node.Len != nil {
			return "[" + exprString(node.Len) + "]" + exprString(node.Elt)
		}
		return "[]" + exprString(node.Elt)
	case *ast.MapType:
		return "map[" + exprString(node.Key) + "]" + exprString(node.Value)
	case *ast.InterfaceType:
		return "interface{...}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func" + formatFuncType(node)
	case *ast.Ellipsis:
		return "..." + exprString(node.Elt)
	case *ast.ChanType:
		prefix := "chan "
		if node.Dir == ast.SEND {
			prefix = "chan<- "
		}
		if node.Dir == ast.RECV {
			prefix = "<-chan "
		}
		return prefix + exprString(node.Value)
	case *ast.ParenExpr:
		return "(" + exprString(node.X) + ")"
	case *ast.BasicLit:
		return node.Value
	case *ast.IndexExpr:
		return exprString(node.X) + "[" + exprString(node.Index) + "]"
	case *ast.IndexListExpr:
		var indices []string
		for _, index := range node.Indices {
			indices = append(indices, exprString(index))
		}
		return exprString(node.X) + "[" + strings.Join(indices, ", ") + "]"
	default:
		return "..."
	}
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

func writeBoundedContext(output string, context BoundedContext) error {
	dir := filepath.Join(output, context.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create bounded context %q: %w", dir, err)
	}
	readmePath := filepath.Join(dir, "README.md")
	file, err := os.Create(readmePath)
	if err != nil {
		return fmt.Errorf("create %q: %w", readmePath, err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	renderContext(writer, context)
	return writer.Flush()
}

func renderContext(writer *bufio.Writer, context BoundedContext) {
	fmt.Fprintf(writer, "# Bounded Context: %s\n\n", context.Name)
	fmt.Fprintln(writer, "This is an observed code structure snapshot, not a statement of architectural intent.")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Implementation bodies are intentionally omitted. Tests are excluded from the main architectural snapshot.")
	fmt.Fprintln(writer)

	fmt.Fprintln(writer, "## Overview")
	fmt.Fprintf(writer, "Observed packages: %d\n\n", len(context.Packages))
	fmt.Fprintln(writer, "The snapshot describes symbols, package dependencies and type relations observed in the source code. Exported symbols are not automatically considered the architectural contract of the Bounded Context.")
	fmt.Fprintln(writer)

	fmt.Fprintln(writer, "## Layers")
	layers := map[string][]string{}
	for _, pkg := range context.Packages {
		layer := packageLayer(pkg.Path)
		layers[layer] = append(layers[layer], pkg.Path)
	}
	for _, layer := range []string{"domain", "application", "infrastructure", "other"} {
		if len(layers[layer]) == 0 {
			continue
		}
		fmt.Fprintf(writer, "### %s\n\n", titleCase(layer))
		sort.Strings(layers[layer])
		for _, path := range layers[layer] {
			fmt.Fprintf(writer, "- `%s`\n", path)
		}
		fmt.Fprintln(writer)
	}

	fmt.Fprintln(writer, "## Exported API")
	fmt.Fprintln(writer, "Exported symbols observed in code. This section is descriptive; it does not assert that these symbols form the architectural contract.")
	fmt.Fprintln(writer)
	writeExportedAPI(writer, context)

	fmt.Fprintln(writer, "## Domain Model")
	writeDomainModel(writer, context)

	fmt.Fprintln(writer, "## Internal Architecture")
	writeInternalArchitecture(writer, context)

	fmt.Fprintln(writer, "## Dependencies")
	writeDependencies(writer, context)

	fmt.Fprintln(writer, "## Source Layout")
	writeSourceLayout(writer, context)
}

func writeExportedAPI(w *bufio.Writer, c BoundedContext) {
	wrote := false
	for _, f := range c.Files {
		if f.IsTest {
			continue
		}
		for _, i := range f.Interfaces {
			if i.Visibility == VisibilityExported {
				fmt.Fprintf(w, "- interface `%s` (package `%s`)\n", i.Name, i.Package)
				wrote = true
			}
		}
		for _, s := range f.Structs {
			if s.Visibility == VisibilityExported {
				fmt.Fprintf(w, "- struct `%s` (package `%s`)\n", s.Name, s.Package)
				wrote = true
			}
		}
		for _, t := range f.DefinedTypes {
			if t.Visibility == VisibilityExported {
				fmt.Fprintf(w, "- type `%s` (package `%s`)\n", t.Name, t.Package)
				wrote = true
			}
		}
		for _, fn := range f.Functions {
			if fn.Visibility == VisibilityExported {
				writeFunction(w, fn)
				wrote = true
			}
		}
		for _, m := range f.Methods {
			if m.Visibility == VisibilityExported {
				writeMethod(w, m)
				wrote = true
			}
		}
	}
	if !wrote {
		fmt.Fprintln(w, "No exported symbols observed.")
		fmt.Fprintln(w)
	}
}

func writeDomainModel(w *bufio.Writer, c BoundedContext) {
	found := false
	for _, f := range c.Files {
		if f.IsTest || packageLayer(f.Path) != "domain" {
			continue
		}
		for _, s := range f.Structs {
			writeStruct(w, s)
			found = true
		}
		for _, t := range f.DefinedTypes {
			writeDefinedType(w, t)
			found = true
		}
		for _, i := range f.Interfaces {
			writeInterface(w, i)
			found = true
		}
	}
	writeRelations(w, c, "domain", "")
	if !found {
		fmt.Fprintln(w, "No domain types observed.")
	}
}

func writeInternalArchitecture(w *bufio.Writer, c BoundedContext) {
	found := false
	for _, f := range c.Files {
		if f.IsTest || packageLayer(f.Path) == "domain" {
			continue
		}
		for _, s := range f.Structs {
			writeStruct(w, s)
			found = true
		}
		for _, i := range f.Interfaces {
			writeInterface(w, i)
			found = true
		}
		for _, t := range f.DefinedTypes {
			writeDefinedType(w, t)
			found = true
		}
		for _, fn := range f.Functions {
			writeFunction(w, fn)
			found = true
		}
		for _, m := range f.Methods {
			writeMethod(w, m)
			found = true
		}
	}
	writeRelations(w, c, "", "domain")
	if !found {
		fmt.Fprintln(w, "No internal symbols observed.")
	}
}

func writeStruct(w *bufio.Writer, s Struct) {
	fmt.Fprintf(w, "### %s (`%s`)\n\n", s.Name, s.Visibility)
	fmt.Fprintf(w, "Package: `%s`\n\n", s.Package)
	if s.Comment != "" {
		fmt.Fprintf(w, "%s\n\n", s.Comment)
	}
	if len(s.Fields) == 0 {
		fmt.Fprintln(w, "Fields: none")
		return
	}
	fmt.Fprintln(w, "Fields:")
	for _, f := range s.Fields {
		embedded := ""
		if f.Embedded {
			embedded = " embedded"
		}
		fmt.Fprintf(w, "- `%s` `%s` — %s%s", f.Name, f.Type, f.Visibility, embedded)
		if f.Comment != "" {
			fmt.Fprintf(w, " — %s", f.Comment)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func writeInterface(w *bufio.Writer, i Interface) {
	fmt.Fprintf(w, "### %s (`%s`)\n\n", i.Name, i.Visibility)
	fmt.Fprintf(w, "Package: `%s`\n\n", i.Package)
	if i.Comment != "" {
		fmt.Fprintf(w, "%s\n\n", i.Comment)
	}
	if len(i.Methods) == 0 {
		fmt.Fprintln(w, "Methods: none")
		return
	}
	fmt.Fprintln(w, "Methods:")
	for _, m := range i.Methods {
		fmt.Fprintf(w, "- `%s` — %s", m.Signature, m.Visibility)
		if m.Comment != "" {
			fmt.Fprintf(w, " — %s", m.Comment)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func writeDefinedType(w *bufio.Writer, t DefinedType) {
	fmt.Fprintf(w, "### %s (`%s`)\n\n", t.Name, t.Visibility)
	fmt.Fprintf(w, "Defined as `%s`. Package: `%s`\n\n", t.Type, t.Package)
	if t.Comment != "" {
		fmt.Fprintf(w, "%s\n\n", t.Comment)
	}
}

func writeFunction(w *bufio.Writer, f Function) {
	fmt.Fprintf(w, "- `%s` — %s\n", f.Signature, f.Visibility)
	if f.Comment != "" {
		fmt.Fprintf(w, "  - %s\n", f.Comment)
	}
	fmt.Fprintln(w)
}
func writeMethod(w *bufio.Writer, m Method) {
	fmt.Fprintf(w, "- `%s` — %s\n", m.Signature, m.Visibility)
	if m.Comment != "" {
		fmt.Fprintf(w, "  - %s\n", m.Comment)
	}
	fmt.Fprintln(w)
}

func writeRelations(w *bufio.Writer, c BoundedContext, fromLayer, toLayer string) {
	wrote := false
	for _, r := range c.Relations {
		if fromLayer != "" && packageLayerFromPackage(r.FromPackage) != fromLayer {
			continue
		}
		if toLayer != "" && packageLayerFromPackage(r.ToPackage) != toLayer {
			continue
		}
		fmt.Fprintf(w, "- `%s.%s` -> `%s.%s`", r.FromPackage, r.FromType, r.ToPackage, r.ToType)
		if r.Field != "" {
			fmt.Fprintf(w, " via field `%s`", r.Field)
		}
		fmt.Fprintf(w, " (%s)\n", r.Kind)
		wrote = true
	}
	if wrote {
		fmt.Fprintln(w)
	}
}

func writeDependencies(w *bufio.Writer, c BoundedContext) {
	if len(c.Dependencies) == 0 {
		fmt.Fprintln(w, "No dependencies observed.")
		return
	}
	for _, d := range c.Dependencies {
		fmt.Fprintf(w, "- `%s` -> `%s` — %s", d.FromPackage, d.ImportPath, d.Kind)
		if d.Target != "" {
			fmt.Fprintf(w, " (`%s`)", d.Target)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func writeSourceLayout(w *bufio.Writer, c BoundedContext) {
	for _, p := range c.Packages {
		fmt.Fprintf(w, "### `%s`\n\n", p.Path)
		for _, f := range p.Files {
			fmt.Fprintf(w, "- `%s`\n", f)
		}
		fmt.Fprintln(w)
	}
}

func packageLayer(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		switch p {
		case "domain", "application", "infrastructure":
			return p
		}
	}
	return "other"
}

func packageLayerFromPackage(path string) string { return packageLayer(path) }
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
