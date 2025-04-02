package codegen

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/imports"
)

// Embed the templates directory
//
//go:embed templates
var templates embed.FS

// Parser uses the provided ParseContext to generate Go code for the API.
type Parser struct {
	tpl *template.Template
	ctx *ParseContext
	cfg Configuration
}

type EnumContext struct {
	Enums      []EnumDefinition
	Imports    []string
	Config     Configuration
	WithHeader bool
}

// TplTypeContext is the context passed to templates to generate code for type definitions.
type TplTypeContext struct {
	Types        []TypeDefinition
	Imports      []string
	SpecLocation string
	Config       Configuration
	WithHeader   bool
}

// TplOperationsContext is the context passed to templates to generate client code.
type TplOperationsContext struct {
	Operations []OperationDefinition
	Imports    []string
	Config     Configuration
	WithHeader bool
}

// NewParser creates a new Parser with the provided ParseConfig and ParseContext.
func NewParser(cfg Configuration, ctx *ParseContext) (*Parser, error) {
	tpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	// load user-provided templates. Will Override built-in versions.
	for name, tplContents := range cfg.UserTemplates {
		userTpl := tpl.New(name)

		txt, err := getUserTemplateText(tplContents)
		if err != nil {
			return nil, fmt.Errorf("error loading user-provided template %q: %w", name, err)
		}

		_, err = userTpl.Parse(txt)
		if err != nil {
			return nil, fmt.Errorf("error parsing user-provided template %q: %w", name, err)
		}
	}

	return &Parser{
		tpl: tpl,
		ctx: ctx,
		cfg: cfg,
	}, nil
}

// Parse generates Go code for the API using the provided ParseContext.
// It returns a map of generated code for each type of definition.
func (p *Parser) Parse() (map[string]string, error) {
	typesOut := make(map[string]string)

	useSingleOutput := p.cfg.UseSingleOutput
	withHeader := !useSingleOutput
	if useSingleOutput {
		out, err := p.ParseTemplates([]string{"header-inc.tmpl"}, EnumContext{
			Imports:    p.ctx.Imports,
			Config:     p.cfg,
			WithHeader: true,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating code for header: %w", err)
		}
		typesOut["header"] = out
	}

	if len(p.ctx.Operations) > 0 {
		opsCtx := &TplOperationsContext{
			Operations: p.ctx.Operations,
			Imports:    p.ctx.Imports,
			Config:     p.cfg,
			WithHeader: withHeader,
		}
		for _, tmpl := range []string{"client", "client-options"} {
			out, err := p.ParseTemplates([]string{tmpl + ".tmpl"}, opsCtx)
			if err != nil {
				return nil, fmt.Errorf("error generating code for client: %w", err)
			}
			typesOut[strcase.ToSnake(tmpl)] = FormatCode(out)
		}
	}

	if len(p.ctx.Enums) > 0 {
		out, err := p.ParseTemplates([]string{"enums.tmpl"}, EnumContext{
			Enums:      p.ctx.Enums,
			Imports:    p.ctx.Imports,
			Config:     p.cfg,
			WithHeader: withHeader,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating code for type enums: %w", err)
		}
		typesOut["enums"] = FormatCode(out)
	}

	for sl, tds := range p.ctx.TypeDefinitions {
		if len(tds) == 0 {
			continue
		}
		typesCtx := &TplTypeContext{
			Types:        tds,
			SpecLocation: string(sl),
			Imports:      p.ctx.Imports,
			Config:       p.cfg,
			WithHeader:   withHeader,
		}
		out, err := p.ParseTemplates([]string{"types.tmpl"}, typesCtx)
		if err != nil {
			return nil, fmt.Errorf("error generating code for %s type definitions: %w", sl, err)
		}
		typesOut[getSpecLocationOutName(sl)] = FormatCode(out)
	}

	if len(p.ctx.AdditionalTypes) > 0 {
		out, err := p.ParseTemplates([]string{"additional-properties.tmpl"}, &TplTypeContext{
			Types:      p.ctx.AdditionalTypes,
			Imports:    p.ctx.Imports,
			Config:     p.cfg,
			WithHeader: withHeader,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating code for additional properties: %w", err)
		}
		typesOut["additional"] = FormatCode(out)
	}

	if len(p.ctx.UnionTypes) > 0 {
		out, err := p.ParseTemplates([]string{"types.tmpl", "union.tmpl"}, &TplTypeContext{
			Types:        p.ctx.UnionTypes,
			SpecLocation: "union",
			Imports:      p.ctx.Imports,
			Config:       p.cfg,
			WithHeader:   withHeader,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating code for union types: %w", err)
		}
		typesOut["unions"] = FormatCode(out)
	}

	if len(p.ctx.UnionWithAdditionalTypes) > 0 {
		out, err := p.ParseTemplates([]string{"union-and-additional-properties.tmpl"}, &TplTypeContext{
			Types:        p.ctx.UnionWithAdditionalTypes,
			SpecLocation: "union_with_additional",
			Imports:      p.ctx.Imports,
			Config:       p.cfg,
			WithHeader:   withHeader,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating code for union types with additional properties: %w", err)
		}
		typesOut["unions_with_additional"] = FormatCode(out)
	}

	if useSingleOutput {
		res := ""
		if header, ok := typesOut["header"]; ok {
			res += header + "\n"
			delete(typesOut, "header")
		}

		for _, code := range typesOut {
			res += code + "\n"
		}
		typesOut = map[string]string{"all": res}
	}

	return typesOut, nil
}

// ParseTemplates parses provided templates with the given data and returns the generated code.
func (p *Parser) ParseTemplates(templates []string, data any) (string, error) {
	var generatedTemplates []string
	for _, tmpl := range templates {
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)

		if err := p.tpl.ExecuteTemplate(w, tmpl, data); err != nil {
			return "", fmt.Errorf("error generating %s: %s", tmpl, err)
		}
		if err := w.Flush(); err != nil {
			return "", fmt.Errorf("error flushing output buffer for %s: %s", tmpl, err)
		}
		generatedTemplates = append(generatedTemplates, buf.String())
	}

	return strings.Join(generatedTemplates, "\n"), nil
}

func loadTemplates() (*template.Template, error) {
	tpl := template.New("templates").Funcs(TemplateFunctions)

	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error walking directory %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}

		buf, err := templates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %w", path, err)
		}

		templateName := strings.TrimPrefix(path, "templates/")
		tmpl := tpl.New(templateName)
		_, err = tmpl.Parse(string(buf))
		if err != nil {
			return fmt.Errorf("parsing template '%s': %w", path, err)
		}
		return nil
	})

	return tpl, err
}

