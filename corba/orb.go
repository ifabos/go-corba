// Package corba provides CORBA functionality for Go
package corba

import (
	"fmt"
	"strings"
	"sync"
)

// ORB represents the Object Request Broker which enables communication
// between objects in a distributed environment
type ORB struct {
	mu                  sync.RWMutex
	objectMap           map[string]interface{}
	isInitialized       bool
	serverRunning       bool
	defaultContext      *Context
	requestProcessor    *RequestProcessor
	interfaceRepository InterfaceRepository
	interceptorRegistry *InterceptorRegistry    // Add interceptor registry
	rootPOA             *POA                    // Add root POA
	poaManagers         []*POAManager           // Add POA managers
	containerManager    *ContainerManager       // Add container manager for CCM
	componentServer     *ComponentServerServant // Add component server for CCM
}

// Constants for well-known CORBA service names
const (
	NamingServiceName       = "NameService"
	InterfaceRepositoryName = "InterfaceRepository"
	RootPOAName             = "RootPOA"
	ComponentServerName     = "ComponentServer" // Add name for component server
)

// Global variables
var (
	namingServiceInstance   *NamingServiceServant
	irServiceInstance       *InterfaceRepositoryServant
	componentServerInstance *ComponentServerServant // Add global component server instance
)

// Init initializes and returns a new ORB instance
func Init() *ORB {
	orb := &ORB{
		objectMap:           make(map[string]interface{}),
		isInitialized:       true,
		defaultContext:      NewContext(),
		interceptorRegistry: NewInterceptorRegistry(), // Initialize interceptor registry
	}
	orb.requestProcessor = NewRequestProcessor(orb)

	// Initialize the Interface Repository as part of ORB initialization
	orb.interfaceRepository = NewInterfaceRepository()

	return orb
}

// Shutdown terminates the ORB
func (orb *ORB) Shutdown(wait bool) {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	if orb.serverRunning && wait {
		// Wait for pending operations to complete
	}

	orb.isInitialized = false
	orb.serverRunning = false
	orb.objectMap = make(map[string]interface{})
}

// CreateClient creates a new CORBA client
func (orb *ORB) CreateClient() *Client {
	return &Client{
		orb: orb,
	}
}

// RegisterObject registers an object with the ORB
func (orb *ORB) RegisterObject(name string, obj interface{}) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	if _, exists := orb.objectMap[name]; exists {
		return fmt.Errorf("object with name %s already registered", name)
	}

	orb.objectMap[name] = obj
	return nil
}

// ResolveObject retrieves an object from the ORB
func (orb *ORB) ResolveObject(name string) (interface{}, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	obj, exists := orb.objectMap[name]
	if !exists {
		return nil, fmt.Errorf("object with name %s not found", name)
	}

	return obj, nil
}

// IsInitialized returns whether the ORB is initialized
func (orb *ORB) IsInitialized() bool {
	orb.mu.RLock()
	defer orb.mu.RUnlock()
	return orb.isInitialized
}

// GetDefaultContext returns the default context for the ORB
func (orb *ORB) GetDefaultContext() *Context {
	return orb.defaultContext
}

// ActivateNamingService initializes and registers the naming service with this ORB
func (orb *ORB) ActivateNamingService(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the naming service is already activated
	if namingServiceInstance != nil {
		return fmt.Errorf("naming service is already active")
	}

	// Create a new naming service servant
	namingServiceInstance = NewNamingServiceServant(orb)

	// Register the naming service with the server
	if err := server.RegisterServant(NamingServiceName, namingServiceInstance); err != nil {
		return fmt.Errorf("failed to register naming service: %w", err)
	}

	return nil
}

// GetNamingService returns the naming service instance
func (orb *ORB) GetNamingService() (*NamingServiceServant, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if namingServiceInstance == nil {
		return nil, fmt.Errorf("naming service is not active")
	}

	return namingServiceInstance, nil
}

// ResolveNameService connects to a remote naming service
func (orb *ORB) ResolveNameService(host string, port int) (*NamingServiceClient, error) {
	return ConnectToNameService(orb, host, port)
}

