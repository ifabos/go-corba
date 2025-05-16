package idl

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

// Generator generates Go code from IDL definitions
type Generator struct {
	module      *Module
	outputDir   string
	packageName string
	templates   *template.Template
	includes    []string
}

// NewGenerator creates a new Go code generator for IDL
func NewGenerator(module *Module, outputDir string) *Generator {
	return &Generator{
		module:      module,
		outputDir:   outputDir,
		packageName: "generated",
		includes:    []string{},
	}
}

// SetPackageName sets the Go package name to use for generated code
func (g *Generator) SetPackageName(name string) {
	g.packageName = name
}

// AddInclude adds an import to include in generated files
func (g *Generator) AddInclude(include string) {
	g.includes = append(g.includes, include)
}

// Generate generates Go code for all types in the module and its submodules
func (g *Generator) Generate() error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return err
	}

	// Initialize templates
	if err := g.initTemplates(); err != nil {
		return err
	}

	// Generate code for the module
	return g.generateModule(g.module, g.outputDir)
}

// Initialize the templates used for code generation
func (g *Generator) initTemplates() error {
	g.templates = template.New("idl").Funcs(template.FuncMap{
		"toLower":      strings.ToLower,
		"toUpper":      strings.ToUpper,
		"capitalize":   capitalize,
		"uncapitalize": uncapitalize,
		"goType":       g.goType,
		"inParams":     g.inParams,
		"outParams":    g.outParams,
		"paramList":    g.paramList,
		"argList":      g.argList,
	})

	// Add templates for different IDL types
	for _, tmpl := range []struct {
		name string
		text string
	}{
		{"file", fileTemplate},
		{"interface", interfaceTemplate},
		{"struct", structTemplate},
		{"enum", enumTemplate},
		{"typedef", typedefTemplate},
		{"union", unionTemplate},
	} {
		if _, err := g.templates.New(tmpl.name).Parse(tmpl.text); err != nil {
			return err
		}
	}

	return nil
}

// Generate code for a module and its submodules
func (g *Generator) generateModule(module *Module, dir string) error {
	// Create directory for the module
	moduleDir := dir
	if module.Name != "" {
		moduleDir = filepath.Join(dir, strings.ToLower(module.Name))
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return err
		}
	}

	// Generate types in this module
	for _, typ := range module.Types {
		if err := g.generateType(typ, moduleDir); err != nil {
			return err
		}
	}

	// Generate submodules
	for _, submodule := range module.Submodules {
		if err := g.generateModule(submodule, moduleDir); err != nil {
			return err
		}
	}

	return nil
}

// Generate code for a type
func (g *Generator) generateType(t Type, dir string) error {
	var buf bytes.Buffer
	var err error

	switch typ := t.(type) {
	case *InterfaceType:
		err = g.templates.ExecuteTemplate(&buf, "interface", map[string]interface{}{
			"Package":   g.packageName,
			"Includes":  g.includes,
			"Interface": typ,
		})

	case *StructType:
		err = g.templates.ExecuteTemplate(&buf, "struct", map[string]interface{}{
			"Package":  g.packageName,
			"Includes": g.includes,
			"Struct":   typ,
		})

	case *EnumType:
		err = g.templates.ExecuteTemplate(&buf, "enum", map[string]interface{}{
			"Package":  g.packageName,
			"Includes": g.includes,
			"Enum":     typ,
		})

	case *TypeDef:
		err = g.templates.ExecuteTemplate(&buf, "typedef", map[string]interface{}{
			"Package":  g.packageName,
			"Includes": g.includes,
			"TypeDef":  typ,
		})

	case *UnionType:
		err = g.templates.ExecuteTemplate(&buf, "union", map[string]interface{}{
			"Package":  g.packageName,
			"Includes": g.includes,
			"Union":    typ,
		})

	default:
		// Skip generating for other types
		return nil
	}

	if err != nil {
		return err
	}

	filename := filepath.Join(dir, strings.ToLower(t.GoTypeName())+".go")
	// Use goimports for better formatting and import grouping
	var formatted []byte
	formatted, err = imports.Process(filename, buf.Bytes(), nil)
	if err != nil {
		// Fallback to go/format if goimports fails
		formatted, err = format.Source(buf.Bytes())
		if err != nil {
			// If formatting fails, write the unformatted code for debugging
			unformattedFile := filename + ".unformatted"
			if err := os.WriteFile(unformattedFile, buf.Bytes(), 0644); err != nil {
				return err
			}
			return fmt.Errorf("failed to format generated code for %s: %v", t.TypeName(), err)
		}
	}
	// Write the formatted code to file
	return os.WriteFile(filename, formatted, 0644)
}

