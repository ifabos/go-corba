// Package naming provides a CORBA Naming Service implementation
package corba

import (
	"errors"
	"fmt"
	"sync"
)

// Common naming service errors
var (
	ErrNameNotFound      = errors.New("name not found")
	ErrNameAlreadyBound  = errors.New("name already bound")
	ErrInvalidNameFormat = errors.New("invalid name format")
	ErrInvalidContext    = errors.New("invalid naming context")
)

// NameComponent represents a single component in a CORBA name
type NameComponent struct {
	ID   string // The identifier
	Kind string // The kind of the component
}

// String returns a string representation of the name component
func (nc NameComponent) String() string {
	if nc.Kind == "" {
		return nc.ID
	}
	return fmt.Sprintf("%s.%s", nc.ID, nc.Kind)
}

// Name is a sequence of name components forming a CORBA name path
type Name []NameComponent

// String returns a string representation of the name
func (n Name) String() string {
	if len(n) == 0 {
		return ""
	}

	result := n[0].String()
	for i := 1; i < len(n); i++ {
		result += "/" + n[i].String()
	}
	return result
}

// Binding represents a name-to-object binding in the naming service
type Binding struct {
	Name Name
	Obj  interface{}
	Type BindingType
}

// BindingType defines whether a binding is for an object or a naming context
type BindingType int

const (
	// ObjectBinding indicates a binding to a regular CORBA object
	ObjectBinding BindingType = iota
	// ContextBinding indicates a binding to a naming context
	ContextBinding
)

// NamingContext represents a context in the CORBA Naming Service
type NamingContext struct {
	mu       sync.RWMutex
	bindings map[string]*Binding
	orb      *ORB
	id       string
}

// NewNamingContext creates a new naming context
func NewNamingContext(orb *ORB, id string) *NamingContext {
	return &NamingContext{
		bindings: make(map[string]*Binding),
		orb:      orb,
		id:       id,
	}
}

