// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Common type management errors
var (
	ErrInvalidType     = errors.New("invalid CORBA type")
	ErrTypeMismatch    = errors.New("type mismatch in conversion")
	ErrInvalidTypeCode = errors.New("invalid TypeCode")
	ErrInvalidAnyValue = errors.New("invalid Any value")
	ErrUnsupportedType = errors.New("unsupported CORBA type")
)

// TCKind is an enumeration of CORBA type kinds (more specific than DefinitionKind)
type TCKind int

// CORBA TCKind constants as defined in the CORBA specification
const (
	// Primitive types
	TC_NULL TCKind = iota
	TC_VOID
	TC_SHORT
	TC_LONG
	TC_USHORT
	TC_ULONG
	TC_FLOAT
	TC_DOUBLE
	TC_BOOLEAN
	TC_CHAR
	TC_OCTET
	TC_ANY
	TC_TYPECODE
	TC_PRINCIPAL
	TC_OBJREF
	TC_STRUCT
	TC_UNION
	TC_ENUM
	TC_STRING
	TC_SEQUENCE
	TC_ARRAY
	TC_ALIAS
	TC_EXCEPT
	TC_LONGLONG
	TC_ULONGLONG
	TC_LONGDOUBLE
	TC_WCHAR
	TC_WSTRING
	TC_FIXED
	TC_VALUE
	TC_VALUE_BOX
	TC_NATIVE
	TC_ABSTRACT_INTERFACE
	TC_LOCAL_INTERFACE
)

// String returns the string representation of the TCKind
func (k TCKind) String() string {
	switch k {
	case TC_NULL:
		return "null"
	case TC_VOID:
		return "void"
	case TC_SHORT:
		return "short"
	case TC_LONG:
		return "long"
	case TC_USHORT:
		return "unsigned short"
	case TC_ULONG:
		return "unsigned long"
	case TC_FLOAT:
		return "float"
	case TC_DOUBLE:
		return "double"
	case TC_BOOLEAN:
		return "boolean"
	case TC_CHAR:
		return "char"
	case TC_OCTET:
		return "octet"
	case TC_ANY:
		return "any"
	case TC_TYPECODE:
		return "TypeCode"
	case TC_PRINCIPAL:
		return "Principal"
	case TC_OBJREF:
		return "Object"
	case TC_STRUCT:
		return "struct"
	case TC_UNION:
		return "union"
	case TC_ENUM:
		return "enum"
	case TC_STRING:
		return "string"
	case TC_SEQUENCE:
		return "sequence"
	case TC_ARRAY:
		return "array"
	case TC_ALIAS:
		return "alias"
	case TC_EXCEPT:
		return "exception"
	case TC_LONGLONG:
		return "long long"
	case TC_ULONGLONG:
		return "unsigned long long"
	case TC_LONGDOUBLE:
		return "long double"
	case TC_WCHAR:
		return "wchar"
	case TC_WSTRING:
		return "wstring"
	case TC_FIXED:
		return "fixed"
	case TC_VALUE:
		return "value"
	case TC_VALUE_BOX:
		return "value box"
	case TC_NATIVE:
		return "native"
	case TC_ABSTRACT_INTERFACE:
		return "abstract interface"
	case TC_LOCAL_INTERFACE:
		return "local interface"
	default:
		return fmt.Sprintf("unknown(%d)", k)
	}
}

// TypeCodeImpl enhances the TypeCode interface with more functionality as per CORBA spec
type TypeCodeImpl interface {
	TypeCode

	// TCKind returns the more specific TCKind
	TCKind() TCKind

	// Param gets a parameter value by index
	Param(int) (interface{}, error)

	// ParameterCount returns the number of parameters
	ParameterCount() int

	// Content returns the content TypeCode (for container types)
	ContentType() (TypeCode, error)

	// MemberCount returns the number of members (for structs, unions, enums)
	MemberCount() int

	// MemberName returns the name of a member
	MemberName(index int) (string, error)

	// MemberType returns the type of a member
	MemberType(index int) (TypeCode, error)

	// MemberLabel returns the label of a union member
	MemberLabel(index int) (interface{}, error)

	// DiscriminatorType returns the discriminator type of a union
	DiscriminatorType() (TypeCode, error)

	// DefaultIndex returns the default case index for a union
	DefaultIndex() int

	// Length returns the bound for strings, sequences, arrays
	Length() int
}

// Any represents the CORBA any type, which can hold any CORBA type
type Any struct {
	typeCode TypeCode
	value    interface{}
}

// NewAny creates a new Any from a value
func NewAny(value interface{}) (*Any, error) {
	tc, err := TypeCodeFromValue(value)
	if err != nil {
		return nil, err
	}

	return &Any{
		typeCode: tc,
		value:    value,
	}, nil
}

