// Package corba provides CORBA functionality for Go
package corba

import (
	"fmt"
	"sync"
)

// The following file implements CORBA Component Model (CCM) support

// ComponentType defines the type of the component
type ComponentType int

const (
	// Component types
	ServiceComponent ComponentType = iota
	SessionComponent
	ProcessComponent
	EntityComponent
)

// ComponentCategory defines the category of the component
type ComponentCategory int

const (
	// Component categories
	BasicComponent ComponentCategory = iota
	ManagedComponent
	ExtendedComponent
)

// ComponentState represents the current state of a component
type ComponentState int

const (
	// Component lifecycle states
	COMP_INACTIVE ComponentState = iota
	COMP_ACTIVE
	COMP_REMOVED
)

// ComponentID represents a unique identifier for a component instance
type ComponentID string

// Component represents the base interface for all CORBA components
type Component interface {
	// Basic component operations
	GetComponentID() ComponentID
	GetState() ComponentState
	GetType() ComponentType
	GetCategory() ComponentCategory

	// Lifecycle operations
	Initialize() error
	Activate() error
	Passivate() error
	Remove() error
}

// ComponentServant is the base implementation for component servants
type ComponentServant struct {
	ID       ComponentID
	State    ComponentState
	Type     ComponentType
	Category ComponentCategory
	mutex    sync.RWMutex
}

// NewComponentServant creates a new component servant
func NewComponentServant(compType ComponentType, category ComponentCategory) *ComponentServant {
	return &ComponentServant{
		ID:       ComponentID(fmt.Sprintf("COMP_%d", GetNextObjectID())),
		State:    COMP_INACTIVE,
		Type:     compType,
		Category: category,
	}
}

// GetComponentID returns the ID of this component
func (c *ComponentServant) GetComponentID() ComponentID {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.ID
}

// GetState returns the current state of the component
func (c *ComponentServant) GetState() ComponentState {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.State
}

// GetType returns the component type
func (c *ComponentServant) GetType() ComponentType {
	return c.Type
}

// GetCategory returns the component category
func (c *ComponentServant) GetCategory() ComponentCategory {
	return c.Category
}

// Initialize initializes the component
func (c *ComponentServant) Initialize() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.State != COMP_INACTIVE {
		return fmt.Errorf("component is not in INACTIVE state")
	}

	// Initialization logic would go here

	return nil
}

// Activate activates the component
func (c *ComponentServant) Activate() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.State != COMP_INACTIVE {
		return fmt.Errorf("component is not in INACTIVE state")
	}

	c.State = COMP_ACTIVE
	return nil
}

// Passivate passivates the component
func (c *ComponentServant) Passivate() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.State != COMP_ACTIVE {
		return fmt.Errorf("component is not in ACTIVE state")
	}

	c.State = COMP_INACTIVE
	return nil
}

// Remove removes the component
func (c *ComponentServant) Remove() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.State = COMP_REMOVED
	return nil
}

// ComponentHome is the interface for component home objects that manage
// component instances
type ComponentHome interface {
	// Basic home operations
	GetComponentType() ComponentType
	CreateComponent() (Component, error)
	FindComponent(ComponentID) (Component, error)
	RemoveComponent(ComponentID) error
	GetAllComponents() ([]Component, error)
}

// ComponentHomeServant is the base implementation for component home servants
type ComponentHomeServant struct {
	ComponentType ComponentType
	Category      ComponentCategory
	Components    map[ComponentID]Component
	mutex         sync.RWMutex
}

// NewComponentHomeServant creates a new component home servant
func NewComponentHomeServant(compType ComponentType, category ComponentCategory) *ComponentHomeServant {
	return &ComponentHomeServant{
		ComponentType: compType,
		Category:      category,
		Components:    make(map[ComponentID]Component),
	}
}

// GetComponentType returns the type of components this home creates
func (h *ComponentHomeServant) GetComponentType() ComponentType {
	return h.ComponentType
}

// CreateComponent creates a new component instance
func (h *ComponentHomeServant) CreateComponent() (Component, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Create a new component
	comp := NewComponentServant(h.ComponentType, h.Category)

	// Initialize the component
	if err := comp.Initialize(); err != nil {
		return nil, err
	}

	// Register the component
	h.Components[comp.GetComponentID()] = comp

	return comp, nil
}

