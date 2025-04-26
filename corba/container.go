// Package corba provides CORBA functionality for Go
package corba

import (
	"fmt"
	"sync"
	"time"
)

// Container defines the interface for a CCM container
type Container interface {
	// Basic container operations
	GetID() string
	GetName() string
	GetComponentType() ComponentType

	// Component management
	InstallComponent(Component) error
	UninstallComponent(ComponentID) error
	FindComponent(ComponentID) (Component, error)
	ActivateComponent(ComponentID) error
	PassivateComponent(ComponentID) error
	GetAllComponents() []Component

	// Container services
	GetPOA() *POA
	GetORB() *ORB
	GetContainerServices() ContainerServices
}

// ContainerState represents the current state of a container
type ContainerState int

const (
	// Container states
	CONTAINER_INACTIVE ContainerState = iota
	CONTAINER_ACTIVE
	CONTAINER_TERMINATED
)

// ContainerImpl provides the base implementation for containers
type ContainerImpl struct {
	ID         string
	Name       string
	Type       ComponentType
	State      ContainerState
	Components map[ComponentID]Component
	POARef     *POA
	ORBRef     *ORB
	Services   ContainerServices
	mutex      sync.RWMutex
}

// NewContainer creates a new container
func NewContainer(name string, componentType ComponentType, orb *ORB, poa *POA) (*ContainerImpl, error) {
	if orb == nil {
		return nil, fmt.Errorf("ORB cannot be nil")
	}

	if poa == nil {
		var err error
		poa, err = orb.GetPOA("RootPOA")
		if err != nil {
			return nil, fmt.Errorf("failed to get RootPOA: %w", err)
		}
	}

	containerID := fmt.Sprintf("Container_%d", time.Now().UnixNano())

	container := &ContainerImpl{
		ID:         containerID,
		Name:       name,
		Type:       componentType,
		State:      CONTAINER_INACTIVE,
		Components: make(map[ComponentID]Component),
		POARef:     poa,
		ORBRef:     orb,
	}

	// Create container services
	services := NewContainerServices(container)
	container.Services = services

	return container, nil
}

// GetID returns the container ID
func (c *ContainerImpl) GetID() string {
	return c.ID
}

// GetName returns the container name
func (c *ContainerImpl) GetName() string {
	return c.Name
}

// GetComponentType returns the component type this container supports
func (c *ContainerImpl) GetComponentType() ComponentType {
	return c.Type
}

// InstallComponent installs a component in this container
func (c *ContainerImpl) InstallComponent(comp Component) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Verify component type
	if comp.GetType() != c.Type {
		return fmt.Errorf("component type mismatch: container supports %d, component is %d",
			c.Type, comp.GetType())
	}

	// Check if component is already installed
	if _, exists := c.Components[comp.GetComponentID()]; exists {
		return fmt.Errorf("component already installed: %s", comp.GetComponentID())
	}

	// Add component to container
	c.Components[comp.GetComponentID()] = comp

	return nil
}

// UninstallComponent removes a component from this container
func (c *ContainerImpl) UninstallComponent(id ComponentID) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if component exists
	comp, exists := c.Components[id]
	if !exists {
		return fmt.Errorf("component not found: %s", id)
	}

	// Check if component is active
	if comp.GetState() == COMP_ACTIVE {
		// Try to passivate it first
		if err := comp.Passivate(); err != nil {
			return fmt.Errorf("failed to passivate component: %w", err)
		}
	}

	// Remove the component
	if err := comp.Remove(); err != nil {
		return fmt.Errorf("failed to remove component: %w", err)
	}

	// Remove from map
	delete(c.Components, id)

	return nil
}

// FindComponent finds a component by ID
func (c *ContainerImpl) FindComponent(id ComponentID) (Component, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	comp, exists := c.Components[id]
	if !exists {
		return nil, fmt.Errorf("component not found: %s", id)
	}

	return comp, nil
}

// ActivateComponent activates a component
func (c *ContainerImpl) ActivateComponent(id ComponentID) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if component exists
	comp, exists := c.Components[id]
	if !exists {
		return fmt.Errorf("component not found: %s", id)
	}

	// Activate the component
	return comp.Activate()
}

// PassivateComponent passivates a component
func (c *ContainerImpl) PassivateComponent(id ComponentID) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if component exists
	comp, exists := c.Components[id]
	if !exists {
		return fmt.Errorf("component not found: %s", id)
	}

	// Passivate the component
	return comp.Passivate()
}

// GetAllComponents returns all components in this container
func (c *ContainerImpl) GetAllComponents() []Component {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	components := make([]Component, 0, len(c.Components))
	for _, comp := range c.Components {
		components = append(components, comp)
	}

	return components
}

// GetPOA returns the POA for this container
func (c *ContainerImpl) GetPOA() *POA {
	return c.POARef
}

// GetORB returns the ORB for this container
func (c *ContainerImpl) GetORB() *ORB {
	return c.ORBRef
}

// GetContainerServices returns the container services
func (c *ContainerImpl) GetContainerServices() ContainerServices {
	return c.Services
}