// NewAnyWithTypeCode creates a new Any with a specific TypeCode
func NewAnyWithTypeCode(tc TypeCode, value interface{}) (*Any, error) {
	// Validate that value matches the TypeCode
	if !validateTypeCodeMatch(tc, value) {
		return nil, ErrTypeMismatch
	}

	return &Any{
		typeCode: tc,
		value:    value,
	}, nil
}

// TypeCode returns the TypeCode of the Any value
func (a *Any) TypeCode() TypeCode {
	return a.typeCode
}

// Value returns the value contained in the Any
func (a *Any) Value() interface{} {
	return a.value
}

// ExtractValue extracts the value to a destination variable
func (a *Any) ExtractValue(dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}

	srcVal := reflect.ValueOf(a.value)
	destElem := destVal.Elem()

	// Check if types are compatible
	if !destElem.Type().AssignableTo(srcVal.Type()) {
		return ErrTypeMismatch
	}

	destElem.Set(srcVal)
	return nil
}

// String returns a string representation of the Any
func (a *Any) String() string {
	return fmt.Sprintf("Any(%s, %v)", a.typeCode.String(), a.value)
}

// TypeCodeRegistry manages TypeCode instances
type TypeCodeRegistry struct {
	mu             sync.RWMutex
	basicTypeCodes map[TCKind]TypeCodeImpl
	customTypes    map[string]TypeCodeImpl
}

// Global TypeCode registry
var globalTypeRegistry *TypeCodeRegistry

func init() {
	globalTypeRegistry = NewTypeCodeRegistry()
	globalTypeRegistry.initializeBasicTypes()
}

// NewTypeCodeRegistry creates a new TypeCode registry
func NewTypeCodeRegistry() *TypeCodeRegistry {
	return &TypeCodeRegistry{
		basicTypeCodes: make(map[TCKind]TypeCodeImpl),
		customTypes:    make(map[string]TypeCodeImpl),
	}
}

// initializeBasicTypes initializes the basic CORBA types
func (r *TypeCodeRegistry) initializeBasicTypes() {
	// Primitive types mapping to Go types
	primitives := []struct {
		kind  TCKind
		dk    DefinitionKind
		id    string
		name  string
		goTyp reflect.Type
	}{
		{TC_SHORT, DK_PRIMITIVE, "IDL:omg.org/CORBA/Short:1.0", "short", reflect.TypeOf(int16(0))},
		{TC_LONG, DK_PRIMITIVE, "IDL:omg.org/CORBA/Long:1.0", "long", reflect.TypeOf(int32(0))},
		{TC_USHORT, DK_PRIMITIVE, "IDL:omg.org/CORBA/UShort:1.0", "unsigned short", reflect.TypeOf(uint16(0))},
		{TC_ULONG, DK_PRIMITIVE, "IDL:omg.org/CORBA/ULong:1.0", "unsigned long", reflect.TypeOf(uint32(0))},
		{TC_FLOAT, DK_PRIMITIVE, "IDL:omg.org/CORBA/Float:1.0", "float", reflect.TypeOf(float32(0))},
		{TC_DOUBLE, DK_PRIMITIVE, "IDL:omg.org/CORBA/Double:1.0", "double", reflect.TypeOf(float64(0))},
		{TC_BOOLEAN, DK_PRIMITIVE, "IDL:omg.org/CORBA/Boolean:1.0", "boolean", reflect.TypeOf(bool(false))},
		{TC_CHAR, DK_PRIMITIVE, "IDL:omg.org/CORBA/Char:1.0", "char", reflect.TypeOf(byte(0))},
		{TC_OCTET, DK_PRIMITIVE, "IDL:omg.org/CORBA/Octet:1.0", "octet", reflect.TypeOf(byte(0))},
		{TC_STRING, DK_STRING, "IDL:omg.org/CORBA/String:1.0", "string", reflect.TypeOf("")},
		{TC_LONGLONG, DK_PRIMITIVE, "IDL:omg.org/CORBA/LongLong:1.0", "long long", reflect.TypeOf(int64(0))},
		{TC_ULONGLONG, DK_PRIMITIVE, "IDL:omg.org/CORBA/ULongLong:1.0", "unsigned long long", reflect.TypeOf(uint64(0))},
		// Add TC_ANY with proper reflection of *Any type
		{TC_ANY, DK_PRIMITIVE, "IDL:omg.org/CORBA/Any:1.0", "any", reflect.TypeOf((*Any)(nil))},
	}

	for _, p := range primitives {
		tc := &basicTypeCode{
			typeCodeBase: typeCodeBase{
				id:   p.id,
				name: p.name,
				kind: p.dk,
			},
			tcKind: p.kind,
			goType: p.goTyp,
		}

		r.basicTypeCodes[p.kind] = tc
		r.customTypes[p.id] = tc
	}
}

// GetBasicTypeCode returns a TypeCode for a basic CORBA type
func (r *TypeCodeRegistry) GetBasicTypeCode(kind TCKind) (TypeCodeImpl, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tc, exists := r.basicTypeCodes[kind]
	if !exists {
		return nil, ErrInvalidTypeCode
	}

	return tc, nil
}