// goType converts an IDL type to a Go type
func (g *Generator) goType(t Type) string {
	return t.GoTypeName()
}

// inParams returns the input parameters for an operation
func (g *Generator) inParams(op Operation) string {
	var params []string
	for _, p := range op.Parameters {
		if p.Direction == In || p.Direction == InOut {
			params = append(params, fmt.Sprintf("%s %s", uncapitalize(p.Name), p.Type.GoTypeName()))
		}
	}
	return strings.Join(params, ", ")
}

// outParams returns the output parameters for an operation
func (g *Generator) outParams(op Operation) string {
	var params []string

	// Add return type if not void
	if op.ReturnType.GoTypeName() != "" {
		params = append(params, op.ReturnType.GoTypeName())
	}

	// Add out and inout parameters
	for _, p := range op.Parameters {
		if p.Direction == Out || p.Direction == InOut {
			params = append(params, p.Type.GoTypeName())
		}
	}

	// Add error return
	params = append(params, "error")

	return strings.Join(params, ", ")
}

// paramList returns function parameter list for an operation
func (g *Generator) paramList(op Operation) string {
	var params []string
	for _, p := range op.Parameters {
		params = append(params, fmt.Sprintf("%s %s", uncapitalize(p.Name), p.Type.GoTypeName()))
	}
	return strings.Join(params, ", ")
}

// argList returns argument list for method calls
func (g *Generator) argList(op Operation) string {
	var args []string
	for _, p := range op.Parameters {
		args = append(args, uncapitalize(p.Name))
	}
	return strings.Join(args, ", ")
}

// capitalize returns a string with first letter capitalized
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// uncapitalize returns a string with first letter lowercased
func uncapitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// Template for Go file header
const fileTemplate = `// Code generated by CORBA IDL Go generator. DO NOT EDIT.
package {{.Package}}

import (
	"fmt"
	{{range .Includes}}
	"{{.}}"
	{{end}}
)
`