// Activate activates the container
func (c *ContainerImpl) Activate() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.State == CONTAINER_ACTIVE {
		return fmt.Errorf("container is already active")
	}

	// Set state to active
	c.State = CONTAINER_ACTIVE

	return nil
}

// Deactivate deactivates the container
func (c *ContainerImpl) Deactivate() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.State != CONTAINER_ACTIVE {
		return fmt.Errorf("container is not active")
	}

	// Passivate all components
	for id, comp := range c.Components {
		if comp.GetState() == COMP_ACTIVE {
			if err := comp.Passivate(); err != nil {
				return fmt.Errorf("failed to passivate component %s: %w", id, err)
			}
		}
	}

	// Set state to inactive
	c.State = CONTAINER_INACTIVE

	return nil
}

// Terminate terminates the container
func (c *ContainerImpl) Terminate() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// First deactivate if needed
	if c.State == CONTAINER_ACTIVE {
		c.mutex.Unlock()
		if err := c.Deactivate(); err != nil {
			return fmt.Errorf("failed to deactivate container: %w", err)
		}
		c.mutex.Lock()
	}

	// Remove all components
	for id, comp := range c.Components {
		if err := comp.Remove(); err != nil {
			return fmt.Errorf("failed to remove component %s: %w", id, err)
		}
	}

	// Clear component map
	c.Components = make(map[ComponentID]Component)

	// Set state to terminated
	c.State = CONTAINER_TERMINATED

	return nil
}

// ContainerServices defines the services provided by a container to components
type ContainerServices interface {
	// Component context service
	GetComponentContext(Component) (ComponentContext, error)

	// Home finding service
	FindHome(string) (ComponentHome, error)

	// Event service
	GetEventChannel(string) (EventChannel, error)
	CreateEventChannel(string) (EventChannel, error)

	// Security service
	GetCallerPrincipal() (Principal, error)
	IsCallerInRole(string) (bool, error)
}

// ContainerServicesImpl provides the base implementation for container services
type ContainerServicesImpl struct {
	container    Container
	contexts     map[ComponentID]*ComponentContextImpl
	homes        map[string]ComponentHome
	eventService *EventServiceImpl
	mutex        sync.RWMutex
}

// NewContainerServices creates a new container services implementation
func NewContainerServices(container Container) *ContainerServicesImpl {
	// Create an EventServiceImpl for this container
	eventService := NewEventServiceImpl(container.GetORB())

	return &ContainerServicesImpl{
		container:    container,
		contexts:     make(map[ComponentID]*ComponentContextImpl),
		homes:        make(map[string]ComponentHome),
		eventService: eventService,
	}
}

// GetComponentContext returns or creates a context for the given component
func (s *ContainerServicesImpl) GetComponentContext(comp Component) (ComponentContext, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := comp.GetComponentID()

	// Check if context already exists
	context, exists := s.contexts[id]
	if exists {
		return context, nil
	}

	// Create a new context
	context = NewComponentContext(comp, s.container,
		s.container.GetORB(), s.container.GetPOA())

	// Store for future use
	s.contexts[id] = context

	return context, nil
}

// FindHome finds a component home by name
func (s *ContainerServicesImpl) FindHome(name string) (ComponentHome, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	home, exists := s.homes[name]
	if !exists {
		return nil, fmt.Errorf("component home not found: %s", name)
	}

	return home, nil
}

// RegisterHome registers a component home
func (s *ContainerServicesImpl) RegisterHome(name string, home ComponentHome) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.homes[name]; exists {
		return fmt.Errorf("component home already registered: %s", name)
	}

	s.homes[name] = home
	return nil
}

// GetEventChannel gets an event channel by name
func (s *ContainerServicesImpl) GetEventChannel(name string) (EventChannel, error) {
	// Use the standard Event Service to get the channel
	return s.eventService.GetChannel(name)
}

// CreateEventChannel creates a new event channel
func (s *ContainerServicesImpl) CreateEventChannel(name string) (EventChannel, error) {
	// Use the standard Event Service to create a push channel
	return s.eventService.CreateChannel(name, PushChannelType)
}

// GetEventService returns the Event Service implementation
func (s *ContainerServicesImpl) GetEventService() *EventServiceImpl {
	return s.eventService
}

// GetCallerPrincipal returns the current caller's principal
func (s *ContainerServicesImpl) GetCallerPrincipal() (Principal, error) {
	// In a real implementation, this would use the security service
	// For now, return a default principal
	return &BasicPrincipal{
		PrincipalName:        "default_user",
		PrincipalRoles:       []string{"default_role"},
		AuthenticationMethod: AuthNone,
		Privileges:           []Privilege{},
	}, nil
}

// IsCallerInRole checks if the current caller has a specific role
func (s *ContainerServicesImpl) IsCallerInRole(role string) (bool, error) {
	principal, err := s.GetCallerPrincipal()
	if err != nil {
		return false, err
	}

	return principal.IsInRole(role), nil
}