// GetTypeCode returns a TypeCode by its ID
func (r *TypeCodeRegistry) GetTypeCode(id string) (TypeCodeImpl, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tc, exists := r.customTypes[id]
	if !exists {
		return nil, ErrInvalidTypeCode
	}

	return tc, nil
}

// RegisterTypeCode registers a TypeCode in the registry
func (r *TypeCodeRegistry) RegisterTypeCode(tc TypeCodeImpl) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.customTypes[tc.Id()] = tc

	// If it's a basic type, also register it by TCKind
	if basic, ok := tc.(*basicTypeCode); ok {
		r.basicTypeCodes[basic.tcKind] = basic
	}
}

// GetOrCreateStructTypeCode creates a new struct TypeCode if it doesn't exist
func (r *TypeCodeRegistry) GetOrCreateStructTypeCode(id string, name string) (*structTypeCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tc, exists := r.customTypes[id]; exists {
		if stc, ok := tc.(*structTypeCode); ok {
			return stc, nil
		}
		return nil, fmt.Errorf("TypeCode with ID %s exists but is not a struct", id)
	}

	stc := &structTypeCode{
		typeCodeBase: typeCodeBase{
			id:   id,
			name: name,
			kind: DK_STRUCT,
		},
		tcKind:      TC_STRUCT,
		members:     make([]StructMember, 0),
		memberTypes: make([]TypeCode, 0),
	}

	r.customTypes[id] = stc
	return stc, nil
}

// GetOrCreateSequenceTypeCode creates a new sequence TypeCode
func (r *TypeCodeRegistry) GetOrCreateSequenceTypeCode(id string, name string, elementType TypeCode, bound int) (*sequenceTypeCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tc, exists := r.customTypes[id]; exists {
		if stc, ok := tc.(*sequenceTypeCode); ok {
			return stc, nil
		}
		return nil, fmt.Errorf("TypeCode with ID %s exists but is not a sequence", id)
	}

	stc := &sequenceTypeCode{
		typeCodeBase: typeCodeBase{
			id:   id,
			name: name,
			kind: DK_SEQUENCE,
		},
		tcKind:      TC_SEQUENCE,
		elementType: elementType,
		bound:       bound,
	}

	r.customTypes[id] = stc
	return stc, nil
}

// GetOrCreateEnumTypeCode creates a new enum TypeCode
func (r *TypeCodeRegistry) GetOrCreateEnumTypeCode(id string, name string) (*enumTypeCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tc, exists := r.customTypes[id]; exists {
		if etc, ok := tc.(*enumTypeCode); ok {
			return etc, nil
		}
		return nil, fmt.Errorf("TypeCode with ID %s exists but is not an enum", id)
	}

	etc := &enumTypeCode{
		typeCodeBase: typeCodeBase{
			id:   id,
			name: name,
			kind: DK_ENUM,
		},
		tcKind:  TC_ENUM,
		members: make([]string, 0),
	}

	r.customTypes[id] = etc
	return etc, nil
}

// GetOrCreateUnionTypeCode creates a new union TypeCode
func (r *TypeCodeRegistry) GetOrCreateUnionTypeCode(id string, name string, discriminatorType TypeCode) (*unionTypeCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tc, exists := r.customTypes[id]; exists {
		if utc, ok := tc.(*unionTypeCode); ok {
			return utc, nil
		}
		return nil, fmt.Errorf("TypeCode with ID %s exists but is not a union", id)
	}

	utc := &unionTypeCode{
		typeCodeBase: typeCodeBase{
			id:   id,
			name: name,
			kind: DK_UNION,
		},
		tcKind:            TC_UNION,
		discriminatorType: discriminatorType,
		members:           make([]UnionMember, 0),
		memberTypes:       make([]TypeCode, 0),
		defaultIndex:      -1,
	}

	r.customTypes[id] = utc
	return utc, nil
}

// basicTypeCode represents a basic CORBA type
type basicTypeCode struct {
	typeCodeBase
	tcKind TCKind
	goType reflect.Type
}

// TCKind returns the TCKind of this type
func (b *basicTypeCode) TCKind() TCKind {
	return b.tcKind
}

// Param gets a parameter value by index
func (b *basicTypeCode) Param(index int) (interface{}, error) {
	return nil, errors.New("basic types have no parameters")
}

// ParameterCount returns the number of parameters
func (b *basicTypeCode) ParameterCount() int {
	return 0
}

// ContentType returns the content TypeCode (for container types)
func (b *basicTypeCode) ContentType() (TypeCode, error) {
	return nil, errors.New("basic types have no content type")
}

// MemberCount returns the number of members
func (b *basicTypeCode) MemberCount() int {
	return 0
}