// GetRequestProcessor returns the DII request processor
func (orb *ORB) GetRequestProcessor() *RequestProcessor {
	return orb.requestProcessor
}

// CreateRequest is a convenience method for creating a DII request
func (orb *ORB) CreateRequest(target *ObjectRef, operation string) *Request {
	return NewRequest(target, operation)
}

// ObjectToReference converts an object to an ObjectRef
func (orb *ORB) ObjectToReference(obj interface{}) (*ObjectRef, error) {
	// Handle the case where the object is already an ObjectRef
	if objRef, ok := obj.(*ObjectRef); ok {
		return objRef, nil
	}

	// Try to get repository ID from the interface repository
	var repoID string
	var err error

	if orb.interfaceRepository != nil {
		repoID, err = orb.interfaceRepository.GetRepositoryID(obj)
		if err != nil {
			// If not found, generate a default one based on the type
			repoID = FormatRepositoryID(fmt.Sprintf("%T", obj), "1.0")
		}
	} else {
		// Generate a default repository ID based on the type
		repoID = FormatRepositoryID(fmt.Sprintf("%T", obj), "1.0")
	}

	// Create a new object reference
	objRef := &ObjectRef{
		typeID: repoID,
		// Generate object key
		objectKey: GenerateObjectKey(""),
		Name:      fmt.Sprintf("object_%d", GetNextObjectID()),
	}

	// Register the object with the ORB
	err = orb.RegisterObject(objRef.Name, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to register object: %w", err)
	}

	return objRef, nil
}

// StringToObject converts a stringified object reference (IOR) to an ObjectRef
func (orb *ORB) StringToObject(iorString string) (*ObjectRef, error) {
	// Parse the IOR string
	ior, err := ParseIOR(iorString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IOR string: %w", err)
	}

	// Create a new ObjectRef
	objRef := &ObjectRef{
		ior:    ior,
		typeID: ior.TypeID,
	}

	// Extract primary profile information
	profile, err := ior.GetPrimaryIIOPProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to extract IIOP profile: %w", err)
	}

	// Set the ObjectRef fields
	objRef.ServerHost = profile.Host
	objRef.ServerPort = int(profile.Port)
	objRef.objectKey = profile.ObjectKey
	objRef.Name = ObjectKeyToString(profile.ObjectKey)

	// Set up the client
	objRef.client = orb.CreateClient()

	return objRef, nil
}

// ObjectToString converts an ObjectRef to a stringified object reference (IOR)
func (orb *ORB) ObjectToString(objRef *ObjectRef) (string, error) {
	if objRef == nil {
		return "", fmt.Errorf("cannot convert nil object reference to string")
	}

	return objRef.ToString()
}

// ActivateInterfaceRepository initializes and registers the Interface Repository with this ORB
func (orb *ORB) ActivateInterfaceRepository(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the Interface Repository is already activated
	if irServiceInstance != nil {
		return fmt.Errorf("interface repository is already active")
	}

	// Create a new Interface Repository servant
	irServiceInstance = NewInterfaceRepositoryServant(orb.interfaceRepository)

	// Register the Interface Repository with the server
	if err := server.RegisterServant(InterfaceRepositoryName, irServiceInstance); err != nil {
		return fmt.Errorf("failed to register interface repository: %w", err)
	}

	return nil
}

// GetInterfaceRepository returns the Interface Repository instance
func (orb *ORB) GetInterfaceRepository() (InterfaceRepository, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if orb.interfaceRepository == nil {
		return nil, fmt.Errorf("interface repository is not initialized")
	}

	return orb.interfaceRepository, nil
}

// ResolveInterfaceRepository connects to a remote Interface Repository
func (orb *ORB) ResolveInterfaceRepository(host string, port int) (*IRClient, error) {
	client := orb.CreateClient()

	// Connect to the server
	err := client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to interface repository: %w", err)
	}

	// Get a reference to the InterfaceRepository object
	objRef, err := client.GetObject(InterfaceRepositoryName, host, port)
	if err != nil {
		client.Disconnect(host, port)
		return nil, fmt.Errorf("failed to get interface repository reference: %w", err)
	}

	return NewIRClient(objRef), nil
}