// Bind associates a name with an object in this context
func (nc *NamingContext) Bind(name Name, obj interface{}) error {
	if len(name) == 0 {
		return ErrInvalidNameFormat
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If we're binding a multi-component name, find or create the intermediate contexts
	if len(name) > 1 {
		return nc.bindInSubContext(name, obj, ObjectBinding)
	}

	// Single component name, bind directly in this context
	key := name[0].String()
	if _, exists := nc.bindings[key]; exists {
		return ErrNameAlreadyBound
	}

	nc.bindings[key] = &Binding{
		Name: name,
		Obj:  obj,
		Type: ObjectBinding,
	}
	return nil
}

// Rebind binds a name to an object, overwriting any existing binding
func (nc *NamingContext) Rebind(name Name, obj interface{}) error {
	if len(name) == 0 {
		return ErrInvalidNameFormat
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If we're binding a multi-component name, find or create the intermediate contexts
	if len(name) > 1 {
		// Find or create the intermediate contexts and rebind in the final context
		firstComp := name[0].String()
		binding, exists := nc.bindings[firstComp]

		if !exists {
			// Create a new context for the first component
			subContext := NewNamingContext(nc.orb, nc.id+"/"+firstComp)
			binding = &Binding{
				Name: Name{name[0]},
				Obj:  subContext,
				Type: ContextBinding,
			}
			nc.bindings[firstComp] = binding
		} else if binding.Type != ContextBinding {
			return ErrInvalidContext
		}

		// Recursively rebind in the subcontext
		subContext, ok := binding.Obj.(*NamingContext)
		if !ok {
			return ErrInvalidContext
		}
		return subContext.Rebind(name[1:], obj)
	}

	// Single component name, rebind directly in this context
	key := name[0].String()
	nc.bindings[key] = &Binding{
		Name: name,
		Obj:  obj,
		Type: ObjectBinding,
	}
	return nil
}

// BindContext binds a name to a naming context
func (nc *NamingContext) BindContext(name Name, context *NamingContext) error {
	if len(name) == 0 {
		return ErrInvalidNameFormat
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If we're binding a multi-component name, find or create the intermediate contexts
	if len(name) > 1 {
		return nc.bindInSubContext(name, context, ContextBinding)
	}

	// Single component name, bind directly in this context
	key := name[0].String()
	if _, exists := nc.bindings[key]; exists {
		return ErrNameAlreadyBound
	}

	nc.bindings[key] = &Binding{
		Name: name,
		Obj:  context,
		Type: ContextBinding,
	}
	return nil
}

// RebindContext binds a name to a naming context, overwriting any existing binding
func (nc *NamingContext) RebindContext(name Name, context *NamingContext) error {
	if len(name) == 0 {
		return ErrInvalidNameFormat
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If we're binding a multi-component name, find or create the intermediate contexts
	if len(name) > 1 {
		// Find or create the intermediate contexts and rebind in the final context
		firstComp := name[0].String()
		binding, exists := nc.bindings[firstComp]

		if !exists {
			// Create a new context for the first component
			subContext := NewNamingContext(nc.orb, nc.id+"/"+firstComp)
			binding = &Binding{
				Name: Name{name[0]},
				Obj:  subContext,
				Type: ContextBinding,
			}
			nc.bindings[firstComp] = binding
		} else if binding.Type != ContextBinding {
			return ErrInvalidContext
		}

		// Recursively rebind in the subcontext
		subContext, ok := binding.Obj.(*NamingContext)
		if !ok {
			return ErrInvalidContext
		}
		return subContext.RebindContext(name[1:], context)
	}

	// Single component name, rebind directly in this context
	key := name[0].String()
	nc.bindings[key] = &Binding{
		Name: name,
		Obj:  context,
		Type: ContextBinding,
	}
	return nil
}

// Resolve returns the object bound to the specified name
func (nc *NamingContext) Resolve(name Name) (interface{}, error) {
	if len(name) == 0 {
		return nil, ErrInvalidNameFormat
	}

	nc.mu.RLock()
	defer nc.mu.RUnlock()

	// If we're resolving a multi-component name, navigate through the naming tree
	if len(name) > 1 {
		firstComp := name[0].String()
		binding, exists := nc.bindings[firstComp]
		if !exists {
			return nil, ErrNameNotFound
		}

		if binding.Type != ContextBinding {
			return nil, ErrInvalidContext
		}

		subContext, ok := binding.Obj.(*NamingContext)
		if !ok {
			return nil, ErrInvalidContext
		}

		// Recursively resolve in the subcontext
		return subContext.Resolve(name[1:])
	}

	// Single component name, resolve directly in this context
	key := name[0].String()
	binding, exists := nc.bindings[key]
	if !exists {
		return nil, ErrNameNotFound
	}

	return binding.Obj, nil
}

// Unbind removes the binding for the specified name
func (nc *NamingContext) Unbind(name Name) error {
	if len(name) == 0 {
		return ErrInvalidNameFormat
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If we're unbinding a multi-component name, navigate through the naming tree
	if len(name) > 1 {
		firstComp := name[0].String()
		binding, exists := nc.bindings[firstComp]
		if !exists {
			return ErrNameNotFound
		}

		if binding.Type != ContextBinding {
			return ErrInvalidContext
		}

		subContext, ok := binding.Obj.(*NamingContext)
		if !ok {
			return ErrInvalidContext
		}

		// Recursively unbind in the subcontext
		return subContext.Unbind(name[1:])
	}

	// Single component name, unbind directly in this context
	key := name[0].String()
	if _, exists := nc.bindings[key]; !exists {
		return ErrNameNotFound
	}

	delete(nc.bindings, key)
	return nil
}

// List returns a list of all bindings in this context
func (nc *NamingContext) List() []*Binding {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	result := make([]*Binding, 0, len(nc.bindings))
	for _, binding := range nc.bindings {
		result = append(result, binding)
	}

	return result
}

// bindInSubContext handles binding in a sub-context
func (nc *NamingContext) bindInSubContext(name Name, obj interface{}, bindingType BindingType) error {
	// Find or create the intermediate contexts and bind in the final context
	firstComp := name[0].String()
	binding, exists := nc.bindings[firstComp]

	if !exists {
		// Create a new context for the first component
		subContext := NewNamingContext(nc.orb, nc.id+"/"+firstComp)
		binding = &Binding{
			Name: Name{name[0]},
			Obj:  subContext,
			Type: ContextBinding,
		}
		nc.bindings[firstComp] = binding
	} else if binding.Type != ContextBinding {
		return ErrInvalidContext
	}

	// Get the subcontext
	subContext, ok := binding.Obj.(*NamingContext)
	if !ok {
		return ErrInvalidContext
	}

	// If there are more than two name components, recursively create contexts
	if len(name) > 2 {
		return subContext.bindInSubContext(name[1:], obj, bindingType)
	}

	// We're at the last parent context, bind the object to the last name component
	switch bindingType {
	case ObjectBinding:
		return subContext.Bind(Name{name[len(name)-1]}, obj)
	case ContextBinding:
		return subContext.BindContext(Name{name[len(name)-1]}, obj.(*NamingContext))
	default:
		return fmt.Errorf("invalid binding type")
	}
}