// MemberName returns the name of a member
func (b *basicTypeCode) MemberName(index int) (string, error) {
	return "", errors.New("basic types have no members")
}

// MemberType returns the type of a member
func (b *basicTypeCode) MemberType(index int) (TypeCode, error) {
	return nil, errors.New("basic types have no members")
}

// MemberLabel returns the label of a union member
func (b *basicTypeCode) MemberLabel(index int) (interface{}, error) {
	return nil, errors.New("basic types have no members")
}

// DiscriminatorType returns the discriminator type of a union
func (b *basicTypeCode) DiscriminatorType() (TypeCode, error) {
	return nil, errors.New("basic types have no discriminator")
}

// DefaultIndex returns the default case index for a union
func (b *basicTypeCode) DefaultIndex() int {
	return -1
}

// Length returns the bound for strings, sequences, arrays
func (b *basicTypeCode) Length() int {
	return 0
}

// structTypeCode represents a struct type
type structTypeCode struct {
	typeCodeBase
	tcKind      TCKind
	members     []StructMember
	memberTypes []TypeCode
	mu          sync.RWMutex
}

// TCKind returns the TCKind of this type
func (s *structTypeCode) TCKind() TCKind {
	return s.tcKind
}

// AddMember adds a member to the struct
func (s *structTypeCode) AddMember(name string, typeCode TypeCode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.members = append(s.members, StructMember{
		Name: name,
		Type: typeCode,
	})
	s.memberTypes = append(s.memberTypes, typeCode)
}

// Param gets a parameter value by index
func (s *structTypeCode) Param(index int) (interface{}, error) {
	if index == 0 {
		return s.id, nil
	} else if index == 1 {
		return s.name, nil
	} else if index-2 < len(s.members) {
		return s.members[index-2].Name, nil
	} else if index-2-len(s.members) < len(s.memberTypes) {
		return s.memberTypes[index-2-len(s.members)], nil
	}
	return nil, fmt.Errorf("parameter index %d out of range", index)
}

// ParameterCount returns the number of parameters
func (s *structTypeCode) ParameterCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return 2 + 2*len(s.members) // id, name, and each member has a name and type
}

// ContentType returns the content TypeCode
func (s *structTypeCode) ContentType() (TypeCode, error) {
	return nil, errors.New("struct types have no content type")
}

// MemberCount returns the number of members
func (s *structTypeCode) MemberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.members)
}

// MemberName returns the name of a member
func (s *structTypeCode) MemberName(index int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.members) {
		return "", fmt.Errorf("member index %d out of range", index)
	}

	return s.members[index].Name, nil
}

// MemberType returns the type of a member
func (s *structTypeCode) MemberType(index int) (TypeCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.members) {
		return nil, fmt.Errorf("member index %d out of range", index)
	}

	return s.members[index].Type, nil
}

// MemberLabel returns the label of a union member
func (s *structTypeCode) MemberLabel(index int) (interface{}, error) {
	return nil, errors.New("struct types don't have member labels")
}

// DiscriminatorType returns the discriminator type of a union
func (s *structTypeCode) DiscriminatorType() (TypeCode, error) {
	return nil, errors.New("struct types don't have discriminators")
}

// DefaultIndex returns the default case index for a union
func (s *structTypeCode) DefaultIndex() int {
	return -1
}

// Length returns the bound for strings, sequences, arrays
func (s *structTypeCode) Length() int {
	return 0
}

// sequenceTypeCode represents a sequence type
type sequenceTypeCode struct {
	typeCodeBase
	tcKind      TCKind
	elementType TypeCode
	bound       int
}

// TCKind returns the TCKind of this type
func (s *sequenceTypeCode) TCKind() TCKind {
	return s.tcKind
}

// Param gets a parameter value by index
func (s *sequenceTypeCode) Param(index int) (interface{}, error) {
	if index == 0 {
		return s.elementType, nil
	} else if index == 1 {
		return s.bound, nil
	}
	return nil, fmt.Errorf("parameter index %d out of range", index)
}

// ParameterCount returns the number of parameters
func (s *sequenceTypeCode) ParameterCount() int {
	return 2 // element type and bound
}

// ContentType returns the content TypeCode
func (s *sequenceTypeCode) ContentType() (TypeCode, error) {
	return s.elementType, nil
}

// MemberCount returns the number of members
func (s *sequenceTypeCode) MemberCount() int {
	return 0
}

// MemberName returns the name of a member
func (s *sequenceTypeCode) MemberName(index int) (string, error) {
	return "", errors.New("sequence types have no named members")
}

// MemberType returns the type of a member
func (s *sequenceTypeCode) MemberType(index int) (TypeCode, error) {
	return nil, errors.New("sequence types have no members")
}

// MemberLabel returns the label of a union member
func (s *sequenceTypeCode) MemberLabel(index int) (interface{}, error) {
	return nil, errors.New("sequence types have no member labels")
}