// RegisterInterface registers an object's interface with the Interface Repository
func (orb *ORB) RegisterInterface(obj interface{}, id string, name string) error {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if orb.interfaceRepository == nil {
		return fmt.Errorf("interface repository is not initialized")
	}

	// First register the servant with the IR
	if err := orb.interfaceRepository.RegisterServant(obj, id); err != nil {
		return err
	}

	// Check if the interface already exists
	_, err := orb.interfaceRepository.LookupInterface(id)
	if err == nil {
		// Interface already exists
		return nil
	}

	// Interface doesn't exist, create it
	repo := orb.interfaceRepository.GetRepository()
	_, err = repo.CreateInterface(id, name)
	if err != nil {
		return err
	}

	return nil
}

// GetInterceptorRegistry returns the interceptor registry
func (orb *ORB) GetInterceptorRegistry() *InterceptorRegistry {
	return orb.interceptorRegistry
}

// RegisterClientRequestInterceptor registers a client request interceptor with the ORB
func (orb *ORB) RegisterClientRequestInterceptor(interceptor ClientRequestInterceptor) {
	orb.interceptorRegistry.RegisterClientRequestInterceptor(interceptor)
}

// RegisterServerRequestInterceptor registers a server request interceptor with the ORB
func (orb *ORB) RegisterServerRequestInterceptor(interceptor ServerRequestInterceptor) {
	orb.interceptorRegistry.RegisterServerRequestInterceptor(interceptor)
}

// RegisterIORInterceptor registers an IOR interceptor with the ORB
func (orb *ORB) RegisterIORInterceptor(interceptor IORInterceptor) {
	orb.interceptorRegistry.RegisterIORInterceptor(interceptor)
}

// ActivateTransactionService initializes and registers the Transaction Service with this ORB
func (orb *ORB) ActivateTransactionService(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the Transaction Service is already activated
	if transactionServiceInstance != nil {
		return fmt.Errorf("transaction service is already active")
	}

	// Create a new Transaction Service implementation
	transactionServiceInstance = NewTransactionServiceImpl(orb)

	// Create a servant for the Transaction Service
	txnServant := &TransactionServiceServant{
		service: transactionServiceInstance,
	}

	// Register the Transaction Service with the server
	if err := server.RegisterServant(TransactionServiceName, txnServant); err != nil {
		return fmt.Errorf("failed to register transaction service: %w", err)
	}

	return nil
}

// GetTransactionService returns the Transaction Service instance
func (orb *ORB) GetTransactionService() (*TransactionServiceImpl, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if transactionServiceInstance == nil {
		return nil, fmt.Errorf("transaction service is not active")
	}

	return transactionServiceInstance, nil
}

// ResolveTransactionService connects to a remote Transaction Service
func (orb *ORB) ResolveTransactionService(host string, port int) (*TransactionServiceClient, error) {
	client := orb.CreateClient()

	// Connect to the server
	err := client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to transaction service: %w", err)
	}

	// Get a reference to the Transaction Service object
	objRef, err := client.GetObject(TransactionServiceName, host, port)
	if err != nil {
		client.Disconnect(host, port)
		return nil, fmt.Errorf("failed to get transaction service reference: %w", err)
	}

	return NewTransactionServiceClient(objRef), nil
}

// ActivateEventService initializes and registers the Event Service with this ORB
func (orb *ORB) ActivateEventService(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the Event Service is already activated
	if eventServiceInstance != nil {
		return fmt.Errorf("event service is already active")
	}

	// Create a new Event Service implementation
	eventServiceInstance = NewEventServiceImpl(orb)

	// Create a servant for the Event Service
	eventServant := &EventServiceServant{
		service: eventServiceInstance,
	}

	// Register the Event Service with the server
	if err := server.RegisterServant(EventServiceName, eventServant); err != nil {
		return fmt.Errorf("failed to register event service: %w", err)
	}

	return nil
}

// GetEventService returns the Event Service instance
func (orb *ORB) GetEventService() (*EventServiceImpl, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if eventServiceInstance == nil {
		return nil, fmt.Errorf("event service is not active")
	}

	return eventServiceInstance, nil
}

