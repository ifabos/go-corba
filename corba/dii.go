// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
)

// Common DII errors
var (
	ErrInvalidArgument      = errors.New("invalid argument type")
	ErrInvalidOperation     = errors.New("invalid operation")
	ErrNoResponse           = errors.New("no response received")
	ErrOperationNotComplete = errors.New("operation not complete")
)

// NamedValue represents a named parameter in a DII request
type NamedValue struct {
	Name  string
	Value interface{}
	Flags int // For parameter direction (in, out, inout)
}

// Parameter flags
const (
	FlagIn       = 1 // Input parameter
	FlagOut      = 2 // Output parameter
	FlagInOut    = 3 // Input/Output parameter
	FlagDeferred = 4 // Deferred (asynchronous) invocation
)

// Request represents a dynamic invocation request
type Request struct {
	Target           *ObjectRef     // The target object reference
	Operation        string         // The operation name to invoke
	Parameters       []*NamedValue  // The parameters for the operation
	Result           *NamedValue    // To store the result
	Exception        error          // To store any exceptions
	Context          *Context       // Context for the request
	Status           int            // Status of the request
	ResponseReceived bool           // Whether a response has been received
	Flags            int            // Request flags
	Environment      interface{}    // Environment for the request
	ServerRequest    *ServerRequest // For DSI integration
}

// Request status
const (
	StatusInit       = 0
	StatusInProgress = 1
	StatusCompleted  = 2
	StatusError      = 3
)

// NewRequest creates a new request for the specified operation on the target
func NewRequest(target *ObjectRef, operation string) *Request {
	return &Request{
		Target:     target,
		Operation:  operation,
		Parameters: make([]*NamedValue, 0),
		Result: &NamedValue{
			Name:  "result",
			Value: nil,
			Flags: FlagOut,
		},
		Context:          NewContext(),
		Status:           StatusInit,
		ResponseReceived: false,
		Flags:            0,
	}
}

// AddParameter adds a parameter to the request
func (r *Request) AddParameter(name string, value interface{}, flag int) error {
	// Validate flag
	if flag != FlagIn && flag != FlagOut && flag != FlagInOut {
		return ErrInvalidArgument
	}

	param := &NamedValue{
		Name:  name,
		Value: value,
		Flags: flag,
	}

	r.Parameters = append(r.Parameters, param)
	return nil
}

// SetResult sets the result value of the request
func (r *Request) SetResult(value interface{}) {
	r.Result.Value = value
}

// GetResult returns the result value of the request
func (r *Request) GetResult() interface{} {
	return r.Result.Value
}

// Invoke sends the request and waits for a response
func (r *Request) Invoke() error {
	// Check if the target is valid
	if r.Target == nil || r.Target.IsNil() {
		return NewCORBASystemException("OBJECT_NOT_EXIST", 0, CompletionStatusNo)
	}

	// Set the request status to in progress
	r.Status = StatusInProgress

	// Extract parameter values for the invocation
	args := make([]interface{}, len(r.Parameters))
	for i, param := range r.Parameters {
		args[i] = param.Value
	}

	// Call the target object reference's Invoke method
	result, err := r.Target.Invoke(r.Operation, args...)
	if err != nil {
		r.Status = StatusError
		r.Exception = err
		return err
	}

	// Store the result
	r.Result.Value = result
	r.ResponseReceived = true
	r.Status = StatusCompleted
	return nil
}

// SendDeferred sends the request asynchronously
func (r *Request) SendDeferred() error {
	// Set deferred flag
	r.Flags |= FlagDeferred

	// Future implementation will use goroutines to handle this properly
	// For now, we'll do a synchronous call as a placeholder
	return r.Invoke()
}

// PollResponse checks if a deferred response has been received
func (r *Request) PollResponse() bool {
	// For the initial implementation, we'll assume the response is ready
	// if the status is completed
	return r.Status == StatusCompleted
}

// GetResponse gets the response for a deferred request
func (r *Request) GetResponse() (interface{}, error) {
	if !r.ResponseReceived {
		return nil, ErrNoResponse
	}

	if r.Status != StatusCompleted {
		return nil, ErrOperationNotComplete
	}

	return r.Result.Value, nil
}

// RequestProcessor handles DII requests
type RequestProcessor struct {
	orb *ORB
}

// NewRequestProcessor creates a new DII request processor
func NewRequestProcessor(orb *ORB) *RequestProcessor {
	return &RequestProcessor{orb: orb}
}

// CreateRequest creates a new request on the specified object reference
func (rp *RequestProcessor) CreateRequest(
	target *ObjectRef,
	operation string,
	params []*NamedValue,
	result *NamedValue,
	exceptions []string,
	ctx *Context) *Request {

	req := NewRequest(target, operation)

	// Copy parameters
	if params != nil {
		req.Parameters = params
	}

	// Set result if provided
	if result != nil {
		req.Result = result
	}

	// Set context if provided
	if ctx != nil {
		req.Context = ctx
	}

	return req
}

// ToServerRequest converts a DII Request to a DSI ServerRequest for server-side processing
func (r *Request) ToServerRequest() *ServerRequest {
	if r.ServerRequest != nil {
		return r.ServerRequest
	}

	// Create new server request
	sr := NewServerRequest(r.Operation, "", 0) // ObjectKey and RequestID would be set by actual implementation

	// Copy arguments from parameters
	for _, param := range r.Parameters {
		if param.Flags == FlagIn || param.Flags == FlagInOut {
			sr.AddArgument(param.Value)
		}
	}

	// Copy context
	sr.Context = r.Context

	// Store reference to server request
	r.ServerRequest = sr

	return sr
}

// UpdateFromServerRequest updates the request with information from a server request
func (r *Request) UpdateFromServerRequest(sr *ServerRequest) {
	// Copy result
	r.SetResult(sr.Result)

	// Copy exception
	r.Exception = sr.Exception

	// Update status
	if sr.Exception != nil {
		r.Status = StatusError
	} else {
		r.Status = StatusCompleted
	}

	r.ResponseReceived = true
}

// InvokeServerRequest processes a server request using a dynamic implementation
func InvokeServerRequest(servant DynamicImplementation, request *ServerRequest) error {
	// Pass the request to the dynamic implementation for processing
	return servant.Invoke(request)
}
