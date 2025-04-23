package idl

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/ifabos/go-corba/corba"
)

// Repository represents an Interface Repository that stores IDL type information
type Repository struct {
	mu    sync.RWMutex
	types map[string]Type
}

// NewRepository creates a new interface repository
func NewRepository() *Repository {
	return &Repository{
		types: make(map[string]Type),
	}
}

// Register adds a type to the repository
func (r *Repository) Register(id string, t Type) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[id] = t
}

// Get retrieves a type from the repository
func (r *Repository) Get(id string) (Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.types[id]
	return t, ok
}

// Helper is a base interface for all CORBA helper classes
type Helper interface {
	// ID returns the repository ID for the type
	ID() string
}

// StubFactory creates client stubs for interfaces
type StubFactory struct {
	orb  *corba.ORB
	repo *Repository
}

// NewStubFactory creates a new factory for client stubs
func NewStubFactory(orb *corba.ORB, repo *Repository) *StubFactory {
	return &StubFactory{
		orb:  orb,
		repo: repo,
	}
}

// CreateStub creates a new stub for a given interface
func (f *StubFactory) CreateStub(objRef *corba.ObjectRef, interfaceName string) (interface{}, error) {
	// In a real implementation, we would use the repository to get the interface type
	// and dynamically create a stub that implements it.
	// This is a simplified version that relies on manually implemented stub constructors.

	// For now, return an error indicating this is not fully implemented
	return nil, fmt.Errorf("stub creation for %s is not fully implemented", interfaceName)
}

// ServantBase is a base for all CORBA servants
type ServantBase struct {
	dispatcher func(methodName string, args []interface{}) (interface{}, error)
}

// SetDispatcher sets the function that will handle method dispatching
func (s *ServantBase) SetDispatcher(dispatcher func(methodName string, args []interface{}) (interface{}, error)) {
	s.dispatcher = dispatcher
}

// Dispatch dispatches a method call to the servant implementation
func (s *ServantBase) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	if s.dispatcher == nil {
		return nil, fmt.Errorf("no dispatcher set for servant")
	}
	return s.dispatcher(methodName, args)
}

// Register registers a servant with the ORB
func Register(orb *corba.ORB, objectName string, servant interface{}) error {
	// Verify that servant implements Dispatch method
	if dispatcher, ok := servant.(interface {
		Dispatch(methodName string, args []interface{}) (interface{}, error)
	}); ok {
		// Create a wrapper that delegates to the servant's Dispatch method
		wrapper := &servantWrapper{
			servant: dispatcher,
		}

		// Register with the ORB
		return orb.RegisterObject(objectName, wrapper)
	}

	return fmt.Errorf("servant does not implement Dispatch method")
}

// servantWrapper wraps a servant to handle CORBA method invocations
type servantWrapper struct {
	servant interface {
		Dispatch(methodName string, args []interface{}) (interface{}, error)
	}
}

// Invoke handles a CORBA method invocation
func (w *servantWrapper) Invoke(methodName string, args ...interface{}) (interface{}, error) {
	return w.servant.Dispatch(methodName, args)
}

// MarshalValue marshals a Go value to a CORBA value
func MarshalValue(value interface{}) (interface{}, error) {
	// In a real implementation, this would handle complex type conversions
	// For now, we just return the value as is
	return value, nil
}

// UnmarshalValue unmarshals a CORBA value to a Go value
func UnmarshalValue(corbaValue interface{}, goType reflect.Type) (interface{}, error) {
	// In a real implementation, this would handle complex type conversions
	// For now, we just try a type assertion
	if reflect.TypeOf(corbaValue).AssignableTo(goType) {
		return corbaValue, nil
	}
	return nil, fmt.Errorf("cannot convert %T to %s", corbaValue, goType)
}

// Any represents a CORBA any type that can hold values of any type
type Any struct {
	TypeCode TypeCode
	Value    interface{}
}

// TypeCode represents the metadata for a type
type TypeCode struct {
	Kind    TCKind
	ID      string
	Name    string
	Length  int
	Content *TypeCode
	Members []TypeCodeMember
}

// TypeCodeMember represents a member in a complex type
type TypeCodeMember struct {
	Name string
	Type *TypeCode
	ID   string
}

// TCKind represents the kind of a TypeCode
type TCKind int

// TypeCode kinds
const (
	TCVoid TCKind = iota
	TCShort
	TCLong
	TCLongLong
	TCUShort
	TCULong
	TCULongLong
	TCFloat
	TCDouble
	TCBoolean
	TCChar
	TCWChar
	TCOctet
	TCAny
	TCString
	TCWString
	TCSequence
	TCArray
	TCStruct
	TCUnion
	TCEnum
	TCAlias
	TCException
	TCValue
	TCValueBox
	TCNative
	TCAbstractInterface
)

// NewAny creates a new Any value
func NewAny(value interface{}) (*Any, error) {
	tc, err := TypeCodeFromValue(value)
	if err != nil {
		return nil, err
	}
	return &Any{
		TypeCode: tc,
		Value:    value,
	}, nil
}

// TypeCodeFromValue creates a TypeCode from a Go value
func TypeCodeFromValue(value interface{}) (TypeCode, error) {
	t := reflect.TypeOf(value)
	switch t.Kind() {
	case reflect.Bool:
		return TypeCode{Kind: TCBoolean, Name: "boolean"}, nil
	case reflect.Int8, reflect.Uint8:
		return TypeCode{Kind: TCOctet, Name: "octet"}, nil
	case reflect.Int16:
		return TypeCode{Kind: TCShort, Name: "short"}, nil
	case reflect.Int32:
		return TypeCode{Kind: TCLong, Name: "long"}, nil
	case reflect.Int64:
		return TypeCode{Kind: TCLongLong, Name: "long long"}, nil
	case reflect.Uint16:
		return TypeCode{Kind: TCUShort, Name: "unsigned short"}, nil
	case reflect.Uint32:
		return TypeCode{Kind: TCULong, Name: "unsigned long"}, nil
	case reflect.Uint64:
		return TypeCode{Kind: TCULongLong, Name: "unsigned long long"}, nil
	case reflect.Float32:
		return TypeCode{Kind: TCFloat, Name: "float"}, nil
	case reflect.Float64:
		return TypeCode{Kind: TCDouble, Name: "double"}, nil
	case reflect.String:
		return TypeCode{Kind: TCString, Name: "string"}, nil
	case reflect.Array, reflect.Slice:
		contentTC, err := TypeCodeFromValue(reflect.Zero(t.Elem()).Interface())
		if err != nil {
			return TypeCode{}, err
		}
		return TypeCode{
			Kind:    TCSequence,
			Name:    "sequence",
			Content: &contentTC,
			Length:  t.Len(), // For arrays, -1 for slices
		}, nil
	case reflect.Struct:
		members := make([]TypeCodeMember, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			memberTC, err := TypeCodeFromValue(reflect.Zero(field.Type).Interface())
			if err != nil {
				return TypeCode{}, err
			}
			members[i] = TypeCodeMember{
				Name: field.Name,
				Type: &memberTC,
			}
		}
		return TypeCode{
			Kind:    TCStruct,
			Name:    t.Name(),
			Members: members,
		}, nil
	default:
		return TypeCode{}, fmt.Errorf("unsupported type: %T", value)
	}
}
