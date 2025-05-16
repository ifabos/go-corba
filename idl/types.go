// Package idl provides support for CORBA IDL (Interface Definition Language)
// including parsing and code generation functionality
package idl

import (
	"fmt"
	"strings"
)

// BasicType represents a primitive IDL type
type BasicType string

// IDL Basic Types
const (
	TypeShort     BasicType = "short"
	TypeLong      BasicType = "long"
	TypeLongLong  BasicType = "long long"
	TypeUShort    BasicType = "unsigned short"
	TypeULong     BasicType = "unsigned long"
	TypeULongLong BasicType = "unsigned long long"
	TypeFloat     BasicType = "float"
	TypeDouble    BasicType = "double"
	TypeBoolean   BasicType = "boolean"
	TypeChar      BasicType = "char"
	TypeWChar     BasicType = "wchar"
	TypeOctet     BasicType = "octet"
	TypeAny       BasicType = "any"
	TypeString    BasicType = "string"
	TypeWString   BasicType = "wstring"
	TypeVoid      BasicType = "void"
)

// Type is the interface for all IDL types
type Type interface {
	TypeName() string
	GoTypeName() string
	GetRepositoryID() string
	SetRepositoryID(string)
}

// Direction represents the parameter direction in IDL operations
type Direction string

// Parameter direction constants
const (
	In    Direction = "in"
	Out   Direction = "out"
	InOut Direction = "inout"
)

// BaseType 提供基本的类型功能，可以被其他类型嵌入
type BaseType struct {
	RepositoryID string
}

// GetRepositoryID returns the repository ID
func (t *BaseType) GetRepositoryID() string {
	return t.RepositoryID
}

// SetRepositoryID sets the repository ID
func (t *BaseType) SetRepositoryID(repoID string) {
	t.RepositoryID = repoID
}

// SimpleType represents a basic IDL type
type SimpleType struct {
	BaseType
	Name BasicType
}

// TypeName returns the IDL type name
func (t *SimpleType) TypeName() string {
	return string(t.Name)
}

// GoTypeName returns the corresponding Go type name
func (t *SimpleType) GoTypeName() string {
	switch t.Name {
	case TypeShort:
		return "int16"
	case TypeLong:
		return "int32"
	case TypeLongLong:
		return "int64"
	case TypeUShort:
		return "uint16"
	case TypeULong:
		return "uint32"
	case TypeULongLong:
		return "uint64"
	case TypeFloat:
		return "float32"
	case TypeDouble:
		return "float64"
	case TypeBoolean:
		return "bool"
	case TypeChar:
		return "byte"
	case TypeWChar:
		return "rune"
	case TypeOctet:
		return "byte"
	case TypeAny:
		return "interface{}"
	case TypeString:
		return "string"
	case TypeWString:
		return "string"
	case TypeVoid:
		return ""
	default:
		// For other types, just use the IDL name
		return string(t.Name)
	}
}

// SequenceType represents an IDL sequence type
type SequenceType struct {
	BaseType
	ElementType Type
	MaxSize     int // -1 for unbounded
}

// TypeName returns the IDL type name
func (t *SequenceType) TypeName() string {
	if t.MaxSize < 0 {
		return fmt.Sprintf("sequence<%s>", t.ElementType.TypeName())
	}
	return fmt.Sprintf("sequence<%s, %d>", t.ElementType.TypeName(), t.MaxSize)
}

// GoTypeName returns the corresponding Go type name
func (t *SequenceType) GoTypeName() string {
	return "[]" + t.ElementType.GoTypeName()
}

// StructType represents an IDL struct type
type StructType struct {
	BaseType
	Name   string
	Module string
	Fields []StructField
}

// StructField represents a field in an IDL struct
type StructField struct {
	Name string
	Type Type
}

// TypeName returns the IDL type name
func (t *StructType) TypeName() string {
	return t.Name
}

// GoTypeName returns the corresponding Go type name
func (t *StructType) GoTypeName() string {
	return t.Name
}

// EnumType represents an IDL enum type
type EnumType struct {
	BaseType
	Name     string
	Module   string
	Elements []string
}

// TypeName returns the IDL type name
func (t *EnumType) TypeName() string {
	return t.Name
}

// GoTypeName returns the corresponding Go type name
func (t *EnumType) GoTypeName() string {
	return t.Name
}

// TypeDef represents an IDL typedef
type TypeDef struct {
	BaseType
	Name     string
	Module   string
	OrigType Type
}

// TypeName returns the IDL type name
func (t *TypeDef) TypeName() string {
	return t.Name
}

