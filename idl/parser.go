package idl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	includedFiles  map[string]bool // 跟踪已经包含的文件，防止循环引用
	currentFile    string          // 当前处理的文件名，用于错误报告
	includeDirs    []string        // include 搜索路径
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
		includedFiles: make(map[string]bool), // 初始化已包含文件的映射
		includeDirs:   []string{},            // 初始化包含目录
	}
}

// SetIncludeHandler sets a handler for #include directives
func (p *Parser) SetIncludeHandler(handler func(string) (io.Reader, error)) {
	p.includeHandler = handler
}

// AddIncludeDir adds a directory to search for included files
func (p *Parser) AddIncludeDir(dir string) {
	p.includeDirs = append(p.includeDirs, dir)
}

// SetIncludeDirs sets the directories to search for included files
func (p *Parser) SetIncludeDirs(dirs []string) {
	p.includeDirs = dirs
}

// SetCurrentFile sets the name of the file currently being processed
func (p *Parser) SetCurrentFile(filename string) {
	p.currentFile = filename
}

// Parse parses an IDL file
func (p *Parser) Parse(reader io.Reader) error {
	filename := p.currentFile
	if filename == "" {
		filename = "<input>"
	}
	p.lexer = newLexer(reader, filename)

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
		return
	}

	// 更新行号和列号
	if l.current == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// nextToken advances to the next token
func (p *Parser) nextToken() error {
	var err error
	p.currentToken, err = p.lexer.nextToken()
	if err != nil {
		return fmt.Errorf("%s:%d:%d: %v", p.lexer.filename, p.lexer.lastLine, p.lexer.lastCol, err)
	}
	return nil
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
			return fmt.Errorf("%s:%d:%d: unexpected token: %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}
	}

	return nil
}

// parsePreprocessor handles preprocessor directives
func (p *Parser) parsePreprocessor() error {
	directive := p.currentToken.value

	// Process the include directive
	if strings.HasPrefix(directive, "#include") {
		// 区分 #include <system.idl> 和 #include "user.idl" 格式
		var includeType string
		var includePath string

		// 匹配 <system.idl> 格式
		reSystem := regexp.MustCompile(`#include\s+<([^>]+)>`)
		matchSystem := reSystem.FindStringSubmatch(directive)
		if len(matchSystem) == 2 {
			includeType = "system"
			includePath = matchSystem[1]
		} else {
			// 匹配 "user.idl" 格式
			reUser := regexp.MustCompile(`#include\s+"([^"]+)"`)
			matchUser := reUser.FindStringSubmatch(directive)
			if len(matchUser) == 2 {
				includeType = "user"
				includePath = matchUser[1]
			} else {
				return fmt.Errorf("%s:%d:%d: invalid include directive: %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, directive)
			}
		}

		// 检查是否已经包含过该文件（防止循环引用）
		// 使用绝对路径来标准化文件引用
		if p.includedFiles[includePath] {
			// 文件已被包含过，跳过
			return p.nextToken()
		}

		// 尝试打开包含的文件
		reader, includeFilePath, err := p.resolveIncludePath(includePath, includeType)
		if err != nil {
			return fmt.Errorf("%s:%d:%d: failed to resolve include %s: %w",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, includePath, err)
		}

		// 标记文件已被包含
		p.includedFiles[includePath] = true

		// 保存当前文件名，以便处理完后恢复
		prevFile := p.currentFile

		// 创建一个新的解析器实例用于包含的文件
		includeParser := NewParser()
		includeParser.currentModule = p.currentModule
		includeParser.includedFiles = p.includedFiles // 共享已包含文件列表
		includeParser.includeDirs = p.includeDirs     // 共享包含目录
		includeParser.SetCurrentFile(includeFilePath)

		// 解析包含的文件
		if err := includeParser.Parse(reader); err != nil {
			return fmt.Errorf("%s:%d:%d: failed to parse included file %s: %w",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, includePath, err)
		}

		// 合并包含文件中定义的类型到当前模块
		for name, typ := range includeParser.currentModule.Types {
			p.currentModule.Types[name] = typ
		}

		// 恢复当前文件名
		p.SetCurrentFile(prevFile)
	} else if strings.HasPrefix(directive, "#pragma") {
		// 处理 #pragma 指令，如 #pragma ID, #pragma prefix 等
		// 根据 CORBA 规范，这些对 IDL 到 Go 代码生成也很重要
		if err := p.parsePragma(directive); err != nil {
			return err
		}
	}

	return p.nextToken()
}

// resolveIncludePath 解析 include 路径并返回对应的 reader
func (p *Parser) resolveIncludePath(path string, includeType string) (io.Reader, string, error) {
	// 首先尝试使用 includeHandler
	reader, err := p.includeHandler(path)
	if err == nil {
		return reader, path, nil
	}

	// 如果 includeHandler 失败，尝试在 includeDirs 中查找文件
	// 对于 system 头文件 (#include <file>)，优先在系统目录中查找
	// 对于 user 头文件 (#include "file")，优先在当前目录查找

	searchDirs := p.includeDirs

	// 如果是用户头文件，首先尝试相对于当前文件的路径
	if includeType == "user" && p.currentFile != "" {
		// 从当前文件的目录开始查找
		baseDir := filepath.Dir(p.currentFile)
		fullPath := filepath.Join(baseDir, path)
		if file, err := os.Open(fullPath); err == nil {
			return file, fullPath, nil
		}
	}

	// 在所有包含目录中查找
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}

		fullPath := filepath.Join(dir, path)
		file, err := os.Open(fullPath)
		if err == nil {
			return file, fullPath, nil
		}
	}

	return nil, "", fmt.Errorf("%s:%d:%d: file not found: %s",
		p.currentToken.filename, p.currentToken.line, p.currentToken.column, path)
}

