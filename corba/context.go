package corba

import (
	"sync"
)

// Context represents a CORBA context that contains a collection of properties
type Context struct {
	mu         sync.RWMutex
	properties map[string]interface{}
	parent     *Context
}

// NewContext creates a new CORBA context
func NewContext() *Context {
	return &Context{
		properties: make(map[string]interface{}),
	}
}

// SetParent sets the parent context
func (c *Context) SetParent(parent *Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parent = parent
}

// GetParent returns the parent context
func (c *Context) GetParent() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.parent
}

// Set adds or updates a property in the context
func (c *Context) Set(name string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.properties[name] = value
}

// Get retrieves a property from the context
func (c *Context) Get(name string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check in this context first
	if val, exists := c.properties[name]; exists {
		return val, true
	}

	// If not found and parent exists, check parent
	if c.parent != nil {
		return c.parent.Get(name)
	}

	return nil, false
}

// GetAll returns all properties from this context and its parent contexts
func (c *Context) GetAll() map[string]interface{} {
	result := make(map[string]interface{})

	// Get properties from parent contexts first (if any)
	if c.parent != nil {
		parentProps := c.parent.GetAll()
		for k, v := range parentProps {
			result[k] = v
		}
	}

	// Add or override with properties from this context
	c.mu.RLock()
	for k, v := range c.properties {
		result[k] = v
	}
	c.mu.RUnlock()

	return result
}

// ObjectRef represents a reference to a CORBA object
type ObjectRef struct {
	Name       string
	ServerHost string
	ServerPort int
	client     *Client
}

// Invoke calls a method on the referenced object using GIOP/IIOP
func (ref *ObjectRef) Invoke(methodName string, args ...interface{}) (interface{}, error) {
	if ref == nil || ref.client == nil {
		return nil, NewCORBASystemException("OBJECT_NOT_EXIST", 0, CompletionStatusNo)
	}

	// Use the client to invoke the method with GIOP/IIOP
	return ref.client.InvokeMethod(ref.Name, methodName, ref.ServerHost, ref.ServerPort, args...)
}

// IsNil checks if this is a nil object reference
func (ref *ObjectRef) IsNil() bool {
	return ref == nil || ref.Name == ""
}

// Equals checks if two object references point to the same object
func (ref *ObjectRef) Equals(other *ObjectRef) bool {
	if ref.IsNil() || other.IsNil() {
		return ref.IsNil() && other.IsNil()
	}

	return ref.Name == other.Name &&
		ref.ServerHost == other.ServerHost &&
		ref.ServerPort == other.ServerPort
}

// CORBASystemException represents a standard CORBA system exception
type CORBASystemException struct {
	Name             string
	Minor            uint32
	CompletionStatus CompletionStatus
}

// Error implements the error interface
func (e *CORBASystemException) Error() string {
	return e.Name
}
