package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

type config struct {
	domainRef string
	protoRef  string
	name      string
	output    string
	pkgName   string
}

type typeRef struct {
	importPath string
	typeName   string
	pkgName    string
	named      *types.Named
	constants  map[string]*types.Const
}

type enumPair struct {
	domainName string
	protoName  string
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "enumgen:", err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.domainRef, "domain", "", "domain enum as full/import/path.Type")
	flag.StringVar(&cfg.protoRef, "proto", "", "proto enum as full/import/path.Type")
	flag.StringVar(&cfg.name, "name", "", "generated function name prefix")
	flag.StringVar(&cfg.output, "output", "", "generated Go file")
	flag.StringVar(&cfg.pkgName, "package", os.Getenv("GOPACKAGE"), "output package name")
	flag.Parse()

	return cfg
}

func run(cfg config) error {
	if cfg.domainRef == "" {
		return fmt.Errorf("-domain is required")
	}
	if cfg.protoRef == "" {
		return fmt.Errorf("-proto is required")
	}
	if cfg.name == "" {
		return fmt.Errorf("-name is required")
	}
	if cfg.output == "" {
		return fmt.Errorf("-output is required")
	}
	if cfg.pkgName == "" {
		return fmt.Errorf("-package is required when GOPACKAGE is not set")
	}

	domainRef, err := loadType(cfg.domainRef)
	if err != nil {
		return fmt.Errorf("load domain type: %w", err)
	}
	protoRef, err := loadType(cfg.protoRef)
	if err != nil {
		return fmt.Errorf("load proto type: %w", err)
	}

	pairs, unspecified, err := matchConstants(domainRef, protoRef)
	if err != nil {
		return err
	}

	source, err := render(cfg, domainRef, protoRef, pairs, unspecified)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(cfg.output, source, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func loadType(ref string) (typeRef, error) {
	importPath, typeName, err := splitTypeRef(ref)
	if err != nil {
		return typeRef{}, err
	}

	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
	}, importPath)
	if err != nil {
		return typeRef{}, err
	}
	if packages.PrintErrors(loaded) > 0 {
		return typeRef{}, fmt.Errorf("package %q contains errors", importPath)
	}
	if len(loaded) != 1 {
		return typeRef{}, fmt.Errorf("expected one package for %q, got %d", importPath, len(loaded))
	}

	pkg := loaded[0]
	object := pkg.Types.Scope().Lookup(typeName)
	typeObject, ok := object.(*types.TypeName)
	if !ok {
		return typeRef{}, fmt.Errorf("%s is not a named type", ref)
	}
	named, ok := typeObject.Type().(*types.Named)
	if !ok {
		return typeRef{}, fmt.Errorf("%s is not a named type", ref)
	}

	constants := make(map[string]*types.Const)
	for _, name := range pkg.Types.Scope().Names() {
		constant, ok := pkg.Types.Scope().Lookup(name).(*types.Const)
		if ok && types.Identical(constant.Type(), named) {
			constants[name] = constant
		}
	}
	if len(constants) == 0 {
		return typeRef{}, fmt.Errorf("%s has no typed constants", ref)
	}

	return typeRef{
		importPath: importPath,
		typeName:   typeName,
		pkgName:    pkg.Name,
		named:      named,
		constants:  constants,
	}, nil
}

func splitTypeRef(ref string) (string, string, error) {
	index := strings.LastIndexByte(ref, '.')
	if index <= 0 || index == len(ref)-1 {
		return "", "", fmt.Errorf("type reference %q must be full/import/path.Type", ref)
	}
	return ref[:index], ref[index+1:], nil
}

