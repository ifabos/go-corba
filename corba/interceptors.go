// Package corba provides a CORBA implementation in Go
package corba

import (
	"context"
	"sync"
)

// InterceptorType defines the type of interceptor
type InterceptorType int

const (
	// RequestInterceptorType for request interceptors
	RequestInterceptorType InterceptorType = iota
	// IORInterceptorType for IOR interceptors
	IORInterceptorType
)

// InterceptorPoint defines when an interceptor is called
type InterceptorPoint int

const (
	// SendingRequest - Client is about to send request
	SendingRequest InterceptorPoint = iota
	// SendRequest - Request is being sent
	SendRequest
	// SendingReply - Server is about to send reply
	SendingReply
	// SendReply - Reply is being sent
	SendReply
	// ReceiveRequest - Server has received request
	ReceiveRequest
	// ReceiveReply - Client has received reply
	ReceiveReply
)

// RequestInfo provides information about a request to interceptors
type RequestInfo struct {
	// The operation being invoked
	Operation string
	// The object key for the target object
	ObjectKey string
	// Arguments for the operation
	Arguments []interface{}
	// Result of the operation
	Result interface{}
	// Any exception that occurred
	Exception Exception
	// The service contexts associated with the request
	ServiceContexts []ServiceContext
	// Request ID
	RequestID uint32
	// Whether the operation is a one-way operation
	ResponseExpected bool
	// For server interceptors, the servant that will handle the request
	Servant interface{}
	// Adapter that received the request
	Adapter string
}

// ServiceContext represents a service context entry
type ServiceContext struct {
	// The ID of the context
	ID uint32
	// The data associated with the context
	Data []byte
}

// ServerRequestInterceptor is invoked during server-side request processing
type ServerRequestInterceptor interface {
	// Name returns the name of the interceptor
	Name() string

	// ReceiveRequest is called before the servant operation is invoked
	ReceiveRequest(info *RequestInfo) error

	// SendReply is called after the servant operation returns
	SendReply(info *RequestInfo) error

	// SendException is called if the operation raises an exception
	SendException(info *RequestInfo, ex Exception) error
}

// ClientRequestInterceptor is invoked during client-side request processing
type ClientRequestInterceptor interface {
	// Name returns the name of the interceptor
	Name() string

	// SendRequest is called before the request is sent to the server
	SendRequest(info *RequestInfo) error

	// ReceiveReply is called after a normal reply is received
	ReceiveReply(info *RequestInfo) error

	// ReceiveException is called if an exception is received
	ReceiveException(info *RequestInfo, ex Exception) error

	// ReceiveOther is called for other outcomes (timeout, etc.)
	ReceiveOther(info *RequestInfo) error
}

// IORInterceptor is invoked during IOR creation
type IORInterceptor interface {
	// Name returns the name of the interceptor
	Name() string

	// EstablishComponents is called when an IOR is created
	EstablishComponents(info *IORInfo) error
}

// IORInfo provides information about an IOR being created
type IORInfo struct {
	// The object key for the IOR
	ObjectKey string
	// The adapter creating the IOR
	Adapter string
	// The IOR components
	Components map[string]interface{}
}

// PolicyFactory creates policy objects
type PolicyFactory interface {
	// CreatePolicy creates a policy object
	CreatePolicy(policyType uint32, value interface{}) (Policy, error)
}

// Policy represents a CORBA policy
type Policy interface {
	// PolicyType returns the type of the policy
	PolicyType() uint32
	// Copy creates a copy of the policy
	Copy() Policy
	// Destroy destroys the policy
	Destroy()
}

// InterceptorRegistry manages interceptors
type InterceptorRegistry struct {
	mu                        sync.RWMutex
	clientRequestInterceptors []ClientRequestInterceptor
	serverRequestInterceptors []ServerRequestInterceptor
	iorInterceptors           []IORInterceptor
}

// NewInterceptorRegistry creates a new interceptor registry
func NewInterceptorRegistry() *InterceptorRegistry {
	return &InterceptorRegistry{
		clientRequestInterceptors: make([]ClientRequestInterceptor, 0),
		serverRequestInterceptors: make([]ServerRequestInterceptor, 0),
		iorInterceptors:           make([]IORInterceptor, 0),
	}
}

// RegisterClientRequestInterceptor registers a client request interceptor
func (r *InterceptorRegistry) RegisterClientRequestInterceptor(interceptor ClientRequestInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clientRequestInterceptors = append(r.clientRequestInterceptors, interceptor)
}

// RegisterServerRequestInterceptor registers a server request interceptor
func (r *InterceptorRegistry) RegisterServerRequestInterceptor(interceptor ServerRequestInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.serverRequestInterceptors = append(r.serverRequestInterceptors, interceptor)
}

// RegisterIORInterceptor registers an IOR interceptor
func (r *InterceptorRegistry) RegisterIORInterceptor(interceptor IORInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.iorInterceptors = append(r.iorInterceptors, interceptor)
}

// GetClientRequestInterceptors returns all registered client request interceptors
func (r *InterceptorRegistry) GetClientRequestInterceptors() []ClientRequestInterceptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ClientRequestInterceptor, len(r.clientRequestInterceptors))
	copy(result, r.clientRequestInterceptors)
	return result
}

// GetServerRequestInterceptors returns all registered server request interceptors
func (r *InterceptorRegistry) GetServerRequestInterceptors() []ServerRequestInterceptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ServerRequestInterceptor, len(r.serverRequestInterceptors))
	copy(result, r.serverRequestInterceptors)
	return result
}

// GetIORInterceptors returns all registered IOR interceptors
func (r *InterceptorRegistry) GetIORInterceptors() []IORInterceptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]IORInterceptor, len(r.iorInterceptors))
	copy(result, r.iorInterceptors)
	return result
}

// ClearInterceptors clears all registered interceptors
func (r *InterceptorRegistry) ClearInterceptors() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clientRequestInterceptors = make([]ClientRequestInterceptor, 0)
	r.serverRequestInterceptors = make([]ServerRequestInterceptor, 0)
	r.iorInterceptors = make([]IORInterceptor, 0)
}

// Portable interceptors key for context values
type contextKey string

const interceptorCtxKey = contextKey("interceptor-context")

// InterceptorContext provides additional context for interceptors
type InterceptorContext struct {
	Data map[string]interface{}
}

// NewInterceptorContext creates a new interceptor context
func NewInterceptorContext() *InterceptorContext {
	return &InterceptorContext{
		Data: make(map[string]interface{}),
	}
}

// WithInterceptorContext adds an interceptor context to a context
func WithInterceptorContext(ctx context.Context, interceptorCtx *InterceptorContext) context.Context {
	return context.WithValue(ctx, interceptorCtxKey, interceptorCtx)
}

// GetInterceptorContext retrieves the interceptor context from a context
func GetInterceptorContext(ctx context.Context) *InterceptorContext {
	if ctx == nil {
		return nil
	}
	if ic, ok := ctx.Value(interceptorCtxKey).(*InterceptorContext); ok {
		return ic
	}
	return nil
}