// ResolveEventService connects to a remote Event Service
func (orb *ORB) ResolveEventService(host string, port int) (*EventServiceClient, error) {
	client := orb.CreateClient()

	// Connect to the server
	err := client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to event service: %w", err)
	}

	// Get a reference to the Event Service object
	objRef, err := client.GetObject(EventServiceName, host, port)
	if err != nil {
		client.Disconnect(host, port)
		return nil, fmt.Errorf("failed to get event service reference: %w", err)
	}

	return &EventServiceClient{objectRef: objRef}, nil
}

// ActivateNotificationService initializes and registers the Notification Service with this ORB
func (orb *ORB) ActivateNotificationService(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the Notification Service is already activated
	if notificationServiceInstance != nil {
		return fmt.Errorf("notification service is already active")
	}

	// Create a new Notification Service implementation
	notificationServiceInstance = NewNotificationServiceImpl(orb)

	// Create a servant for the Notification Service
	notificationServant := &NotificationServiceServant{
		service: notificationServiceInstance,
	}

	// Register the Notification Service with the server
	if err := server.RegisterServant(NotificationServiceName, notificationServant); err != nil {
		return fmt.Errorf("failed to register notification service: %w", err)
	}

	return nil
}

// GetNotificationService returns the Notification Service instance
func (orb *ORB) GetNotificationService() (*NotificationServiceImpl, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if notificationServiceInstance == nil {
		return nil, fmt.Errorf("notification service is not active")
	}

	return notificationServiceInstance, nil
}

// ResolveNotificationService connects to a remote Notification Service
func (orb *ORB) ResolveNotificationService(host string, port int) (*NotificationServiceClient, error) {
	client := orb.CreateClient()

	// Connect to the server
	err := client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service: %w", err)
	}

	// Get a reference to the Notification Service object
	objRef, err := client.GetObject(NotificationServiceName, host, port)
	if err != nil {
		client.Disconnect(host, port)
		return nil, fmt.Errorf("failed to get notification service reference: %w", err)
	}

	return &NotificationServiceClient{objectRef: objRef}, nil
}

// GetRootPOA returns the root POA, creating it if it doesn't exist
func (orb *ORB) GetRootPOA() *POA {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	if orb.rootPOA == nil {
		orb.rootPOA = orb.NewRootPOA()
	}

	return orb.rootPOA
}

// GetPOA retrieves a POA by its name path, separated by "/"
func (orb *ORB) GetPOA(poaNamePath string) (*POA, error) {
	if poaNamePath == "" || poaNamePath == RootPOAName {
		return orb.GetRootPOA(), nil
	}

	// Start with the root POA
	root := orb.GetRootPOA()

	// Split the path into segments and navigate
	segments := parseNamePath(poaNamePath)
	current := root

	for i, segment := range segments {
		// Skip the root segment if present
		if i == 0 && segment == RootPOAName {
			continue
		}

		// Find the child POA by name
		child, err := current.FindPOA(segment, true)
		if err != nil {
			return nil, fmt.Errorf("POA not found at segment '%s' of path '%s': %w",
				segment, poaNamePath, err)
		}

		current = child
	}

	return current, nil
}

// Helper function to parse POA name paths
func parseNamePath(path string) []string {
	// Implementation can be enhanced to handle escaping and other edge cases
	// For now, a simple split by "/" will do
	parts := make([]string, 0)

	// Split on "/" and filter empty segments
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}

// GetPOAManager returns the POA manager with the given index
func (orb *ORB) GetPOAManager(index int) (*POAManager, error) {
	orb.mu.RLock()
	defer orb.mu.RUnlock()

	if index < 0 || index >= len(orb.poaManagers) {
		return nil, fmt.Errorf("invalid POA manager index: %d", index)
	}

	return orb.poaManagers[index], nil
}

// CreatePOAManager creates a new POA manager
func (orb *ORB) CreatePOAManager() *POAManager {
	return orb.NewPOAManager()
}

