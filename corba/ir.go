// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"fmt"
)

// Common Interface Repository errors
var (
	ErrInterfaceNotFound     = errors.New("interface not found")
	ErrInvalidInterfaceID    = errors.New("invalid interface ID")
	ErrOperationNotDefined   = errors.New("operation not defined")
	ErrParameterNotDefined   = errors.New("parameter not defined")
	ErrAttributeNotDefined   = errors.New("attribute not defined")
	ErrModuleNotFound        = errors.New("module not found")
	ErrDuplicateDefinition   = errors.New("duplicate definition")
	ErrInvalidDefinitionKind = errors.New("invalid definition kind")
)

// DefinitionKind defines the type of Interface Repository object
type DefinitionKind int

const (
	// Definition kinds as per CORBA specification
	DK_NONE DefinitionKind = iota
	DK_ALL
	DK_ATTRIBUTE
	DK_CONSTANT
	DK_EXCEPTION
	DK_INTERFACE
	DK_MODULE
	DK_OPERATION
	DK_TYPEDEF
	DK_ALIAS
	DK_STRUCT
	DK_UNION
	DK_ENUM
	DK_PRIMITIVE
	DK_STRING
	DK_SEQUENCE
	DK_ARRAY
	DK_REPOSITORY
	DK_WSTRING
	DK_FIXED
	DK_VALUE
	DK_VALUE_BOX
	DK_NATIVE
	DK_ABSTRACT_INTERFACE
	DK_LOCAL_INTERFACE
)

// String returns the string representation of the DefinitionKind
func (dk DefinitionKind) String() string {
	switch dk {
	case DK_NONE:
		return "NONE"
	case DK_ALL:
		return "ALL"
	case DK_ATTRIBUTE:
		return "ATTRIBUTE"
	case DK_CONSTANT:
		return "CONSTANT"
	case DK_EXCEPTION:
		return "EXCEPTION"
	case DK_INTERFACE:
		return "INTERFACE"
	case DK_MODULE:
		return "MODULE"
	case DK_OPERATION:
		return "OPERATION"
	case DK_TYPEDEF:
		return "TYPEDEF"
	case DK_ALIAS:
		return "ALIAS"
	case DK_STRUCT:
		return "STRUCT"
	case DK_UNION:
		return "UNION"
	case DK_ENUM:
		return "ENUM"
	case DK_PRIMITIVE:
		return "PRIMITIVE"
	case DK_STRING:
		return "STRING"
	case DK_SEQUENCE:
		return "SEQUENCE"
	case DK_ARRAY:
		return "ARRAY"
	case DK_REPOSITORY:
		return "REPOSITORY"
	case DK_WSTRING:
		return "WSTRING"
	case DK_FIXED:
		return "FIXED"
	case DK_VALUE:
		return "VALUE"
	case DK_VALUE_BOX:
		return "VALUE_BOX"
	case DK_NATIVE:
		return "NATIVE"
	case DK_ABSTRACT_INTERFACE:
		return "ABSTRACT_INTERFACE"
	case DK_LOCAL_INTERFACE:
		return "LOCAL_INTERFACE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", dk)
	}
}

// ParameterMode defines the direction of a parameter
type ParameterMode int

const (
	PARAM_IN    ParameterMode = iota // Input parameter
	PARAM_OUT                        // Output parameter
	PARAM_INOUT                      // Input/Output parameter
)

// String returns the string representation of the ParameterMode
func (pm ParameterMode) String() string {
	switch pm {
	case PARAM_IN:
		return "IN"
	case PARAM_OUT:
		return "OUT"
	case PARAM_INOUT:
		return "INOUT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", pm)
	}
}

// IRObject is the base interface for all Interface Repository objects
type IRObject interface {
	// Get the DefinitionKind of this IRObject
	DefKind() DefinitionKind

	// Get the repository ID of this IRObject
	Id() string

	// Get the name of this IRObject
	Name() string

	// Get the container of this IRObject
	Container() Container

	// Get a description of this IRObject
	Describe() string
}

// Container is an interface for IRObjects that can contain other IRObjects
type Container interface {
	IRObject

	// Get the list of contained objects
	Contents(limit DefinitionKind) []IRObject

	// Lookup an object by name
	Lookup(name string) (IRObject, error)

	// Get the list of contained objects that match the specified name
	LookupName(search_name string, levels int, limit DefinitionKind) []IRObject

	// Add an object to this container
	Add(obj IRObject) error
}

// Contained is an interface for IRObjects that can be contained within Containers
type Contained interface {
	IRObject

	// Move this object into another container
	Move(new_container Container, new_name string) error
}