// parsePragma 处理 IDL 中的 pragma 指令
func (p *Parser) parsePragma(directive string) error {
	// 处理 #pragma ID 指令
	reID := regexp.MustCompile(`#pragma\s+ID\s+([a-zA-Z_][a-zA-Z0-9_]*(?:::[a-zA-Z_][a-zA-Z0-9_]*)*)\s+"([^"]+)"`)
	matchID := reID.FindStringSubmatch(directive)
	if len(matchID) == 3 {
		// 处理 ID pragma，关联接口名称和 Repository ID
		identifierName := matchID[1]
		repositoryID := matchID[2]

		// 解析可能是作用域名称（包含::）的标识符
		parts := strings.Split(identifierName, "::")
		targetModule := p.currentModule
		typeName := identifierName

		// 如果是作用域名称，需要找到对应的模块和类型
		if len(parts) > 1 {
			// 最后一部分是类型名
			typeName = parts[len(parts)-1]

			// 前面的部分是模块路径
			modulePath := parts[:len(parts)-1]

			// 找到目标模块
			for _, moduleName := range modulePath {
				if submod, exists := targetModule.GetSubmodule(moduleName); exists {
					targetModule = submod
				} else {
					// 模块不存在，忽略这个 pragma
					return nil
				}
			}
		}

		// 在目标模块中查找类型
		if typ, ok := targetModule.Types[typeName]; ok {
			typ.SetRepositoryID(repositoryID)
		}

		return nil
	}

	// 处理 #pragma prefix 指令
	rePrefix := regexp.MustCompile(`#pragma\s+prefix\s+"([^"]+)"`)
	matchPrefix := rePrefix.FindStringSubmatch(directive)
	if len(matchPrefix) == 2 {
		// 处理 prefix pragma，设置当前模块的前缀
		prefix := matchPrefix[1]
		p.currentModule.Prefix = prefix
		return nil
	}

	// 处理 #pragma version 指令
	reVersion := regexp.MustCompile(`#pragma\s+version\s+([a-zA-Z_][a-zA-Z0-9_]*(?:::[a-zA-Z_][a-zA-Z0-9_]*)*)\s+([0-9]+)\.([0-9]+)`)
	matchVersion := reVersion.FindStringSubmatch(directive)
	if len(matchVersion) == 4 {
		// 版本信息，可用于构建 Repository ID
		// 在此实现中，我们将版本信息作为 Repository ID 的一部分存储
		return nil
	}

	// 其他 pragma 指令暂时忽略
	return nil
}

