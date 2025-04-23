// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"fmt"
	"reflect"
)

// Common DSI errors
var (
	ErrInvalidDynamicServant   = errors.New("invalid dynamic servant")
	ErrMissingServerRequest    = errors.New("missing server request")
	ErrInvalidArgumentSkeleton = errors.New("invalid argument skeleton")
	ErrOperationNotFound       = errors.New("operation not found in the interface repository")
)

// ServerRequest represents a request being processed by a dynamic skeleton
type ServerRequest struct {
	// The name of the operation being invoked
	Operation string

	// The object key for the target object
	ObjectKey string

	// Arguments for the operation
	Arguments []interface{}

	// Result storage for the operation
	Result interface{}

	// Exception if any occurred
	Exception error

	// Request ID from the GIOP header
	RequestID uint32

	// The context for this request
	Context *Context
}

// NewServerRequest creates a new server request
func NewServerRequest(operation string, objectKey string, requestID uint32) *ServerRequest {
	return &ServerRequest{
		Operation: operation,
		ObjectKey: objectKey,
		RequestID: requestID,
		Arguments: make([]interface{}, 0),
		Context:   NewContext(),
	}
}

// AddArgument adds an argument to the server request
func (sr *ServerRequest) AddArgument(value interface{}) {
	sr.Arguments = append(sr.Arguments, value)
}

// SetResult sets the result of the operation
func (sr *ServerRequest) SetResult(result interface{}) {
	sr.Result = result
}

// SetException sets an exception for the operation
func (sr *ServerRequest) SetException(err error) {
	sr.Exception = err
}

// DynamicImplementation defines the interface that dynamic servants must implement
// to handle requests through the Dynamic Skeleton Interface (DSI)
type DynamicImplementation interface {
	// Invoke is called when a request arrives for a dynamic servant
	Invoke(request *ServerRequest) error
}

// DynamicServant is a base implementation of the DynamicImplementation interface
// that provides common functionality for dynamic servants
type DynamicServant struct {
	// Interface repository ID
	RepositoryID string

	// Operations supported by this servant
	Operations map[string]OperationInfo
}

// OperationInfo describes an operation in the dynamic skeleton
type OperationInfo struct {
	Name       string
	ReturnType reflect.Type
	Parameters []ParameterInfo
}

// ParameterInfo describes a parameter for an operation
type ParameterInfo struct {
	Name      string
	Type      reflect.Type
	Direction int // FlagIn, FlagOut, FlagInOut
}

// NewDynamicServant creates a new dynamic servant
func NewDynamicServant(repoID string) *DynamicServant {
	return &DynamicServant{
		RepositoryID: repoID,
		Operations:   make(map[string]OperationInfo),
	}
}

// AddOperation adds a new operation to the dynamic servant
func (ds *DynamicServant) AddOperation(name string, returnType reflect.Type) {
	ds.Operations[name] = OperationInfo{
		Name:       name,
		ReturnType: returnType,
		Parameters: make([]ParameterInfo, 0),
	}
}

// AddParameter adds a parameter to an operation
func (ds *DynamicServant) AddParameter(opName, paramName string, paramType reflect.Type, direction int) error {
	op, exists := ds.Operations[opName]
	if !exists {
		return fmt.Errorf("operation %s not found", opName)
	}

	op.Parameters = append(op.Parameters, ParameterInfo{
		Name:      paramName,
		Type:      paramType,
		Direction: direction,
	})

	ds.Operations[opName] = op
	return nil
}

// ValidateOperation checks if the operation exists and has valid parameters
func (ds *DynamicServant) ValidateOperation(opName string, args []interface{}) error {
	op, exists := ds.Operations[opName]
	if !exists {
		return fmt.Errorf("operation %s not defined in dynamic servant", opName)
	}

	// Count required input parameters
	inParamCount := 0
	for _, param := range op.Parameters {
		if param.Direction == FlagIn || param.Direction == FlagInOut {
			inParamCount++
		}
	}

	if len(args) != inParamCount {
		return fmt.Errorf("wrong number of arguments for operation %s: expected %d, got %d",
			opName, inParamCount, len(args))
	}

	return nil
}

// DynamicServantAdapter adapts a DynamicImplementation to the regular servant interface
type DynamicServantAdapter struct {
	Servant DynamicImplementation
}

// Dispatch implements the Dispatch method expected by the server
// It creates a ServerRequest and passes it to the dynamic implementation
func (adapter *DynamicServantAdapter) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	// Create a server request
	request := NewServerRequest(methodName, "", 0) // ObjectKey and RequestID would be filled by actual implementation

	// Add arguments
	for _, arg := range args {
		request.AddArgument(arg)
	}

	// Invoke the dynamic implementation
	err := adapter.Servant.Invoke(request)
	if err != nil {
		return nil, err
	}

	// If an exception was set, return it
	if request.Exception != nil {
		return nil, request.Exception
	}

	// Return the result
	return request.Result, nil
}

// RegisterDynamicServant registers a DynamicImplementation servant with the server
func (s *Server) RegisterDynamicServant(objectName string, servant DynamicImplementation) error {
	// Create an adapter to bridge between the DynamicImplementation and the regular servant interface
	adapter := &DynamicServantAdapter{
		Servant: servant,
	}

	// Register the adapter as a regular servant
	return s.RegisterServant(objectName, adapter)
}