// Repository is the root interface of the Interface Repository
type Repository interface {
	Container

	// Lookup an object by its repository ID
	LookupId(id string) (IRObject, error)

	// Create a new IDL Module
	CreateModule(id string, name string) (ModuleDef, error)

	// Create a new IDL Interface
	CreateInterface(id string, name string) (InterfaceDef, error)

	// Create a new IDL Struct
	CreateStruct(id string, name string) (StructDef, error)

	// Create a new IDL Exception
	CreateException(id string, name string) (ExceptionDef, error)

	// Create a new IDL Enum
	CreateEnum(id string, name string) (EnumDef, error)

	// Create a new IDL Union
	CreateUnion(id string, name string) (UnionDef, error)

	// Create a new IDL Alias
	CreateAlias(id string, name string, original TypeCode) (AliasDef, error)

	// Destroy the repository
	Destroy() error
}

// ModuleDef defines an IDL Module in the Interface Repository
type ModuleDef interface {
	Container
	Contained
}

// InterfaceDef defines an IDL Interface in the Interface Repository
type InterfaceDef interface {
	Container
	Contained

	// Get the list of base interfaces
	BaseInterfaces() []InterfaceDef

	// Create a new IDL Attribute
	CreateAttribute(id string, name string, type_code TypeCode, mode int) (AttributeDef, error)

	// Create a new IDL Operation
	CreateOperation(id string, name string, result TypeCode, mode int) (OperationDef, error)
}

// OperationDef defines an IDL Operation in the Interface Repository
type OperationDef interface {
	Contained

	// Get the result type of the operation
	Result() TypeCode

	// Get the list of parameters
	Params() []ParameterDescription

	// Get the list of exceptions
	Exceptions() []ExceptionDef

	// Add a parameter to the operation
	AddParameter(name string, type_code TypeCode, mode ParameterMode) error

	// Add an exception to the operation
	AddException(except ExceptionDef) error
}

// AttributeDef defines an IDL Attribute in the Interface Repository
type AttributeDef interface {
	Contained

	// Get the type of the attribute
	Type() TypeCode

	// Get the mode of the attribute (readonly or not)
	Mode() int
}

// ParameterDescription describes an operation parameter
type ParameterDescription struct {
	Name string
	Type TypeCode
	Mode ParameterMode
}

// TypeCode represents the type system of CORBA
type TypeCode interface {
	// Get the kind of this type
	Kind() DefinitionKind

	// Get the ID of this type
	Id() string

	// Get the name of this type
	Name() string

	// Check if this type is equal to another type
	Equal(TypeCode) bool

	// Get a string representation of this type
	String() string
}

// StructDef defines an IDL Struct in the Interface Repository
type StructDef interface {
	Contained
	TypeCode

	// Get the list of members in this struct
	Members() []StructMember

	// Add a member to this struct
	AddMember(name string, type_code TypeCode) error
}

// StructMember describes a member of a struct
type StructMember struct {
	Name string
	Type TypeCode
}

// ExceptionDef defines an IDL Exception in the Interface Repository
type ExceptionDef interface {
	Contained
	TypeCode

	// Get the list of members in this exception
	Members() []StructMember

	// Add a member to this exception
	AddMember(name string, type_code TypeCode) error
}

// UnionDef defines an IDL Union in the Interface Repository
type UnionDef interface {
	Contained
	TypeCode

	// Get the discriminator type for this union
	Discriminator() TypeCode

	// Get the list of members in this union
	Members() []UnionMember

	// Add a member to this union
	AddMember(name string, label interface{}, type_code TypeCode) error
}

// UnionMember describes a member of a union
type UnionMember struct {
	Name  string
	Label interface{} // The discriminator value
	Type  TypeCode
}

// EnumDef defines an IDL Enum in the Interface Repository
type EnumDef interface {
	Contained
	TypeCode

	// Get the list of members in this enum
	Members() []string

	// Add a member to this enum
	AddMember(name string) error
}

// AliasDef defines an IDL Alias (typedef) in the Interface Repository
type AliasDef interface {
	Contained
	TypeCode

	// Get the original type that is being aliased
	OriginalType() TypeCode
}

// InterfaceRepository provides access to the Interface Repository service
type InterfaceRepository interface {
	// Get the repository root
	GetRepository() Repository

	// Register a servant with the IR
	RegisterServant(servant interface{}, id string) error

	// Lookup an interface by ID
	LookupInterface(id string) (InterfaceDef, error)

	// Get the implementation repository ID for an object
	GetRepositoryID(obj interface{}) (string, error)

	// Check if an object implements a specific interface
	IsA(obj interface{}, interfaceID string) (bool, error)

	// Get the list of all interfaces that an object implements
	GetInterfaces(obj interface{}) ([]string, error)
}