func matchConstants(domainRef, protoRef typeRef) ([]enumPair, string, error) {
	protoPrefix := protoRef.typeName + "_"
	unspecified := ""
	for protoName := range protoRef.constants {
		if strings.HasPrefix(protoName, protoPrefix) &&
			strings.HasSuffix(strings.TrimPrefix(protoName, protoPrefix), "_UNSPECIFIED") {
			if unspecified != "" {
				return nil, "", fmt.Errorf(
					"%s defines multiple unspecified constants: %s and %s",
					protoRef.typeName,
					unspecified,
					protoName,
				)
			}
			unspecified = protoName
		}
	}
	if unspecified == "" {
		return nil, "", fmt.Errorf("%s must define an *_UNSPECIFIED constant", protoRef.typeName)
	}

	pairs := make([]enumPair, 0, len(domainRef.constants))
	for protoName := range protoRef.constants {
		if protoName == unspecified {
			continue
		}
		if !strings.HasPrefix(protoName, protoPrefix) {
			return nil, "", fmt.Errorf("proto constant %s does not start with %s", protoName, protoPrefix)
		}

		domainName := strings.TrimPrefix(protoName, protoPrefix)
		if _, ok := domainRef.constants[domainName]; !ok {
			return nil, "", fmt.Errorf(
				"proto constant %s has no matching %s constant %s",
				protoName,
				domainRef.typeName,
				domainName,
			)
		}
		pairs = append(pairs, enumPair{
			domainName: domainName,
			protoName:  protoName,
		})
	}

	for domainName := range domainRef.constants {
		protoName := protoPrefix + domainName
		if _, ok := protoRef.constants[protoName]; !ok {
			return nil, "", fmt.Errorf(
				"domain constant %s has no matching %s constant %s",
				domainName,
				protoRef.typeName,
				protoName,
			)
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].domainName < pairs[j].domainName
	})
	return pairs, unspecified, nil
}

func render(
	cfg config,
	domainRef typeRef,
	protoRef typeRef,
	pairs []enumPair,
	unspecified string,
) ([]byte, error) {
	domainAlias := importAlias(domainRef.pkgName, "domain", "")
	protoAlias := importAlias(protoRef.pkgName, "proto", domainAlias)

	var source bytes.Buffer
	fmt.Fprintln(&source, "// Code generated by enumgen. DO NOT EDIT.")
	fmt.Fprintln(&source)
	fmt.Fprintf(&source, "package %s\n\n", cfg.pkgName)
	fmt.Fprintln(&source, "import (")
	fmt.Fprintln(&source, "\t\"fmt\"")
	fmt.Fprintf(&source, "\t%s %q\n", domainAlias, domainRef.importPath)
	fmt.Fprintf(&source, "\t%s %q\n", protoAlias, protoRef.importPath)
	fmt.Fprintln(&source, ")")
	fmt.Fprintln(&source)

	fmt.Fprintf(
		&source,
		"func %sToProto(value %s.%s) %s.%s {\n",
		cfg.name,
		domainAlias,
		domainRef.typeName,
		protoAlias,
		protoRef.typeName,
	)
	fmt.Fprintln(&source, "\tswitch value {")
	for _, pair := range pairs {
		fmt.Fprintf(
			&source,
			"\tcase %s.%s:\n\t\treturn %s.%s\n",
			domainAlias,
			pair.domainName,
			protoAlias,
			pair.protoName,
		)
	}
	fmt.Fprintf(&source, "\tdefault:\n\t\treturn %s.%s\n", protoAlias, unspecified)
	fmt.Fprintln(&source, "\t}")
	fmt.Fprintln(&source, "}")
	fmt.Fprintln(&source)

	fmt.Fprintf(
		&source,
		"func %sFromProto(value %s.%s) (%s.%s, error) {\n",
		cfg.name,
		protoAlias,
		protoRef.typeName,
		domainAlias,
		domainRef.typeName,
	)
	fmt.Fprintln(&source, "\tswitch value {")
	for _, pair := range pairs {
		fmt.Fprintf(
			&source,
			"\tcase %s.%s:\n\t\treturn %s.%s, nil\n",
			protoAlias,
			pair.protoName,
			domainAlias,
			pair.domainName,
		)
	}
	fmt.Fprintln(&source, "\tdefault:")
	fmt.Fprintf(&source, "\t\tvar zero %s.%s\n", domainAlias, domainRef.typeName)
	fmt.Fprintf(
		&source,
		"\t\treturn zero, fmt.Errorf(\"unsupported %s: %%v\", value)\n",
		protoRef.typeName,
	)
	fmt.Fprintln(&source, "\t}")
	fmt.Fprintln(&source, "}")

	formatted, err := format.Source(source.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w\n%s", err, source.String())
	}
	return formatted, nil
}

func importAlias(pkgName, fallback, conflict string) string {
	alias := sanitizeIdentifier(pkgName)
	if alias == "" {
		alias = fallback
	}
	if alias == conflict {
		alias += "pkg"
	}
	return alias
}

func sanitizeIdentifier(value string) string {
	var result []rune
	for index, r := range value {
		if unicode.IsLetter(r) || r == '_' || index > 0 && unicode.IsDigit(r) {
			result = append(result, r)
		}
	}
	return string(result)
}
