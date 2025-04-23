package idl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Parser represents an IDL parser that reads and parses IDL files
type Parser struct {
	lexer          *lexer
	currentToken   *token
	rootModule     *Module
	currentModule  *Module
	includeHandler func(string) (io.Reader, error)
}

// NewParser creates a new IDL parser
func NewParser() *Parser {
	rootModule := NewModule("")
	return &Parser{
		rootModule:    rootModule,
		currentModule: rootModule,
		includeHandler: func(path string) (io.Reader, error) {
			return nil, fmt.Errorf("include not supported: %s", path)
		},
	}
}

// SetIncludeHandler sets a handler for #include directives
func (p *Parser) SetIncludeHandler(handler func(string) (io.Reader, error)) {
	p.includeHandler = handler
}

// Parse parses an IDL file
func (p *Parser) Parse(reader io.Reader) error {
	p.lexer = newLexer(reader)

	// Get the first token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse the IDL file
	return p.parseIDLFile()
}

// GetRootModule returns the root module containing all parsed types
func (p *Parser) GetRootModule() *Module {
	return p.rootModule
}

// readChar reads the next character
func (l *lexer) readChar() {
	var err error
	l.current, _, err = l.reader.ReadRune()
	if err != nil {
		if err == io.EOF {
			l.eof = true
		}
		l.current = 0
	}
}

// nextToken advances to the next token
func (p *Parser) nextToken() error {
	var err error
	p.currentToken, err = p.lexer.nextToken()
	return err
}