// DiscriminatorType returns the discriminator type of a union
func (s *sequenceTypeCode) DiscriminatorType() (TypeCode, error) {
	return nil, errors.New("sequence types have no discriminator")
}

// DefaultIndex returns the default case index for a union
func (s *sequenceTypeCode) DefaultIndex() int {
	return -1
}

// Length returns the bound for strings, sequences, arrays
func (s *sequenceTypeCode) Length() int {
	return s.bound
}

// enumTypeCode represents an enum type
type enumTypeCode struct {
	typeCodeBase
	tcKind  TCKind
	members []string
	mu      sync.RWMutex
}

// TCKind returns the TCKind of this type
func (e *enumTypeCode) TCKind() TCKind {
	return e.tcKind
}

// AddMember adds a member to the enum
func (e *enumTypeCode) AddMember(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.members = append(e.members, name)
}

// Param gets a parameter value by index
func (e *enumTypeCode) Param(index int) (interface{}, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if index == 0 {
		return e.id, nil
	} else if index == 1 {
		return e.name, nil
	} else if index-2 < len(e.members) {
		return e.members[index-2], nil
	}
	return nil, fmt.Errorf("parameter index %d out of range", index)
}

// ParameterCount returns the number of parameters
func (e *enumTypeCode) ParameterCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return 2 + len(e.members) // id, name, and each member name
}

// ContentType returns the content TypeCode
func (e *enumTypeCode) ContentType() (TypeCode, error) {
	return nil, errors.New("enum types have no content type")
}

// MemberCount returns the number of members
func (e *enumTypeCode) MemberCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.members)
}

// MemberName returns the name of a member
func (e *enumTypeCode) MemberName(index int) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if index < 0 || index >= len(e.members) {
		return "", fmt.Errorf("member index %d out of range", index)
	}

	return e.members[index], nil
}

// MemberType returns the type of a member
func (e *enumTypeCode) MemberType(index int) (TypeCode, error) {
	return nil, errors.New("enum members don't have types")
}

// MemberLabel returns the label of a union member
func (e *enumTypeCode) MemberLabel(index int) (interface{}, error) {
	return nil, errors.New("enum types don't have member labels")
}

// DiscriminatorType returns the discriminator type of a union
func (e *enumTypeCode) DiscriminatorType() (TypeCode, error) {
	return nil, errors.New("enum types don't have discriminators")
}

// DefaultIndex returns the default case index for a union
func (e *enumTypeCode) DefaultIndex() int {
	return -1
}

// Length returns the bound for strings, sequences, arrays
func (e *enumTypeCode) Length() int {
	return 0
}

// unionTypeCode represents a union type
type unionTypeCode struct {
	typeCodeBase
	tcKind            TCKind
	discriminatorType TypeCode
	members           []UnionMember
	memberTypes       []TypeCode
	defaultIndex      int
	mu                sync.RWMutex
}

// TCKind returns the TCKind of this type
func (u *unionTypeCode) TCKind() TCKind {
	return u.tcKind
}

// AddMember adds a member to the union
func (u *unionTypeCode) AddMember(name string, label interface{}, typeCode TypeCode) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.members = append(u.members, UnionMember{
		Name:  name,
		Label: label,
		Type:  typeCode,
	})
	u.memberTypes = append(u.memberTypes, typeCode)
}

// SetDefaultMember sets the default case for the union
func (u *unionTypeCode) SetDefaultMember(index int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if index < 0 || index >= len(u.members) {
		return fmt.Errorf("member index %d out of range", index)
	}

	u.defaultIndex = index
	return nil
}

// Param gets a parameter value by index
func (u *unionTypeCode) Param(index int) (interface{}, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if index == 0 {
		return u.id, nil
	} else if index == 1 {
		return u.name, nil
	} else if index == 2 {
		return u.discriminatorType, nil
	}
	// More parameters for members and their types
	return nil, fmt.Errorf("parameter index %d out of range", index)
}

// ParameterCount returns the number of parameters
func (u *unionTypeCode) ParameterCount() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	// id, name, discriminator, default index, and each member has a name, label, and type
	return 4 + 3*len(u.members)
}

// ContentType returns the content TypeCode
func (u *unionTypeCode) ContentType() (TypeCode, error) {
	return nil, errors.New("union types have no content type")
}

// MemberCount returns the number of members
func (u *unionTypeCode) MemberCount() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return len(u.members)
}

// MemberName returns the name of a member
func (u *unionTypeCode) MemberName(index int) (string, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if index < 0 || index >= len(u.members) {
		return "", fmt.Errorf("member index %d out of range", index)
	}

	return u.members[index].Name, nil
}

// MemberType returns the type of a member
func (u *unionTypeCode) MemberType(index int) (TypeCode, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if index < 0 || index >= len(u.members) {
		return nil, fmt.Errorf("member index %d out of range", index)
	}

	return u.members[index].Type, nil
}