// GetContainerManager returns the CCM container manager, creating it if it doesn't exist
func (orb *ORB) GetContainerManager() *ContainerManager {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	if orb.containerManager == nil {
		orb.containerManager = NewContainerManager(orb)
	}

	return orb.containerManager
}

// GetComponentServer returns the component server, creating it if it doesn't exist
func (orb *ORB) GetComponentServer() *ComponentServerServant {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	if orb.componentServer == nil {
		orb.componentServer = NewComponentServerServant(orb, orb.GetContainerManager())
	}

	return orb.componentServer
}

// ActivateComponentServer initializes and registers the Component Server with this ORB
func (orb *ORB) ActivateComponentServer(server *Server) error {
	orb.mu.Lock()
	defer orb.mu.Unlock()

	// Check if the component server is already activated
	if componentServerInstance != nil {
		return fmt.Errorf("component server is already active")
	}

	// Create the component server
	componentServerInstance = orb.GetComponentServer()

	// Register the component server with the server
	if err := server.RegisterServant(ComponentServerName, componentServerInstance); err != nil {
		return fmt.Errorf("failed to register component server: %w", err)
	}

	// Initialize standard CCM containers
	if err := componentServerInstance.GetContainerManager().CreateStandardContainers(); err != nil {
		return fmt.Errorf("failed to create standard containers: %w", err)
	}

	return nil
}

// CreateComponent creates a new component of the specified type using the appropriate container
func (orb *ORB) CreateComponent(componentType ComponentType, category ComponentCategory) (Component, error) {
	// Get the component server
	componentServer := orb.GetComponentServer()

	// Determine the home name based on component type
	homeName := ""
	switch componentType {
	case ServiceComponent:
		homeName = "ServiceHome"
	case SessionComponent:
		homeName = "SessionHome"
	case ProcessComponent:
		homeName = "ProcessHome"
	case EntityComponent:
		homeName = "EntityHome"
	default:
		return nil, fmt.Errorf("invalid component type: %d", componentType)
	}

	// Get or create the component home
	home, err := componentServer.GetOrCreateHomeByName(homeName, componentType)
	if err != nil {
		return nil, fmt.Errorf("failed to get component home: %w", err)
	}

	// Create the component
	component, err := home.CreateComponent()
	if err != nil {
		return nil, fmt.Errorf("failed to create component: %w", err)
	}

	return component, nil
}

// FindComponent finds a component by ID
func (orb *ORB) FindComponent(id ComponentID) (Component, error) {
	// Get the container manager
	containerManager := orb.GetContainerManager()

	// Find the container that hosts the component
	container, err := containerManager.FindComponentContainer(id)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	// Get the component from the container
	component, err := container.FindComponent(id)
	if err != nil {
		return nil, fmt.Errorf("component not found in container: %w", err)
	}

	return component, nil
}

// CreateComponentReference creates an object reference for a component
func (orb *ORB) CreateComponentReference(component Component) (*ObjectRef, error) {
	// Get the container manager
	containerManager := orb.GetContainerManager()

	// Find the container that hosts the component
	container, err := containerManager.FindComponentContainer(component.GetComponentID())
	if err != nil {
		return nil, fmt.Errorf("component not found in any container: %w", err)
	}

	// Get the POA for the container
	poa := container.GetPOA()

	// Create a reference type ID based on the component type
	typeIDBase := ""
	switch component.GetType() {
	case ServiceComponent:
		typeIDBase = "IDL:CORBA/ServiceComponent"
	case SessionComponent:
		typeIDBase = "IDL:CORBA/SessionComponent"
	case ProcessComponent:
		typeIDBase = "IDL:CORBA/ProcessComponent"
	case EntityComponent:
		typeIDBase = "IDL:CORBA/EntityComponent"
	default:
		typeIDBase = "IDL:CORBA/Component"
	}

	// Format the repository ID with version
	repositoryID := fmt.Sprintf("%s:1.0", typeIDBase)

	// Create an object ID for the component
	objectID := []byte(string(component.GetComponentID()))

	// Create the reference
	ref := poa.CreateReferenceWithId(objectID, repositoryID)

	return ref, nil
}