// parseIDLFile parses an entire IDL file
func (p *Parser) parseIDLFile() error {
	for {
		// Check for end of file
		if p.currentToken.typ == tokenEOF {
			break
		}

		// Handle preprocessor directives
		if p.currentToken.typ == tokenPreprocessor {
			if err := p.parsePreprocessor(); err != nil {
				return err
			}
			continue
		}

		// Parse module, interface, struct, etc.
		switch p.currentToken.value {
		case "module":
			if err := p.parseModule(); err != nil {
				return err
			}
		case "interface":
			if err := p.parseInterface(); err != nil {
				return err
			}
		case "struct":
			if err := p.parseStruct(); err != nil {
				return err
			}
		case "enum":
			if err := p.parseEnum(); err != nil {
				return err
			}
		case "typedef":
			if err := p.parseTypedef(); err != nil {
				return err
			}
		case "union":
			if err := p.parseUnion(); err != nil {
				return err
			}
		case "const":
			if err := p.parseConst(); err != nil {
				return err
			}
		case "exception":
			if err := p.parseException(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected token: %s", p.currentToken.value)
		}
	}

	return nil
}

// parsePreprocessor handles preprocessor directives
func (p *Parser) parsePreprocessor() error {
	directive := p.currentToken.value

	// Process the include directive
	if strings.HasPrefix(directive, "#include") {
		// Extract the include path
		re := regexp.MustCompile(`#include\s+[<"]([^>"]+)[>"]`)
		match := re.FindStringSubmatch(directive)
		if len(match) < 2 {
			return fmt.Errorf("invalid include directive: %s", directive)
		}

		includePath := match[1]
		reader, err := p.includeHandler(includePath)
		if err != nil {
			return fmt.Errorf("failed to handle include %s: %w", includePath, err)
		}

		// Create a new parser for the included file
		includeParser := NewParser()
		includeParser.currentModule = p.currentModule

		// Parse the included file
		if err := includeParser.Parse(reader); err != nil {
			return fmt.Errorf("failed to parse included file %s: %w", includePath, err)
		}

		// Merge the included file's types into the current module
		for name, typ := range includeParser.currentModule.Types {
			p.currentModule.Types[name] = typ
		}
	}

	return p.nextToken()
}

// parseModule parses an IDL module
func (p *Parser) parseModule() error {
	// Skip "module" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get module name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected module name, got %s", p.currentToken.value)
	}

	moduleName := p.currentToken.value

	// Create or get the module
	var module *Module
	existingModule, exists := p.currentModule.GetSubmodule(moduleName)
	if exists {
		module = existingModule
	} else {
		module = p.currentModule.AddSubmodule(moduleName)
	}

	// Save the current module to restore it later
	parentModule := p.currentModule
	p.currentModule = module

	// Skip the module name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after module name, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse module contents
	for p.currentToken.typ != tokenCloseBrace {
		// Parse module contents (interfaces, structs, etc.)
		switch p.currentToken.value {
		case "module":
			if err := p.parseModule(); err != nil {
				return err
			}
		case "interface":
			if err := p.parseInterface(); err != nil {
				return err
			}
		case "struct":
			if err := p.parseStruct(); err != nil {
				return err
			}
		case "enum":
			if err := p.parseEnum(); err != nil {
				return err
			}
		case "typedef":
			if err := p.parseTypedef(); err != nil {
				return err
			}
		case "union":
			if err := p.parseUnion(); err != nil {
				return err
			}
		case "const":
			if err := p.parseConst(); err != nil {
				return err
			}
		case "exception":
			if err := p.parseException(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected token in module: %s", p.currentToken.value)
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after module definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Restore the parent module
	p.currentModule = parentModule

	return nil
}

// parseInterface parses an IDL interface
func (p *Parser) parseInterface() error {
	// Skip "interface" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get interface name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected interface name, got %s", p.currentToken.value)
	}

	interfaceName := p.currentToken.value

	// Skip the interface name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Interface type
	interfaceType := &InterfaceType{
		Name:       interfaceName,
		Module:     p.currentModule.Name,
		Parents:    []string{},
		Operations: []Operation{},
		Attributes: []Attribute{},
	}

	// Check for inheritance
	if p.currentToken.value == ":" {
		// Skip the colon
		if err := p.nextToken(); err != nil {
			return err
		}

		// Parse parent interfaces
		for {
			if p.currentToken.typ != tokenIdentifier {
				return fmt.Errorf("expected parent interface name, got %s", p.currentToken.value)
			}

			interfaceType.Parents = append(interfaceType.Parents, p.currentToken.value)

			// Skip the parent interface name
			if err := p.nextToken(); err != nil {
				return err
			}

			// Check for more parents
			if p.currentToken.value != "," {
				break
			}

			// Skip the comma
			if err := p.nextToken(); err != nil {
				return err
			}
		}
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after interface name, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse interface contents
	for p.currentToken.typ != tokenCloseBrace {
		// Check for attribute
		if p.currentToken.value == "readonly" || p.currentToken.value == "attribute" {
			readonly := false
			if p.currentToken.value == "readonly" {
				readonly = true
				// Skip "readonly"
				if err := p.nextToken(); err != nil {
					return err
				}

				// Expect "attribute"
				if p.currentToken.value != "attribute" {
					return fmt.Errorf("expected 'attribute' after 'readonly', got %s", p.currentToken.value)
				}
			}

			// Skip "attribute"
			if err := p.nextToken(); err != nil {
				return err
			}

			// Parse attribute type
			attrType, err := p.parseType()
			if err != nil {
				return err
			}

			// Parse attribute name
			if p.currentToken.typ != tokenIdentifier {
				return fmt.Errorf("expected attribute name, got %s", p.currentToken.value)
			}

			attrName := p.currentToken.value

			// Skip attribute name
			if err := p.nextToken(); err != nil {
				return err
			}

			// Create attribute
			attribute := Attribute{
				Name:     attrName,
				Type:     attrType,
				Readonly: readonly,
			}

			// Add attribute to interface
			interfaceType.Attributes = append(interfaceType.Attributes, attribute)

			// Expect semicolon
			if p.currentToken.typ != tokenSemicolon {
				return fmt.Errorf("expected ';' after attribute, got %s", p.currentToken.value)
			}

			// Skip semicolon
			if err := p.nextToken(); err != nil {
				return err
			}

			continue
		}

		// Check for oneway
		oneway := false
		if p.currentToken.value == "oneway" {
			oneway = true
			// Skip "oneway"
			if err := p.nextToken(); err != nil {
				return err
			}
		}

		// Parse return type
		returnType, err := p.parseType()
		if err != nil {
			return err
		}

		// Parse operation name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("expected operation name, got %s", p.currentToken.value)
		}

		operationName := p.currentToken.value

		// Skip operation name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Expect opening parenthesis
		if p.currentToken.typ != tokenOpenParen {
			return fmt.Errorf("expected '(' after operation name, got %s", p.currentToken.value)
		}

		// Skip opening parenthesis
		if err := p.nextToken(); err != nil {
			return err
		}

		// Parse parameters
		var parameters []Parameter
		if p.currentToken.typ != tokenCloseParen {
			for {
				// Parse parameter direction
				var direction Direction = In
				if p.currentToken.value == "in" || p.currentToken.value == "out" || p.currentToken.value == "inout" {
					direction = Direction(p.currentToken.value)
					// Skip direction
					if err := p.nextToken(); err != nil {
						return err
					}
				}

				// Parse parameter type
				paramType, err := p.parseType()
				if err != nil {
					return err
				}

				// Parse parameter name
				if p.currentToken.typ != tokenIdentifier {
					return fmt.Errorf("expected parameter name, got %s", p.currentToken.value)
				}

				paramName := p.currentToken.value

				// Skip parameter name
				if err := p.nextToken(); err != nil {
					return err
				}

				// Create parameter
				parameter := Parameter{
					Name:      paramName,
					Type:      paramType,
					Direction: direction,
				}

				// Add parameter to list
				parameters = append(parameters, parameter)

				// Check for more parameters
				if p.currentToken.typ != tokenComma {
					break
				}

				// Skip comma
				if err := p.nextToken(); err != nil {
					return err
				}
			}
		}

		// Expect closing parenthesis
		if p.currentToken.typ != tokenCloseParen {
			return fmt.Errorf("expected ')' after parameters, got %s", p.currentToken.value)
		}

		// Skip closing parenthesis
		if err := p.nextToken(); err != nil {
			return err
		}

		// Check for raises clause
		var raises []string
		if p.currentToken.value == "raises" {
			// Skip "raises"
			if err := p.nextToken(); err != nil {
				return err
			}

			// Expect opening parenthesis
			if p.currentToken.typ != tokenOpenParen {
				return fmt.Errorf("expected '(' after raises, got %s", p.currentToken.value)
			}

			// Skip opening parenthesis
			if err := p.nextToken(); err != nil {
				return err
			}

			// Parse exception list
			for {
				if p.currentToken.typ != tokenIdentifier {
					return fmt.Errorf("expected exception name, got %s", p.currentToken.value)
				}

				raises = append(raises, p.currentToken.value)

				// Skip exception name
				if err := p.nextToken(); err != nil {
					return err
				}

				// Check for more exceptions
				if p.currentToken.typ != tokenComma {
					break
				}

				// Skip comma
				if err := p.nextToken(); err != nil {
					return err
				}
			}

			// Expect closing parenthesis
			if p.currentToken.typ != tokenCloseParen {
				return fmt.Errorf("expected ')' after exception list, got %s", p.currentToken.value)
			}

			// Skip closing parenthesis
			if err := p.nextToken(); err != nil {
				return err
			}
		}

		// Create operation
		operation := Operation{
			Name:       operationName,
			ReturnType: returnType,
			Parameters: parameters,
			Raises:     raises,
			Oneway:     oneway,
		}

		// Add operation to interface
		interfaceType.Operations = append(interfaceType.Operations, operation)

		// Expect semicolon
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("expected ';' after operation, got %s", p.currentToken.value)
		}

		// Skip semicolon
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after interface definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add interface to current module
	p.currentModule.AddType(interfaceName, interfaceType)

	return nil
}

// parseType parses an IDL type
func (p *Parser) parseType() (Type, error) {
	if p.currentToken.typ != tokenIdentifier {
		return nil, fmt.Errorf("expected type name, got %s", p.currentToken.value)
	}

	typeName := p.currentToken.value

	// Skip the type name
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Check for sequence
	if typeName == "sequence" {
		// Expect '<'
		if p.currentToken.value != "<" {
			return nil, fmt.Errorf("expected '<' after sequence, got %s", p.currentToken.value)
		}

		// Skip '<'
		if err := p.nextToken(); err != nil {
			return nil, err
		}

		// Parse element type
		elementType, err := p.parseType()
		if err != nil {
			return nil, err
		}

		// Check for sequence size
		maxSize := -1 // unbounded by default
		if p.currentToken.value == "," {
			// Skip ','
			if err := p.nextToken(); err != nil {
				return nil, err
			}

			// Parse size
			if p.currentToken.typ != tokenNumber {
				return nil, fmt.Errorf("expected sequence size, got %s", p.currentToken.value)
			}

			size, err := strconv.Atoi(p.currentToken.value)
			if err != nil {
				return nil, fmt.Errorf("invalid sequence size: %s", p.currentToken.value)
			}

			maxSize = size

			// Skip size
			if err := p.nextToken(); err != nil {
				return nil, err
			}
		}

		// Expect '>'
		if p.currentToken.value != ">" {
			return nil, fmt.Errorf("expected '>' after sequence type, got %s", p.currentToken.value)
		}

		// Skip '>'
		if err := p.nextToken(); err != nil {
			return nil, err
		}

		return &SequenceType{
			ElementType: elementType,
			MaxSize:     maxSize,
		}, nil
	}

	// Handle primitive types
	for _, bt := range []BasicType{
		TypeShort, TypeLong, TypeLongLong, TypeUShort, TypeULong, TypeULongLong,
		TypeFloat, TypeDouble, TypeBoolean, TypeChar, TypeWChar, TypeOctet,
		TypeAny, TypeString, TypeWString, TypeVoid,
	} {
		if string(bt) == typeName {
			return &SimpleType{Name: bt}, nil
		}
	}

	// Handle "unsigned long" and "unsigned short"
	if typeName == "unsigned" {
		// Get the next part of the type
		if p.currentToken.value == "short" {
			// Skip "short"
			if err := p.nextToken(); err != nil {
				return nil, err
			}
			return &SimpleType{Name: TypeUShort}, nil
		} else if p.currentToken.value == "long" {
			// Skip "long"
			if err := p.nextToken(); err != nil {
				return nil, err
			}

			// Check for "long long"
			if p.currentToken.value == "long" {
				// Skip second "long"
				if err := p.nextToken(); err != nil {
					return nil, err
				}
				return &SimpleType{Name: TypeULongLong}, nil
			}

			return &SimpleType{Name: TypeULong}, nil
		}

		return nil, fmt.Errorf("expected 'short' or 'long' after 'unsigned', got %s", p.currentToken.value)
	}

	// Handle "long long"
	if typeName == "long" && p.currentToken.value == "long" {
		// Skip second "long"
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return &SimpleType{Name: TypeLongLong}, nil
	}

	// For other types, look them up in the current module or parent modules
	// This would require a more complex implementation to handle scoping correctly

	// For now, just return as a simple type
	return &SimpleType{Name: BasicType(typeName)}, nil
}

// parseStruct parses an IDL struct
func (p *Parser) parseStruct() error {
	// Skip "struct" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get struct name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected struct name, got %s", p.currentToken.value)
	}

	structName := p.currentToken.value

	// Skip the struct name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Create struct type
	structType := &StructType{
		Name:   structName,
		Module: p.currentModule.Name,
		Fields: []StructField{},
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after struct name, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse struct fields
	for p.currentToken.typ != tokenCloseBrace {
		// Parse field type
		fieldType, err := p.parseType()
		if err != nil {
			return err
		}

		// Parse field name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("expected field name, got %s", p.currentToken.value)
		}

		fieldName := p.currentToken.value

		// Skip field name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Add field to struct
		structType.Fields = append(structType.Fields, StructField{
			Name: fieldName,
			Type: fieldType,
		})

		// Expect semicolon
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("expected ';' after field definition, got %s", p.currentToken.value)
		}

		// Skip semicolon
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after struct definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add struct to current module
	p.currentModule.AddType(structName, structType)

	return nil
}

// parseEnum parses an IDL enum
func (p *Parser) parseEnum() error {
	// Skip "enum" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get enum name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected enum name, got %s", p.currentToken.value)
	}

	enumName := p.currentToken.value

	// Skip the enum name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Create enum type
	enumType := &EnumType{
		Name:     enumName,
		Module:   p.currentModule.Name,
		Elements: []string{},
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after enum name, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse enum elements
	for {
		// Parse element name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("expected enum element name, got %s", p.currentToken.value)
		}

		elementName := p.currentToken.value

		// Add element to enum
		enumType.Elements = append(enumType.Elements, elementName)

		// Skip element name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Check for comma
		if p.currentToken.typ != tokenComma {
			break
		}

		// Skip comma
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Expect closing brace
	if p.currentToken.typ != tokenCloseBrace {
		return fmt.Errorf("expected '}' after enum elements, got %s", p.currentToken.value)
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after enum definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add enum to current module
	p.currentModule.AddType(enumName, enumType)

	return nil
}

// parseTypedef parses an IDL typedef
func (p *Parser) parseTypedef() error {
	// Skip "typedef" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse original type
	origType, err := p.parseType()
	if err != nil {
		return err
	}

	// Get new type name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected typedef name, got %s", p.currentToken.value)
	}

	typeName := p.currentToken.value

	// Skip the type name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Create typedef
	typeDef := &TypeDef{
		Name:     typeName,
		Module:   p.currentModule.Name,
		OrigType: origType,
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after typedef, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add typedef to current module
	p.currentModule.AddType(typeName, typeDef)

	return nil
}

// parseUnion parses an IDL union
func (p *Parser) parseUnion() error {
	// Skip "union" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get union name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected union name, got %s", p.currentToken.value)
	}

	unionName := p.currentToken.value

	// Skip the union name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect "switch"
	if p.currentToken.value != "switch" {
		return fmt.Errorf("expected 'switch' after union name, got %s", p.currentToken.value)
	}

	// Skip "switch"
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect opening parenthesis
	if p.currentToken.typ != tokenOpenParen {
		return fmt.Errorf("expected '(' after switch, got %s", p.currentToken.value)
	}

	// Skip opening parenthesis
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse discriminant type
	discriminantType, err := p.parseType()
	if err != nil {
		return err
	}

	// Expect closing parenthesis
	if p.currentToken.typ != tokenCloseParen {
		return fmt.Errorf("expected ')' after discriminant type, got %s", p.currentToken.value)
	}

	// Skip closing parenthesis
	if err := p.nextToken(); err != nil {
		return err
	}

	// Create union type
	unionType := &UnionType{
		Name:         unionName,
		Module:       p.currentModule.Name,
		Discriminant: discriminantType,
		Cases:        []UnionCase{},
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after union header, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse union cases
	for p.currentToken.typ != tokenCloseBrace {
		var labels []string

		// Parse case label(s)
		for p.currentToken.value == "case" || p.currentToken.value == "default" {
			if p.currentToken.value == "case" {
				// Skip "case"
				if err := p.nextToken(); err != nil {
					return err
				}

				// Parse case value (this is simplified; should handle expressions)
				label := p.currentToken.value

				// Skip case value
				if err := p.nextToken(); err != nil {
					return err
				}

				labels = append(labels, label)
			} else { // default
				// Skip "default"
				if err := p.nextToken(); err != nil {
					return err
				}

				labels = append(labels, "default")
			}

			// Expect colon
			if p.currentToken.typ != tokenColon {
				return fmt.Errorf("expected ':' after case label, got %s", p.currentToken.value)
			}

			// Skip colon
			if err := p.nextToken(); err != nil {
				return err
			}
		}

		// Parse case type
		caseType, err := p.parseType()
		if err != nil {
			return err
		}

		// Parse case name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("expected case name, got %s", p.currentToken.value)
		}

		caseName := p.currentToken.value

		// Skip case name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Add case to union
		unionType.Cases = append(unionType.Cases, UnionCase{
			Labels: labels,
			Name:   caseName,
			Type:   caseType,
		})

		// Expect semicolon
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("expected ';' after case, got %s", p.currentToken.value)
		}

		// Skip semicolon
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after union definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add union to current module
	p.currentModule.AddType(unionName, unionType)

	return nil
}

// parseConst is a placeholder - would parse an IDL const
func (p *Parser) parseConst() error {
	// Skip "const" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// TODO: Implement const parsing

	// For now, just skip until semicolon
	for p.currentToken.typ != tokenSemicolon {
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Skip the semicolon
	return p.nextToken()
}

// parseException is a placeholder - would parse an IDL exception
func (p *Parser) parseException() error {
	// Skip "exception" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get exception name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("expected exception name, got %s", p.currentToken.value)
	}

	exceptionName := p.currentToken.value

	// Skip the exception name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Create struct type for the exception
	exceptionType := &StructType{
		Name:   exceptionName,
		Module: p.currentModule.Name,
		Fields: []StructField{},
	}

	// Expect opening brace
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("expected '{' after exception name, got %s", p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse exception fields
	for p.currentToken.typ != tokenCloseBrace {
		// Parse field type
		fieldType, err := p.parseType()
		if err != nil {
			return err
		}

		// Parse field name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("expected field name, got %s", p.currentToken.value)
		}

		fieldName := p.currentToken.value

		// Skip field name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Add field to exception
		exceptionType.Fields = append(exceptionType.Fields, StructField{
			Name: fieldName,
			Type: fieldType,
		})

		// Expect semicolon
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("expected ';' after field definition, got %s", p.currentToken.value)
		}

		// Skip semicolon
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("expected ';' after exception definition, got %s", p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add exception to current module
	p.currentModule.AddType(exceptionName, exceptionType)

	return nil
}

// Token types for lexical analysis
type tokenType int

const (
	tokenIdentifier tokenType = iota
	tokenNumber
	tokenString
	tokenChar
	tokenOperator
	tokenOpenBrace
	tokenCloseBrace
	tokenOpenParen
	tokenCloseParen
	tokenOpenBracket
	tokenCloseBracket
	tokenSemicolon
	tokenColon
	tokenComma
	tokenPreprocessor
	tokenEOF
)

// Token represents a lexical token
type token struct {
	typ   tokenType
	value string
}

// Lexer performs lexical analysis of IDL files
type lexer struct {
	reader  *bufio.Reader
	current rune
	eof     bool
}

// newLexer creates a new lexer
func newLexer(r io.Reader) *lexer {
	lex := &lexer{
		reader: bufio.NewReader(r),
	}
	lex.readChar()
	return lex
}

// skipWhitespace skips whitespace characters
func (l *lexer) skipWhitespace() {
	for !l.eof && (l.current == ' ' || l.current == '\t' || l.current == '\n' || l.current == '\r') {
		l.readChar()
	}
}

// skipComment skips a comment
func (l *lexer) skipComment() {
	if l.current == '/' && !l.eof {
		l.readChar()
		if l.current == '/' {
			// Single-line comment
			for !l.eof && l.current != '\n' {
				l.readChar()
			}
		} else if l.current == '*' {
			// Multi-line comment
			for !l.eof {
				l.readChar()
				if l.current == '*' {
					l.readChar()
					if l.current == '/' {
						l.readChar()
						break
					}
				}
			}
		} else {
			// Not a comment, put back the second '/'
			// This is simplified, real implementation would need proper peeking
			l.readChar()
		}
	}
}

// nextToken returns the next token
func (l *lexer) nextToken() (*token, error) {
	// Skip whitespace and comments
	for !l.eof {
		l.skipWhitespace()
		if l.current == '/' {
			// Potential comment
			l.skipComment()
		} else {
			break
		}
	}

	if l.eof {
		return &token{typ: tokenEOF, value: ""}, nil
	}

	// Process token
	switch {
	case l.current == '{':
		l.readChar()
		return &token{typ: tokenOpenBrace, value: "{"}, nil
	case l.current == '}':
		l.readChar()
		return &token{typ: tokenCloseBrace, value: "}"}, nil
	case l.current == '(':
		l.readChar()
		return &token{typ: tokenOpenParen, value: "("}, nil
	case l.current == ')':
		l.readChar()
		return &token{typ: tokenCloseParen, value: ")"}, nil
	case l.current == '[':
		l.readChar()
		return &token{typ: tokenOpenBracket, value: "["}, nil
	case l.current == ']':
		l.readChar()
		return &token{typ: tokenCloseBracket, value: "]"}, nil
	case l.current == ';':
		l.readChar()
		return &token{typ: tokenSemicolon, value: ";"}, nil
	case l.current == ':':
		l.readChar()
		return &token{typ: tokenColon, value: ":"}, nil
	case l.current == ',':
		l.readChar()
		return &token{typ: tokenComma, value: ","}, nil
	case l.current == '#':
		// Preprocessor directive
		return l.readPreprocessor()
	case isLetter(l.current) || l.current == '_':
		// Identifier
		return l.readIdentifier()
	case isDigit(l.current):
		// Number
		return l.readNumber()
	case l.current == '"':
		// String
		return l.readString()
	case l.current == '\'':
		// Character
		return l.readCharLiteral()
	case isOperator(l.current):
		// Operator
		return l.readOperator()
	default:
		return nil, fmt.Errorf("unexpected character: %c", l.current)
	}
}

// readPreprocessor reads a preprocessor directive
func (l *lexer) readPreprocessor() (*token, error) {
	var directive strings.Builder
	directive.WriteRune(l.current)
	l.readChar()

	// Read the rest of the directive
	for !l.eof && l.current != '\n' {
		directive.WriteRune(l.current)
		l.readChar()
	}

	return &token{typ: tokenPreprocessor, value: directive.String()}, nil
}

// readIdentifier reads an identifier
func (l *lexer) readIdentifier() (*token, error) {
	var ident strings.Builder
	ident.WriteRune(l.current)
	l.readChar()

	for !l.eof && (isLetter(l.current) || isDigit(l.current) || l.current == '_') {
		ident.WriteRune(l.current)
		l.readChar()
	}

	return &token{typ: tokenIdentifier, value: ident.String()}, nil
}

// readNumber reads a number
func (l *lexer) readNumber() (*token, error) {
	var num strings.Builder
	num.WriteRune(l.current)
	l.readChar()

	for !l.eof && isDigit(l.current) {
		num.WriteRune(l.current)
		l.readChar()
	}

	// Handle decimal point
	if !l.eof && l.current == '.' {
		num.WriteRune(l.current)
		l.readChar()

		for !l.eof && isDigit(l.current) {
			num.WriteRune(l.current)
			l.readChar()
		}
	}

	return &token{typ: tokenNumber, value: num.String()}, nil
}

// readString reads a string literal
func (l *lexer) readString() (*token, error) {
	var str strings.Builder
	// Skip the opening quote
	l.readChar()

	for !l.eof && l.current != '"' {
		// Handle escape sequences
		if l.current == '\\' {
			l.readChar()
			if l.eof {
				return nil, errors.New("unterminated string literal")
			}
		}
		str.WriteRune(l.current)
		l.readChar()
	}

	// Skip the closing quote
	if l.eof {
		return nil, errors.New("unterminated string literal")
	}
	l.readChar()

	return &token{typ: tokenString, value: str.String()}, nil
}

// readCharLiteral reads a character literal
func (l *lexer) readCharLiteral() (*token, error) {
	var ch strings.Builder
	// Skip the opening quote
	l.readChar()

	if l.eof {
		return nil, errors.New("unterminated character literal")
	}

	// Handle escape sequence
	if l.current == '\\' {
		ch.WriteRune(l.current)
		l.readChar()
		if l.eof {
			return nil, errors.New("unterminated character literal")
		}
	}
	ch.WriteRune(l.current)
	l.readChar()

	// Skip the closing quote
	if l.current != '\'' {
		return nil, errors.New("unterminated character literal")
	}
	l.readChar()

	return &token{typ: tokenChar, value: ch.String()}, nil
}

// readOperator reads an operator
func (l *lexer) readOperator() (*token, error) {
	var op strings.Builder
	op.WriteRune(l.current)
	l.readChar()

	// Handle multi-character operators
	if !l.eof && isOperator(l.current) {
		op.WriteRune(l.current)
		l.readChar()
	}

	return &token{typ: tokenOperator, value: op.String()}, nil
}

// isLetter checks if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit checks if a rune is a digit
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isOperator checks if a rune is an operator
func isOperator(r rune) bool {
	return r == '+' || r == '-' || r == '*' || r == '/' || r == '=' || r == '<' || r == '>' || r == '!'
}