// MemberLabel returns the label of a union member
func (u *unionTypeCode) MemberLabel(index int) (interface{}, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if index < 0 || index >= len(u.members) {
		return nil, fmt.Errorf("member index %d out of range", index)
	}

	return u.members[index].Label, nil
}

// DiscriminatorType returns the discriminator type of a union
func (u *unionTypeCode) DiscriminatorType() (TypeCode, error) {
	return u.discriminatorType, nil
}

// DefaultIndex returns the default case index for a union
func (u *unionTypeCode) DefaultIndex() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.defaultIndex
}

// Length returns the bound for strings, sequences, arrays
func (u *unionTypeCode) Length() int {
	return 0
}

// TypeCodeFromKind gets a TypeCode for a basic kind
func TypeCodeFromKind(kind TCKind) (TypeCode, error) {
	tc, err := globalTypeRegistry.GetBasicTypeCode(kind)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

// TypeCodeFromValue creates a TypeCode from a Go value
func TypeCodeFromValue(value interface{}) (TypeCode, error) {
	if value == nil {
		tc, err := TypeCodeFromKind(TC_NULL)
		if err != nil {
			return nil, err
		}
		return tc, nil
	}

	v := reflect.ValueOf(value)
	return typeCodeFromReflectValue(v)
}

// typeCodeFromReflectValue creates a TypeCode from a reflect.Value
func typeCodeFromReflectValue(v reflect.Value) (TypeCode, error) {
	switch v.Kind() {
	case reflect.Bool:
		return TypeCodeFromKind(TC_BOOLEAN)
	case reflect.Int8:
		return TypeCodeFromKind(TC_CHAR)
	case reflect.Int16:
		return TypeCodeFromKind(TC_SHORT)
	case reflect.Int32:
		return TypeCodeFromKind(TC_LONG)
	case reflect.Int64:
		return TypeCodeFromKind(TC_LONGLONG)
	case reflect.Uint8:
		return TypeCodeFromKind(TC_OCTET)
	case reflect.Uint16:
		return TypeCodeFromKind(TC_USHORT)
	case reflect.Uint32:
		return TypeCodeFromKind(TC_ULONG)
	case reflect.Uint64:
		return TypeCodeFromKind(TC_ULONGLONG)
	case reflect.Float32:
		return TypeCodeFromKind(TC_FLOAT)
	case reflect.Float64:
		return TypeCodeFromKind(TC_DOUBLE)
	case reflect.String:
		return TypeCodeFromKind(TC_STRING)
	case reflect.Ptr:
		// Handle *Any specifically
		if v.Type() == reflect.TypeOf((*Any)(nil)) {
			return TypeCodeFromKind(TC_ANY)
		}
		// Dereference pointers
		if v.IsNil() {
			return nil, errors.New("cannot create TypeCode from nil pointer")
		}
		return typeCodeFromReflectValue(v.Elem())
	case reflect.Slice, reflect.Array:
		elemType, err := typeCodeFromReflectValue(reflect.New(v.Type().Elem()).Elem())
		if err != nil {
			return nil, err
		}

		// Create a sequence TypeCode
		id := fmt.Sprintf("IDL:Sequence_%s:1.0", elemType.Id())
		name := fmt.Sprintf("sequence<%s>", elemType.Name())

		stc, err := globalTypeRegistry.GetOrCreateSequenceTypeCode(id, name, elemType, 0)
		if err != nil {
			return nil, err
		}

		return stc, nil
	case reflect.Struct:
		// For structs, check if it's a known type
		return nil, fmt.Errorf("automatic TypeCode for struct types not yet supported")
	default:
		return nil, fmt.Errorf("unsupported Go type: %s", v.Type().String())
	}
}

// validateTypeCodeMatch checks if a value matches a TypeCode
func validateTypeCodeMatch(tc TypeCode, value interface{}) bool {
	if value == nil {
		return tc.Kind() == DK_PRIMITIVE // Any primitive could be nil
	}

	v := reflect.ValueOf(value)

	// Handle special cases
	if tc.Kind() == DK_PRIMITIVE {
		if tcImpl, ok := tc.(TypeCodeImpl); ok {
			if tcImpl.TCKind() == TC_ANY {
				_, ok := value.(*Any)
				return ok
			}
		}
	}

	// For other types, do basic kind checks
	switch tc.Kind() {
	case DK_PRIMITIVE:
		// Check primitive type compatibility
		switch v.Kind() {
		case reflect.Bool:
			return tc.Name() == "boolean"
		case reflect.Int8:
			return tc.Name() == "char"
		case reflect.Int16:
			return tc.Name() == "short"
		case reflect.Int32:
			return tc.Name() == "long"
		case reflect.Int64:
			return tc.Name() == "long long"
		case reflect.Uint8:
			return tc.Name() == "octet"
		case reflect.Uint16:
			return tc.Name() == "unsigned short"
		case reflect.Uint32:
			return tc.Name() == "unsigned long"
		case reflect.Uint64:
			return tc.Name() == "unsigned long long"
		case reflect.Float32:
			return tc.Name() == "float"
		case reflect.Float64:
			return tc.Name() == "double"
		}
	case DK_STRING:
		return v.Kind() == reflect.String
	case DK_SEQUENCE:
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return false
		}

		// Check element type compatibility
		if tcImpl, ok := tc.(TypeCodeImpl); ok {
			elemTC, err := tcImpl.ContentType()
			if err != nil {
				return false
			}

			// Sample one element if available
			if v.Len() > 0 {
				return validateTypeCodeMatch(elemTC, v.Index(0).Interface())
			}
			return true // Empty slice/array, assume compatible
		}
	case DK_STRUCT:
		return v.Kind() == reflect.Struct
	case DK_ENUM:
		// Enums are usually represented as ints in Go
		return v.Kind() == reflect.Int
	case DK_UNION:
		// Unions are complex, would need specific validation
		return true // Assume valid for now
	}

	return false
}