// FindComponent finds a component by ID
func (h *ComponentHomeServant) FindComponent(id ComponentID) (Component, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	comp, exists := h.Components[id]
	if !exists {
		return nil, fmt.Errorf("component not found: %s", id)
	}

	return comp, nil
}

// RemoveComponent removes a component
func (h *ComponentHomeServant) RemoveComponent(id ComponentID) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	comp, exists := h.Components[id]
	if !exists {
		return fmt.Errorf("component not found: %s", id)
	}

	// Remove the component
	if err := comp.Remove(); err != nil {
		return err
	}

	// Remove from the map
	delete(h.Components, id)

	return nil
}

// GetAllComponents returns all components managed by this home
func (h *ComponentHomeServant) GetAllComponents() ([]Component, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	components := make([]Component, 0, len(h.Components))
	for _, comp := range h.Components {
		components = append(components, comp)
	}

	return components, nil
}

// ComponentAttribute represents a component attribute
type ComponentAttribute struct {
	Name     string
	Type     string
	Value    interface{}
	ReadOnly bool
}

// ComponentEvent represents a component event
type ComponentEvent struct {
	Name      string
	Payload   interface{}
	Timestamp int64
}

// EventSource is the interface for component event sources
type EventSource interface {
	GetName() string
	Connect(EventSink) error
	Disconnect(EventSink) error
	EmitEvent(ComponentEvent) error
}

// EventSink is the interface for component event sinks
type EventSink interface {
	GetName() string
	ConsumeEvent(ComponentEvent) error
}

// ComponentContext represents the context in which a component executes
type ComponentContext interface {
	// Context operations
	GetComponent() Component
	GetContainer() Container
	GetORB() *ORB
	GetPOA() *POA

	// Attribute operations
	GetAttribute(string) (interface{}, error)
	SetAttribute(string, interface{}) error
	GetAttributes() []ComponentAttribute
}

// ComponentContextImpl is the implementation of ComponentContext
type ComponentContextImpl struct {
	Component      Component
	ContainerRef   Container
	ORBRef         *ORB
	POARef         *POA
	Attributes     map[string]ComponentAttribute
	AttributeMutex sync.RWMutex
}

// NewComponentContext creates a new component context
func NewComponentContext(comp Component, container Container, orb *ORB, poa *POA) *ComponentContextImpl {
	return &ComponentContextImpl{
		Component:    comp,
		ContainerRef: container,
		ORBRef:       orb,
		POARef:       poa,
		Attributes:   make(map[string]ComponentAttribute),
	}
}

// GetComponent returns the component associated with this context
func (ctx *ComponentContextImpl) GetComponent() Component {
	return ctx.Component
}

// GetContainer returns the container this component is running in
func (ctx *ComponentContextImpl) GetContainer() Container {
	return ctx.ContainerRef
}

// GetORB returns the ORB
func (ctx *ComponentContextImpl) GetORB() *ORB {
	return ctx.ORBRef
}

// GetPOA returns the POA
func (ctx *ComponentContextImpl) GetPOA() *POA {
	return ctx.POARef
}

// GetAttribute gets an attribute value
func (ctx *ComponentContextImpl) GetAttribute(name string) (interface{}, error) {
	ctx.AttributeMutex.RLock()
	defer ctx.AttributeMutex.RUnlock()

	attr, exists := ctx.Attributes[name]
	if !exists {
		return nil, fmt.Errorf("attribute not found: %s", name)
	}

	return attr.Value, nil
}

// SetAttribute sets an attribute value
func (ctx *ComponentContextImpl) SetAttribute(name string, value interface{}) error {
	ctx.AttributeMutex.Lock()
	defer ctx.AttributeMutex.Unlock()

	attr, exists := ctx.Attributes[name]
	if !exists {
		attr = ComponentAttribute{
			Name:     name,
			Type:     fmt.Sprintf("%T", value),
			ReadOnly: false,
		}
	} else if attr.ReadOnly {
		return fmt.Errorf("attribute is read-only: %s", name)
	}

	attr.Value = value
	ctx.Attributes[name] = attr

	return nil
}

// GetAttributes returns all attributes
func (ctx *ComponentContextImpl) GetAttributes() []ComponentAttribute {
	ctx.AttributeMutex.RLock()
	defer ctx.AttributeMutex.RUnlock()

	attributes := make([]ComponentAttribute, 0, len(ctx.Attributes))
	for _, attr := range ctx.Attributes {
		attributes = append(attributes, attr)
	}

	return attributes
}
