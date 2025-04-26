package corba

import (
	"fmt"
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
	ior        *IOR   // Added IOR reference
	objectKey  []byte // Added object key for proper identification
	typeID     string // Added type ID (repository ID)
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

	// If both have IORs, compare them
	if ref.ior != nil && other.ior != nil {
		// Compare type IDs
		if ref.ior.TypeID != other.ior.TypeID {
			return false
		}

		// For a more thorough comparison, we would compare profiles
		// But for simplicity, we'll just compare object keys if available
		if ref.objectKey != nil && other.objectKey != nil {
			if len(ref.objectKey) != len(other.objectKey) {
				return false
			}

			for i := range ref.objectKey {
				if ref.objectKey[i] != other.objectKey[i] {
					return false
				}
			}

			return true
		}
	}

	// Fall back to comparing the basic fields
	return ref.Name == other.Name &&
		ref.ServerHost == other.ServerHost &&
		ref.ServerPort == other.ServerPort
}

// GetIOR returns the IOR associated with this reference
func (ref *ObjectRef) GetIOR() *IOR {
	return ref.ior
}

// SetIOR sets the IOR for this reference and updates related fields
func (ref *ObjectRef) SetIOR(ior *IOR) error {
	if ior == nil {
		return fmt.Errorf("cannot set nil IOR")
	}

	ref.ior = ior
	ref.typeID = ior.TypeID

	// Extract information from the primary IIOP profile
	profile, err := ior.GetPrimaryIIOPProfile()
	if err != nil {
		return err
	}

	ref.ServerHost = profile.Host
	ref.ServerPort = int(profile.Port)
	ref.objectKey = profile.ObjectKey
	ref.Name = ObjectKeyToString(profile.ObjectKey)

	return nil
}

// GetTypeID returns the repository ID (type ID) of the object
func (ref *ObjectRef) GetTypeID() string {
	if ref.ior != nil {
		return ref.ior.TypeID
	}
	return ref.typeID
}

// SetTypeID sets the repository ID (type ID) of the object
func (ref *ObjectRef) SetTypeID(typeID string) {
	ref.typeID = typeID
	if ref.ior != nil {
		ref.ior.TypeID = typeID
	}
}

// ToString returns the stringified IOR representation
func (ref *ObjectRef) ToString() (string, error) {
	if ref.ior == nil {
		// Create an IOR if none exists
		ior := NewIOR(ref.GetTypeID())
		version := IIOPVersion{Major: 1, Minor: 2} // Use IIOP 1.2

		// If we don't have an object key, generate one
		objKey := ref.objectKey
		if len(objKey) == 0 {
			objKey = ObjectKeyFromString(ref.Name)
		}

		ior.AddIIOPProfile(version, ref.ServerHost, uint16(ref.ServerPort), objKey)
		ref.ior = ior
	}

	return ref.ior.ToString(), nil
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