// GetTypeCode returns a TypeCode by its ID from the global registry
func GetTypeCode(id string) (TypeCodeImpl, error) {
	return globalTypeRegistry.GetTypeCode(id)
}

// GetBasicTypeCode returns a basic TypeCode by kind from the global registry
func GetBasicTypeCode(kind TCKind) (TypeCodeImpl, error) {
	return globalTypeRegistry.GetBasicTypeCode(kind)
}

// RegisterTypeCode registers a TypeCode in the global registry
func RegisterTypeCode(tc TypeCodeImpl) {
	globalTypeRegistry.RegisterTypeCode(tc)
}

// CreateStructTypeCode creates and registers a struct TypeCode
func CreateStructTypeCode(id string, name string) (*structTypeCode, error) {
	return globalTypeRegistry.GetOrCreateStructTypeCode(id, name)
}

// CreateSequenceTypeCode creates and registers a sequence TypeCode
func CreateSequenceTypeCode(id string, name string, elementType TypeCode, bound int) (*sequenceTypeCode, error) {
	return globalTypeRegistry.GetOrCreateSequenceTypeCode(id, name, elementType, bound)
}

// CreateEnumTypeCode creates and registers an enum TypeCode
func CreateEnumTypeCode(id string, name string) (*enumTypeCode, error) {
	return globalTypeRegistry.GetOrCreateEnumTypeCode(id, name)
}

// CreateUnionTypeCode creates and registers a union TypeCode
func CreateUnionTypeCode(id string, name string, discriminatorType TypeCode) (*unionTypeCode, error) {
	return globalTypeRegistry.GetOrCreateUnionTypeCode(id, name, discriminatorType)
}

// CORBA to Go type conversion helpers

// CORBAToGo converts a CORBA value to a Go value
func CORBAToGo(value interface{}, tc TypeCode) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	// If it's an Any, extract the value and convert recursively
	if any, ok := value.(*Any); ok {
		return CORBAToGo(any.Value(), any.TypeCode())
	}

	// Handle basic types
	tcImpl, ok := tc.(TypeCodeImpl)
	if !ok {
		return nil, fmt.Errorf("TypeCode is not a TypeCodeImpl")
	}

	switch tcImpl.TCKind() {
	case TC_NULL:
		return nil, nil
	case TC_BOOLEAN:
		if b, ok := value.(bool); ok {
			return b, nil
		}
	case TC_SHORT:
		switch v := value.(type) {
		case int16:
			return v, nil
		case int:
			return int16(v), nil
		case int32:
			return int16(v), nil
		case int64:
			return int16(v), nil
		}
	case TC_LONG:
		switch v := value.(type) {
		case int32:
			return v, nil
		case int:
			return int32(v), nil
		case int64:
			return int32(v), nil
		}
	case TC_USHORT:
		switch v := value.(type) {
		case uint16:
			return v, nil
		case uint:
			return uint16(v), nil
		case uint32:
			return uint16(v), nil
		case uint64:
			return uint16(v), nil
		}
	case TC_ULONG:
		switch v := value.(type) {
		case uint32:
			return v, nil
		case uint:
			return uint32(v), nil
		case uint64:
			return uint32(v), nil
		}
	case TC_FLOAT:
		switch v := value.(type) {
		case float32:
			return v, nil
		case float64:
			return float32(v), nil
		}
	case TC_DOUBLE:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		}
	case TC_CHAR, TC_OCTET:
		switch v := value.(type) {
		case byte:
			return v, nil
		case int8:
			return byte(v), nil
		case int:
			return byte(v), nil
		}
	case TC_STRING:
		if s, ok := value.(string); ok {
			return s, nil
		}
	case TC_LONGLONG:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		}
	case TC_ULONGLONG:
		switch v := value.(type) {
		case uint64:
			return v, nil
		case uint:
			return uint64(v), nil
		}
	case TC_SEQUENCE:
		// Convert a slice
		contentType, err := tcImpl.ContentType()
		if err != nil {
			return nil, err
		}

		// Get the slice value
		vSlice, ok := value.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected slice, got %T", value)
		}

		// Create a slice of the correct type
		_, ok = contentType.(TypeCodeImpl)
		if !ok {
			return nil, fmt.Errorf("ContentType is not a TypeCodeImpl")
		}

		// For each element, convert to Go and create a new slice
		result := make([]interface{}, len(vSlice))
		for i, elem := range vSlice {
			goVal, err := CORBAToGo(elem, contentType)
			if err != nil {
				return nil, err
			}
			result[i] = goVal
		}

		return result, nil

	case TC_STRUCT:
		// For structs, we need to know the Go type to create an instance
		// This would typically come from generated code
		return nil, fmt.Errorf("struct conversion requires type information from generated code")

	case TC_ENUM:
		// For enums, convert to int
		if i, ok := value.(int); ok {
			return i, nil
		}

	case TC_UNION:
		// Unions need special handling
		return nil, fmt.Errorf("union conversion requires type information from generated code")
	}

	return nil, fmt.Errorf("conversion from CORBA to Go not supported for type %s", tc.Name())
}