// Template for Go interface from IDL interface
const interfaceTemplate = `{{template "file" .}}

// {{.Interface.Name}} is a CORBA interface
type {{.Interface.Name}} interface {
	{{range .Interface.Operations}}
	{{.Name}}({{paramList .}}) ({{outParams .}})
	{{end}}
	{{range .Interface.Attributes}}
	Get{{capitalize .Name}}() ({{goType .Type}}, error)
	{{if not .Readonly}}Set{{capitalize .Name}}(value {{goType .Type}}) error{{end}}
	{{end}}
}

// {{.Interface.Name}}Helper provides utility functions for {{.Interface.Name}}
type {{.Interface.Name}}Helper struct{}

// ID returns the repository ID for {{.Interface.Name}}
func (h *{{.Interface.Name}}Helper) ID() string {
	return "IDL:{{if .Interface.Module}}{{.Interface.Module}}/{{end}}{{.Interface.Name}}:1.0"
}

// Narrow converts a generic object reference to {{.Interface.Name}}
func (h *{{.Interface.Name}}Helper) Narrow(obj interface{}) ({{.Interface.Name}}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot narrow nil object to {{.Interface.Name}}")
	}
	
	if intf, ok := obj.({{.Interface.Name}}); ok {
		return intf, nil
	}
	
	return nil, fmt.Errorf("object does not implement {{.Interface.Name}}")
}

// {{.Interface.Name}}Stub implements {{.Interface.Name}} for client-side stubs
type {{.Interface.Name}}Stub struct {
	ObjectRef *corba.ObjectRef
}

{{range .Interface.Operations}}
// {{.Name}} implements the {{.Name}} operation
func (stub *{{$.Interface.Name}}Stub) {{.Name}}({{paramList .}}) ({{outParams .}}) {
	// Invoke remote method via CORBA
	result, err := stub.ObjectRef.Invoke("{{.Name}}", {{argList .}})
	if err != nil {
		{{if eq (goType .ReturnType) ""}}
		return err
		{{else}}
		var zero {{goType .ReturnType}}
		return zero, err
		{{end}}
	}
	
	{{if eq (goType .ReturnType) ""}}
	return nil
	{{else}}
	if typedResult, ok := result.({{goType .ReturnType}}); ok {
		return typedResult, nil
	}
	return nil, fmt.Errorf("unexpected result type from {{.Name}}")
	{{end}}
}
{{end}}

{{range .Interface.Attributes}}
// Get{{capitalize .Name}} gets the {{.Name}} attribute
func (stub *{{$.Interface.Name}}Stub) Get{{capitalize .Name}}() ({{goType .Type}}, error) {
	result, err := stub.ObjectRef.Invoke("_get_{{.Name}}")
	if err != nil {
		var zero {{goType .Type}}
		return zero, err
	}
	
	if typedResult, ok := result.({{goType .Type}}); ok {
		return typedResult, nil
	}
	return {{goType .Type}}{}, fmt.Errorf("unexpected result type from _get_{{.Name}}")
}

{{if not .Readonly}}
// Set{{capitalize .Name}} sets the {{.Name}} attribute
func (stub *{{$.Interface.Name}}Stub) Set{{capitalize .Name}}(value {{goType .Type}}) error {
	_, err := stub.ObjectRef.Invoke("_set_{{.Name}}", value)
	return err
}
{{end}}
{{end}}

// {{.Interface.Name}}Servant is the server-side implementation base for {{.Interface.Name}}
type {{.Interface.Name}}Servant struct {
	// Embed the implementation here
	Impl {{.Interface.Name}}
}

// Dispatch handles incoming method calls to the servant
func (servant *{{.Interface.Name}}Servant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	{{range .Interface.Operations}}
	case "{{.Name}}":
		// TODO: Convert args to appropriate types
		{{if eq (goType .ReturnType) ""}}
		err := servant.Impl.{{.Name}}(/* args */)
		return nil, err
		{{else}}
		result, err := servant.Impl.{{.Name}}(/* args */)
		return result, err
		{{end}}
	{{end}}
	{{range .Interface.Attributes}}
	case "_get_{{.Name}}":
		return servant.Impl.Get{{capitalize .Name}}()
	{{if not .Readonly}}
	case "_set_{{.Name}}":
		if len(args) != 1 {
			return nil, fmt.Errorf("wrong number of arguments for _set_{{.Name}}")
		}
		value, ok := args[0].({{goType .Type}})
		if !ok {
			return nil, fmt.Errorf("wrong argument type for _set_{{.Name}}")
		}
		return nil, servant.Impl.Set{{capitalize .Name}}(value)
	{{end}}
	{{end}}
	default:
		return nil, fmt.Errorf("method %s not found", methodName)
	}
}
`

// Template for Go struct from IDL struct
const structTemplate = `{{template "file" .}}

// {{.Struct.Name}} represents an IDL struct
type {{.Struct.Name}} struct {
	{{range .Struct.Fields}}
	{{capitalize .Name}} {{goType .Type}}
	{{end}}
}

// {{.Struct.Name}}Helper provides utility functions for {{.Struct.Name}}
type {{.Struct.Name}}Helper struct{}

// ID returns the repository ID for {{.Struct.Name}}
func (h *{{.Struct.Name}}Helper) ID() string {
	return "IDL:{{if .Struct.Module}}{{.Struct.Module}}/{{end}}{{.Struct.Name}}:1.0"
}

// New{{.Struct.Name}} creates a new instance of {{.Struct.Name}}
func New{{.Struct.Name}}() *{{.Struct.Name}} {
	return &{{.Struct.Name}}{}
}
`