// parseModule parses an IDL module
func (p *Parser) parseModule() error {
	// Skip "module" token
	if err := p.nextToken(); err != nil {
		return err
	}

	// Get module name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("%s:%d:%d: expected module name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after module name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: unexpected token in module: %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("%s:%d:%d: expected ';' after module definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected interface name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		Types:      make(map[string]Type), // Initialize for nested enums
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
				return fmt.Errorf("%s:%d:%d: expected parent interface name, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after interface name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
					return fmt.Errorf("%s:%d:%d: expected 'attribute' after 'readonly', got %s",
						p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return fmt.Errorf("%s:%d:%d: expected attribute name, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return fmt.Errorf("%s:%d:%d: expected ';' after attribute, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}

			// Skip semicolon
			if err := p.nextToken(); err != nil {
				return err
			}

			continue
		}

		// Support enums inside interface (CORBA spec)
		if p.currentToken.value == "enum" {
			if err := p.nextToken(); err != nil {
				return err
			}
			// Get enum name
			if p.currentToken.typ != tokenIdentifier {
				return fmt.Errorf("%s:%d:%d: expected enum name, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}
			enumName := p.currentToken.value
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
				return fmt.Errorf("%s:%d:%d: expected '{' after enum name, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}
			if err := p.nextToken(); err != nil {
				return err
			}
			// Parse enum elements
			for p.currentToken.typ != tokenCloseBrace {
				if p.currentToken.typ != tokenIdentifier {
					return fmt.Errorf("%s:%d:%d: expected enum element name, got %s",
						p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
				}
				enumType.Elements = append(enumType.Elements, p.currentToken.value)
				if err := p.nextToken(); err != nil {
					return err
				}
				if p.currentToken.typ == tokenComma {
					if err := p.nextToken(); err != nil {
						return err
					}
				} else if p.currentToken.typ != tokenCloseBrace {
					return fmt.Errorf("%s:%d:%d: expected ',' or '}' after enum element, got %s",
						p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
				}
			}
			// Skip the closing brace
			if err := p.nextToken(); err != nil {
				return err
			}
			// Expect semicolon
			if p.currentToken.typ != tokenSemicolon {
				return fmt.Errorf("%s:%d:%d: expected ';' after enum definition, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}
			if err := p.nextToken(); err != nil {
				return err
			}
			// Add enum to the interface's Types map (or currentModule if that's your design)
			if interfaceType.Types == nil {
				interfaceType.Types = make(map[string]Type)
			}
			interfaceType.Types[enumName] = enumType
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
			return fmt.Errorf("%s:%d:%d: expected operation name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		operationName := p.currentToken.value

		// Skip operation name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Expect opening parenthesis
		if p.currentToken.typ != tokenOpenParen {
			return fmt.Errorf("%s:%d:%d: expected '(' after operation name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
					return fmt.Errorf("%s:%d:%d: expected parameter name, got %s",
						p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected ')' after parameters, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return fmt.Errorf("%s:%d:%d: expected '(' after raises, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}

			// Skip opening parenthesis
			if err := p.nextToken(); err != nil {
				return err
			}

			// Parse exception list
			for {
				if p.currentToken.typ != tokenIdentifier {
					return fmt.Errorf("%s:%d:%d: expected exception name, got %s",
						p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return fmt.Errorf("%s:%d:%d: expected ')' after exception list, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected ';' after operation, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ';' after interface definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return nil, fmt.Errorf("%s:%d:%d: expected type name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return nil, fmt.Errorf("%s:%d:%d: expected '<' after sequence, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return nil, fmt.Errorf("%s:%d:%d: expected sequence size, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}

			size, err := strconv.Atoi(p.currentToken.value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d:%d: invalid sequence size: %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
			}

			maxSize = size

			// Skip size
			if err := p.nextToken(); err != nil {
				return nil, err
			}
		}

		// Expect '>'
		if p.currentToken.value != ">" {
			return nil, fmt.Errorf("%s:%d:%d: expected '>' after sequence type, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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

		return nil, fmt.Errorf("%s:%d:%d: expected 'short' or 'long' after 'unsigned', got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	// Handle "long long"
	if typeName == "long" && p.currentToken.value == "long" {
		// Skip second "long"
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return &SimpleType{Name: TypeLongLong}, nil
	}

	// 处理作用域名称，例如 A::B::C
	if strings.Contains(typeName, "::") {
		return &ScopedType{Name: typeName}, nil
	}

	// 对于其他类型，以简单类型返回
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
		return fmt.Errorf("%s:%d:%d: expected struct name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after struct name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected field name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected ';' after field definition, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ';' after struct definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected enum name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after enum name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	// Skip the opening brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Parse enum elements
	// Continue parsing until we find a closing brace
	for p.currentToken.typ != tokenCloseBrace {
		// Parse element name
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("%s:%d:%d: expected enum element name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		elementName := p.currentToken.value

		// Add element to enum
		enumType.Elements = append(enumType.Elements, elementName)

		// Skip element name
		if err := p.nextToken(); err != nil {
			return err
		}

		// Check for comma or closing brace
		if p.currentToken.typ == tokenComma {
			// Skip comma
			if err := p.nextToken(); err != nil {
				return err
			}
		} else if p.currentToken.typ != tokenCloseBrace {
			// If not a comma or closing brace, it's an error
			return fmt.Errorf("%s:%d:%d: expected ',' or '}' after enum element, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}
	}

	// Skip the closing brace
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect semicolon
	if p.currentToken.typ != tokenSemicolon {
		return fmt.Errorf("%s:%d:%d: expected ';' after enum definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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

	// 检查是否是内联结构体定义
	if p.currentToken.value == "struct" {
		// 处理内联结构体定义，可能是匿名或命名结构体
		return p.parseInlineStructTypedef()
	}

	// 处理常规类型定义
	origType, err := p.parseType()
	if err != nil {
		return err
	}

	// Get new type name
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("%s:%d:%d: expected typedef name, got %s", p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ';' after typedef, got %s", p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	// Skip the semicolon
	if err := p.nextToken(); err != nil {
		return err
	}

	// Add typedef to current module
	p.currentModule.AddType(typeName, typeDef)

	return nil
}

// parseInlineStructTypedef 解析内联结构体定义，如 typedef struct {...} Name; 或 typedef struct StructName {...} Name;
func (p *Parser) parseInlineStructTypedef() error {
	// 已经跳过了 "typedef" 和 "struct" 标记
	if err := p.nextToken(); err != nil {
		return err
	}

	// 检查是否有结构体名称（命名结构体）
	structName := ""
	if p.currentToken.typ == tokenIdentifier {
		// 这是一个命名结构体
		structName = p.currentToken.value

		// 跳过结构体名称
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// 创建结构体类型
	structType := &StructType{
		Name:   structName, // 可能为空，表示匿名结构体
		Module: p.currentModule.Name,
		Fields: []StructField{},
	}

	// 期望左大括号
	if p.currentToken.typ != tokenOpenBrace {
		return fmt.Errorf("%s:%d:%d: expected '{' after 'struct' or struct name, got %s",
			p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	// 跳过左大括号
	if err := p.nextToken(); err != nil {
		return err
	}

	// 解析结构体字段
	for p.currentToken.typ != tokenCloseBrace {
		// 解析字段类型
		fieldType, err := p.parseType()
		if err != nil {
			return err
		}

		// 解析字段名
		if p.currentToken.typ != tokenIdentifier {
			return fmt.Errorf("%s:%d:%d: expected field name, got %s",
				p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		fieldName := p.currentToken.value

		// 跳过字段名
		if err := p.nextToken(); err != nil {
			return err
		}

		// 添加字段到结构体
		structType.Fields = append(structType.Fields, StructField{
			Name: fieldName,
			Type: fieldType,
		})

		// 期望分号
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("%s:%d:%d: expected ';' after field definition, got %s",
				p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		// 跳过分号
		if err := p.nextToken(); err != nil {
			return err
		}
	}

	// 跳过右大括号
	if err := p.nextToken(); err != nil {
		return err
	}

	// 获取typedef别名
	if p.currentToken.typ != tokenIdentifier {
		return fmt.Errorf("%s:%d:%d: expected typedef name, got %s",
			p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	typedefName := p.currentToken.value

	// 如果结构体有名称，先将结构体添加到模块
	if structName != "" {
		// 如果在typedef之前结构体已经有名字，我们需要先添加结构体定义
		p.currentModule.AddType(structName, structType)

		// 然后创建一个typedef，指向已命名的结构体
		typeDef := &TypeDef{
			Name:     typedefName,
			Module:   p.currentModule.Name,
			OrigType: &ScopedType{Name: structName},
		}

		// 跳过typedef名称
		if err := p.nextToken(); err != nil {
			return err
		}

		// 期望分号
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("%s:%d:%d: expected ';' after typedef struct definition, got %s",
				p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		// 跳过分号
		if err := p.nextToken(); err != nil {
			return err
		}

		// 添加typedef到当前模块
		p.currentModule.AddType(typedefName, typeDef)
	} else {
		// 对于匿名结构体，设置结构体名称为typedef名称
		structType.Name = typedefName

		// 跳过typedef名称
		if err := p.nextToken(); err != nil {
			return err
		}

		// 期望分号
		if p.currentToken.typ != tokenSemicolon {
			return fmt.Errorf("%s:%d:%d: expected ';' after typedef struct definition, got %s",
				p.lexer.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
		}

		// 跳过分号
		if err := p.nextToken(); err != nil {
			return err
		}

		// 添加结构体类型到当前模块
		p.currentModule.AddType(typedefName, structType)
	}

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
		return fmt.Errorf("%s:%d:%d: expected union name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	unionName := p.currentToken.value

	// Skip the union name
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect "switch"
	if p.currentToken.value != "switch" {
		return fmt.Errorf("%s:%d:%d: expected 'switch' after union name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
	}

	// Skip "switch"
	if err := p.nextToken(); err != nil {
		return err
	}

	// Expect opening parenthesis
	if p.currentToken.typ != tokenOpenParen {
		return fmt.Errorf("%s:%d:%d: expected '(' after switch, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ')' after discriminant type, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after union header, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
				return fmt.Errorf("%s:%d:%d: expected ':' after case label, got %s",
					p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected case name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected ';' after case, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ';' after union definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected exception name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected '{' after exception name, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected field name, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
			return fmt.Errorf("%s:%d:%d: expected ';' after field definition, got %s",
				p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
		return fmt.Errorf("%s:%d:%d: expected ';' after exception definition, got %s",
			p.currentToken.filename, p.currentToken.line, p.currentToken.column, p.currentToken.value)
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
	typ      tokenType
	value    string
	line     int    // 标记的起始行号
	column   int    // 标记的起始列号
	filename string // 标记所在的文件名
}

// Lexer performs lexical analysis of IDL files
type lexer struct {
	reader   *bufio.Reader
	current  rune
	eof      bool
	line     int    // 当前行号
	column   int    // 当前列号
	lastLine int    // 上一个标记的行号
	lastCol  int    // 上一个标记的列号
	filename string // 当前处理的文件名
}

// newLexer creates a new lexer
func newLexer(r io.Reader, filename string) *lexer {
	lex := &lexer{
		reader:   bufio.NewReader(r),
		line:     1, // 行号从1开始
		column:   0, // 列号从0开始
		lastLine: 1,
		lastCol:  0,
		filename: filename,
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
		return &token{
			typ:      tokenEOF,
			value:    "",
			line:     l.line,
			column:   l.column,
			filename: l.filename,
		}, nil
	}

	// 保存标记的起始位置
	l.lastLine = l.line
	l.lastCol = l.column

	// Process token
	switch {
	case l.current == '{':
		l.readChar()
		return &token{typ: tokenOpenBrace, value: "{", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == '}':
		l.readChar()
		return &token{typ: tokenCloseBrace, value: "}", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == '(':
		l.readChar()
		return &token{typ: tokenOpenParen, value: "(", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == ')':
		l.readChar()
		return &token{typ: tokenCloseParen, value: ")", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == '[':
		l.readChar()
		return &token{typ: tokenOpenBracket, value: "[", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == ']':
		l.readChar()
		return &token{typ: tokenCloseBracket, value: "]", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == ';':
		l.readChar()
		return &token{typ: tokenSemicolon, value: ";", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == ':':
		l.readChar()
		return &token{typ: tokenColon, value: ":", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
	case l.current == ',':
		l.readChar()
		return &token{typ: tokenComma, value: ",", line: l.lastLine, column: l.lastCol, filename: l.filename}, nil
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
		return nil, fmt.Errorf("%s:%d:%d: unexpected character: %c", l.filename, l.lastLine, l.lastCol, l.current)
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

	return &token{
		typ:      tokenPreprocessor,
		value:    directive.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
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

	return &token{
		typ:      tokenIdentifier,
		value:    ident.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
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

	return &token{
		typ:      tokenNumber,
		value:    num.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
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
				return nil, fmt.Errorf("%s:%d:%d: unterminated string literal", l.filename, l.lastLine, l.lastCol)
			}
		}
		str.WriteRune(l.current)
		l.readChar()
	}

	// Skip the closing quote
	if l.eof {
		return nil, fmt.Errorf("%s:%d:%d: unterminated string literal", l.filename, l.lastLine, l.lastCol)
	}
	l.readChar()

	return &token{
		typ:      tokenString,
		value:    str.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
}

// readCharLiteral reads a character literal
func (l *lexer) readCharLiteral() (*token, error) {
	var ch strings.Builder
	// Skip the opening quote
	l.readChar()

	if l.eof {
		return nil, fmt.Errorf("%s:%d:%d: unterminated character literal", l.filename, l.lastLine, l.lastCol)
	}

	// Handle escape sequence
	if l.current == '\\' {
		ch.WriteRune(l.current)
		l.readChar()
		if l.eof {
			return nil, fmt.Errorf("%s:%d:%d: unterminated character literal", l.filename, l.lastLine, l.lastCol)
		}
	}
	ch.WriteRune(l.current)
	l.readChar()

	// Skip the closing quote
	if l.current != '\'' {
		return nil, fmt.Errorf("%s:%d:%d: unterminated character literal", l.filename, l.lastLine, l.lastCol)
	}
	l.readChar()

	return &token{
		typ:      tokenChar,
		value:    ch.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
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

	return &token{
		typ:      tokenOperator,
		value:    op.String(),
		line:     l.lastLine,
		column:   l.lastCol,
		filename: l.filename,
	}, nil
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
