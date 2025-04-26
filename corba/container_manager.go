// Package corba provides CORBA functionality for Go
package corba

import (
	"fmt"
	"sync"
)

// Constants for standard CCM container names
const (
	// Container service names
	ServiceContainerName = "ServiceContainer"
	SessionContainerName = "SessionContainer"
	ProcessContainerName = "ProcessContainer"
	EntityContainerName  = "EntityContainer"
)

// ContainerManager manages component containers in a CORBA environment
type ContainerManager struct {
	orb           *ORB
	containers    map[string]*ContainerImpl
	containerPOAs map[string]*POA
	mutex         sync.RWMutex
}

// NewContainerManager creates a new container manager
func NewContainerManager(orb *ORB) *ContainerManager {
	return &ContainerManager{
		orb:           orb,
		containers:    make(map[string]*ContainerImpl),
		containerPOAs: make(map[string]*POA),
	}
}

// CreateContainer creates a new container of the specified type
func (cm *ContainerManager) CreateContainer(name string, componentType ComponentType) (*ContainerImpl, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if container with this name already exists
	if _, exists := cm.containers[name]; exists {
		return nil, fmt.Errorf("container already exists: %s", name)
	}

	// Create a POA for the container
	poaName := fmt.Sprintf("Container_%s_POA", name)

	// Get the root POA
	rootPOA := cm.orb.GetRootPOA()

	// Create policies for the container POA
	policies := []POAPolicy{
		NewLifespanPolicy(TransientLifespan),
		NewRequestProcessingPolicy(UseActiveObjectMapOnly),
		NewIdAssignmentPolicy(SystemAssignedID),
		NewIdUniquenessPolicy(UniqueID),
		NewImplicitActivationPolicy(ImplicitActivationEnabled),
		NewServantRetentionPolicy(RetainServants),
		NewThreadPolicy(ORBControlledModel),
	}

	// Create a POA for this container
	containerPOA, err := rootPOA.CreatePOA(poaName, nil, policies)
	if err != nil {
		return nil, fmt.Errorf("failed to create POA for container: %w", err)
	}

	// Create the container with the new POA
	container, err := NewContainer(name, componentType, cm.orb, containerPOA)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Store the container and its POA
	cm.containers[name] = container
	cm.containerPOAs[name] = containerPOA

	// Activate the POA
	containerPOA.Activate()

	// Activate the container
	if err := container.Activate(); err != nil {
		return nil, fmt.Errorf("failed to activate container: %w", err)
	}

	return container, nil
}

// GetContainer retrieves a container by name
func (cm *ContainerManager) GetContainer(name string) (*ContainerImpl, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	container, exists := cm.containers[name]
	if !exists {
		return nil, fmt.Errorf("container not found: %s", name)
	}

	return container, nil
}

// FindComponentContainer finds the container that hosts a component
func (cm *ContainerManager) FindComponentContainer(componentID ComponentID) (*ContainerImpl, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for _, container := range cm.containers {
		comp, err := container.FindComponent(componentID)
		if err == nil && comp != nil {
			return container, nil
		}
	}

	return nil, fmt.Errorf("component not found in any container: %s", componentID)
}

// RemoveContainer deactivates and removes a container
func (cm *ContainerManager) RemoveContainer(name string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if container exists
	container, exists := cm.containers[name]
	if !exists {
		return fmt.Errorf("container not found: %s", name)
	}

	// Terminate the container
	if err := container.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate container: %w", err)
	}

	// Get the POA
	poa, exists := cm.containerPOAs[name]
	if exists {
		// Deactivate the POA
		if err := poa.Deactivate(true, true); err != nil {
			return fmt.Errorf("failed to deactivate container POA: %w", err)
		}
	}

	// Remove from maps
	delete(cm.containers, name)
	delete(cm.containerPOAs, name)

	return nil
}

// GetAllContainers returns all containers
func (cm *ContainerManager) GetAllContainers() []*ContainerImpl {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	containers := make([]*ContainerImpl, 0, len(cm.containers))
	for _, container := range cm.containers {
		containers = append(containers, container)
	}

	return containers
}

// GetORB returns the ORB
func (cm *ContainerManager) GetORB() *ORB {
	return cm.orb
}

// CreateStandardContainers creates the standard set of containers for CCM
func (cm *ContainerManager) CreateStandardContainers() error {
	// Create service container
	_, err := cm.CreateContainer(ServiceContainerName, ServiceComponent)
	if err != nil {
		return fmt.Errorf("failed to create service container: %w", err)
	}

	// Create session container
	_, err = cm.CreateContainer(SessionContainerName, SessionComponent)
	if err != nil {
		return fmt.Errorf("failed to create session container: %w", err)
	}

	// Create process container
	_, err = cm.CreateContainer(ProcessContainerName, ProcessComponent)
	if err != nil {
		return fmt.Errorf("failed to create process container: %w", err)
	}

	// Create entity container
	_, err = cm.CreateContainer(EntityContainerName, EntityComponent)
	if err != nil {
		return fmt.Errorf("failed to create entity container: %w", err)
	}

	return nil
}

// ComponentServerServant is a servant implementation for the component server
// that provides access to containers and CCM functionality
type ComponentServerServant struct {
	containerManager *ContainerManager
	orb              *ORB
}

// NewComponentServerServant creates a new component server servant
func NewComponentServerServant(orb *ORB, containerManager *ContainerManager) *ComponentServerServant {
	return &ComponentServerServant{
		containerManager: containerManager,
		orb:              orb,
	}
}

// GetContainerManager returns the container manager
func (s *ComponentServerServant) GetContainerManager() *ContainerManager {
	return s.containerManager
}

// GetContainer gets a container by name
func (s *ComponentServerServant) GetContainer(name string) (*ContainerImpl, error) {
	return s.containerManager.GetContainer(name)
}

// CreateContainer creates a new container
func (s *ComponentServerServant) CreateContainer(name string, componentType ComponentType) (*ContainerImpl, error) {
	return s.containerManager.CreateContainer(name, componentType)
}

// RemoveContainer removes a container
func (s *ComponentServerServant) RemoveContainer(name string) error {
	return s.containerManager.RemoveContainer(name)
}

// GetOrCreateHomeByName gets or creates a component home in the appropriate container
func (s *ComponentServerServant) GetOrCreateHomeByName(homeName string, componentType ComponentType) (ComponentHome, error) {
	// Choose container based on component type
	containerName := ""
	switch componentType {
	case ServiceComponent:
		containerName = ServiceContainerName
	case SessionComponent:
		containerName = SessionContainerName
	case ProcessComponent:
		containerName = ProcessContainerName
	case EntityComponent:
		containerName = EntityContainerName
	default:
		return nil, fmt.Errorf("invalid component type: %d", componentType)
	}

	// Get the container
	container, err := s.containerManager.GetContainer(containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Try to find the home in the container
	services := container.GetContainerServices().(*ContainerServicesImpl)
	home, err := services.FindHome(homeName)
	if err == nil {
		// Home found, return it
		return home, nil
	}

	// Create a new home
	home = NewComponentHomeServant(componentType, BasicComponent)

	// Register the home
	if err := services.RegisterHome(homeName, home); err != nil {
		return nil, fmt.Errorf("failed to register component home: %w", err)
	}

	return home, nil
}