// Template for Go enum from IDL enum
const enumTemplate = `{{template "file" .}}

// {{.Enum.Name}} represents an IDL enum
type {{.Enum.Name}} int

const (
	{{range $i, $e := .Enum.Elements}}
	{{if eq $i 0}}
	{{$.Enum.Name}}_{{$e}} {{$.Enum.Name}} = iota
	{{else}}
	{{$.Enum.Name}}_{{$e}}
	{{end}}
	{{end}}
)

// String converts the enum to a string
func (e {{.Enum.Name}}) String() string {
	names := []string{
		{{range .Enum.Elements}}
		"{{.}}",
		{{end}}
	}
	if e < 0 || int(e) >= len(names) {
		return fmt.Sprintf("{{.Enum.Name}}(%d)", e)
	}
	return names[e]
}

// {{.Enum.Name}}Helper provides utility functions for {{.Enum.Name}}
type {{.Enum.Name}}Helper struct{}

// ID returns the repository ID for {{.Enum.Name}}
func (h *{{.Enum.Name}}Helper) ID() string {
	return "IDL:{{if .Enum.Module}}{{.Enum.Module}}/{{end}}{{.Enum.Name}}:1.0"
}
`

// Template for Go typedef from IDL typedef
const typedefTemplate = `{{template "file" .}}

// {{.TypeDef.Name}} is a type alias for {{goType .TypeDef.OrigType}}
type {{.TypeDef.Name}} = {{goType .TypeDef.OrigType}}

// {{.TypeDef.Name}}Helper provides utility functions for {{.TypeDef.Name}}
type {{.TypeDef.Name}}Helper struct{}

// ID returns the repository ID for {{.TypeDef.Name}}
func (h *{{.TypeDef.Name}}Helper) ID() string {
	return "IDL:{{if .TypeDef.Module}}{{.TypeDef.Module}}/{{end}}{{.TypeDef.Name}}:1.0"
}
`

// Template for Go union from IDL union
const unionTemplate = `{{template "file" .}}

// {{.Union.Name}} represents an IDL union
type {{.Union.Name}} struct {
	Discriminant {{goType .Union.Discriminant}}
	Value interface{}
}

// Constants for {{.Union.Name}} discriminant values
const (
	{{range $i, $case := .Union.Cases}}
	{{range $j, $label := $case.Labels}}
	{{if ne $label "default"}}
	{{$.Union.Name}}_{{$case.Name}}_Case {{goType $.Union.Discriminant}} = {{$label}}
	{{end}}
	{{end}}
	{{end}}
)

{{range .Union.Cases}}
// Set{{capitalize .Name}} sets the union to the {{.Name}} case
func (u *{{$.Union.Name}}) Set{{capitalize .Name}}(value {{goType .Type}}) {
	u.Value = value
	{{if eq (index .Labels 0) "default"}}
	// Using default case
	{{else}}
	u.Discriminant = {{$.Union.Name}}_{{.Name}}_Case
	{{end}}
}

// Get{{capitalize .Name}} gets the {{.Name}} value if it's the active case
func (u *{{$.Union.Name}}) Get{{capitalize .Name}}() ({{goType .Type}}, bool) {
	{{range .Labels}}
	{{if eq . "default"}}
	// Check if the default case is active
	isDefault := true
	{{range $i, $case := $.Union.Cases}}
		{{range $j, $label := $case.Labels}}
			{{if and (ne $label "default") (ne $case.Name $.Name)}}
	if u.Discriminant == {{$.Union.Name}}_{{$case.Name}}_Case {
		isDefault = false
	}
			{{end}}
		{{end}}
	{{end}}
	if isDefault {
		if value, ok := u.Value.({{goType .Type}}); ok {
			return value, true
		}
	}
	{{else}}
	if u.Discriminant == {{$.Union.Name}}_{{$.Name}}_Case {
		if value, ok := u.Value.({{goType $.Type}}); ok {
			return value, true
		}
	}
	{{end}}
	{{end}}
	var zero {{goType .Type}}
	return zero, false
}
{{end}}

// {{.Union.Name}}Helper provides utility functions for {{.Union.Name}}
type {{.Union.Name}}Helper struct{}

// ID returns the repository ID for {{.Union.Name}}
func (h *{{.Union.Name}}Helper) ID() string {
	return "IDL:{{if .Union.Module}}{{.Union.Module}}/{{end}}{{.Union.Name}}:1.0"
}

// New{{.Union.Name}} creates a new instance of {{.Union.Name}}
func New{{.Union.Name}}() *{{.Union.Name}} {
	return &{{.Union.Name}}{}
}
`