// GoTypeName returns the corresponding Go type name
func (t *TypeDef) GoTypeName() string {
	return t.Name
}

// UnionType represents an IDL union type
type UnionType struct {
	BaseType
	Name         string
	Module       string
	Discriminant Type
	Cases        []UnionCase
}

// UnionCase represents a case in an IDL union
type UnionCase struct {
	Labels []string
	Name   string
	Type   Type
}

// TypeName returns the IDL type name
func (t *UnionType) TypeName() string {
	return t.Name
}

// GoTypeName returns the corresponding Go type name
func (t *UnionType) GoTypeName() string {
	return t.Name
}

// InterfaceType represents an IDL interface type
type InterfaceType struct {
	BaseType
	Name       string
	Module     string
	Parents    []string
	Operations []Operation
	Attributes []Attribute
	Types      map[string]Type // Support for nested enums/types
}

// Operation represents an operation in an IDL interface
type Operation struct {
	Name       string
	ReturnType Type
	Parameters []Parameter
	Raises     []string
	Oneway     bool
}

// Parameter represents a parameter in an IDL operation
type Parameter struct {
	Name      string
	Type      Type
	Direction Direction
}

// Attribute represents an attribute in an IDL interface
type Attribute struct {
	Name     string
	Type     Type
	Readonly bool
}

// TypeName returns the IDL type name
func (t *InterfaceType) TypeName() string {
	return t.Name
}

// GoTypeName returns the corresponding Go type name
func (t *InterfaceType) GoTypeName() string {
	return t.Name
}

// Module represents an IDL module that contains types
type Module struct {
	Name       string
	Parent     *Module
	Types      map[string]Type
	Submodules map[string]*Module
	Prefix     string // Repository ID prefix for this module
}

// NewModule creates a new IDL module
func NewModule(name string) *Module {
	return &Module{
		Name:       name,
		Types:      make(map[string]Type),
		Submodules: make(map[string]*Module),
	}
}

// AddSubmodule adds a submodule with the given name
func (m *Module) AddSubmodule(name string) *Module {
	submodule := NewModule(name)
	submodule.Parent = m
	m.Submodules[name] = submodule
	return submodule
}

// GetSubmodule gets a submodule by name
func (m *Module) GetSubmodule(name string) (*Module, bool) {
	submodule, exists := m.Submodules[name]
	return submodule, exists
}

// AddType adds a type to the module
func (m *Module) AddType(name string, typ Type) {
	m.Types[name] = typ
}

// GetType gets a type by name
func (m *Module) GetType(name string) (Type, bool) {
	typ, exists := m.Types[name]
	return typ, exists
}

// FullName returns the fully qualified module name
func (m *Module) FullName() string {
	if m.Parent == nil || m.Parent.Name == "" {
		return m.Name
	}
	return m.Parent.FullName() + "::" + m.Name
}

// Path returns the module path as a slice of names
func (m *Module) Path() []string {
	if m.Name == "" {
		return []string{}
	}

	if m.Parent == nil || m.Parent.Name == "" {
		return []string{m.Name}
	}

	return append(m.Parent.Path(), m.Name)
}

// AllTypes returns all types in the module and its submodules
func (m *Module) AllTypes() map[string]Type {
	result := make(map[string]Type)

	// Add types from this module
	for name, typ := range m.Types {
		result[name] = typ
	}

	// Add types from submodules with qualified names
	for subName, submodule := range m.Submodules {
		for name, typ := range submodule.AllTypes() {
			result[subName+"::"+name] = typ
		}
	}

	return result
}

// GenerateRepositoryID 根据模块前缀和类型名生成标准的 Repository ID
func GenerateRepositoryID(modulePath []string, typeName string, version string) string {
	if version == "" {
		version = "1.0"
	}

	// 构建作用域路径
	path := ""
	for i, mod := range modulePath {
		if i > 0 {
			path += "/"
		}
		path += mod
	}

	// 添加类型名
	if path != "" {
		path += "/" + typeName
	} else {
		path = typeName
	}

	// 返回 "IDL:路径:版本" 格式的 Repository ID
	return fmt.Sprintf("IDL:%s:%s", path, version)
}

// BuildRepositoryID 根据模块层次结构构建 Repository ID
func (m *Module) BuildRepositoryID(typeName string, version string) string {
	path := m.Path()
	return GenerateRepositoryID(path, typeName, version)
}

// GoPackageName returns the Go package name for this module
func (m *Module) GoPackageName() string {
	if m.Name == "" {
		return "main"
	}
	return strings.ToLower(m.Name)
}