// GoToCORBA converts a Go value to a CORBA value
func GoToCORBA(goValue interface{}) (interface{}, TypeCode, error) {
	if goValue == nil {
		tc, _ := TypeCodeFromKind(TC_NULL)
		return nil, tc, nil
	}

	// If it's already an Any, return as is
	if any, ok := goValue.(*Any); ok {
		return any, any.TypeCode(), nil
	}

	v := reflect.ValueOf(goValue)

	// Handle basic types
	switch v.Kind() {
	case reflect.Bool:
		tc, _ := TypeCodeFromKind(TC_BOOLEAN)
		return v.Bool(), tc, nil
	case reflect.Int8:
		tc, _ := TypeCodeFromKind(TC_CHAR)
		return byte(v.Int()), tc, nil
	case reflect.Int16:
		tc, _ := TypeCodeFromKind(TC_SHORT)
		return int16(v.Int()), tc, nil
	case reflect.Int32:
		tc, _ := TypeCodeFromKind(TC_LONG)
		return int32(v.Int()), tc, nil
	case reflect.Int64:
		tc, _ := TypeCodeFromKind(TC_LONGLONG)
		return v.Int(), tc, nil
	case reflect.Uint8:
		tc, _ := TypeCodeFromKind(TC_OCTET)
		return byte(v.Uint()), tc, nil
	case reflect.Uint16:
		tc, _ := TypeCodeFromKind(TC_USHORT)
		return uint16(v.Uint()), tc, nil
	case reflect.Uint32:
		tc, _ := TypeCodeFromKind(TC_ULONG)
		return uint32(v.Uint()), tc, nil
	case reflect.Uint64:
		tc, _ := TypeCodeFromKind(TC_ULONGLONG)
		return v.Uint(), tc, nil
	case reflect.Float32:
		tc, _ := TypeCodeFromKind(TC_FLOAT)
		return float32(v.Float()), tc, nil
	case reflect.Float64:
		tc, _ := TypeCodeFromKind(TC_DOUBLE)
		return v.Float(), tc, nil
	case reflect.String:
		tc, _ := TypeCodeFromKind(TC_STRING)
		return v.String(), tc, nil
	case reflect.Slice, reflect.Array:
		// Create a sequence
		elemTC, err := TypeCodeFromValue(reflect.New(v.Type().Elem()).Elem().Interface())
		if err != nil {
			return nil, nil, err
		}

		id := fmt.Sprintf("IDL:Sequence_%s:1.0", elemTC.Id())
		name := fmt.Sprintf("sequence<%s>", elemTC.Name())

		seqTC, err := CreateSequenceTypeCode(id, name, elemTC, 0)
		if err != nil {
			return nil, nil, err
		}

		// Convert each element
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			corbaElem, _, err := GoToCORBA(elem)
			if err != nil {
				return nil, nil, err
			}
			result[i] = corbaElem
		}

		return result, seqTC, nil

	case reflect.Struct:
		// Complex types need additional type information
		return nil, nil, fmt.Errorf("struct conversion requires generated code support")

	case reflect.Ptr:
		// Handle pointers by dereferencing
		if v.IsNil() {
			tc, _ := TypeCodeFromKind(TC_NULL)
			return nil, tc, nil
		}
		return GoToCORBA(v.Elem().Interface())

	default:
		return nil, nil, fmt.Errorf("conversion from Go to CORBA not supported for type %s", v.Type().String())
	}
}

// Get type code for any value
func GetTypeCodeForValue(value interface{}) (TypeCode, error) {
	tc, err := TypeCodeFromValue(value)
	if err != nil {
		return nil, err
	}
	return tc, nil
}