// FormatCode formats the provided Go code.
// It optimizes imports and formats the code using gofmt.
func FormatCode(src string) string {
	src = strings.Trim(src, "\n") + "\n"
	if src == "" {
		return src
	}

	res, err := optimizeImports([]byte(src))
	if err != nil {
		return string(res)
	}

	res, err = format.Source(res)
	if err != nil {
		return string(res)
	}

	return sanitizeCode(string(res))
}

// sanitizeCode runs sanitizers across the generated Go code to ensure the
// generated code will be able to compile.
func sanitizeCode(src string) string {
	// remove any byte-order-marks which break Go-Code
	// See: https://groups.google.com/forum/#!topic/golang-nuts/OToNIPdfkks
	return strings.ReplaceAll(src, "\uFEFF", "")
}

func optimizeImports(src []byte) ([]byte, error) {
	outBytes, err := imports.Process("gen.go", src, nil)
	if err != nil {
		return src, err
	}
	return outBytes, nil
}

func getSpecLocationOutName(specLocation SpecLocation) string {
	switch specLocation {
	case SpecLocationPath:
		return "paths"
	case SpecLocationQuery:
		return "queries"
	case SpecLocationHeader:
		return "headers"
	case SpecLocationBody:
		return "payloads"
	case SpecLocationResponse:
		return "responses"
	case SpecLocationSchema:
		return "types"
	case SpecLocationUnion:
		return "unions"
	default:
		return string(specLocation)
	}
}

// getUserTemplateText attempts to retrieve the template text from a passed string or file..
func getUserTemplateText(inputData string) (template string, err error) {
	// if the input data is more than one line, assume its a template and return that data.
	if strings.Contains(inputData, "\n") {
		return inputData, nil
	}

	// load data from file
	data, err := os.ReadFile(inputData)
	// return data if found and loaded
	if err == nil {
		return string(data), nil
	}

	// check for non "not found" errors
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to open file %s: %w", inputData, err)
	}

	return string(data), nil
}
