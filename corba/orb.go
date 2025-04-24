package corba

import (
	"fmt"
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
	interceptorRegistry *InterceptorRegistry // Add interceptor registry
}

// Constants for well-known CORBA service names
const (
	NamingServiceName       = "NameService"
	InterfaceRepositoryName = "InterfaceRepository"
)

// Global variables
var (
	namingServiceInstance *NamingServiceServant
	irServiceInstance     *InterfaceRepositoryServant
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
	// This is a simplified implementation for the current architecture
	// In a full CORBA implementation, this would create a proper IOR

	// For now, we'll handle only ObjectRef objects
	if objRef, ok := obj.(*ObjectRef); ok {
		return objRef, nil
	}

	return nil, fmt.Errorf("cannot convert object to reference: unsupported type %T", obj)
}

// StringToObject converts a stringified object reference (IOR) to an ObjectRef
func (orb *ORB) StringToObject(ior string) (*ObjectRef, error) {
	// In a full implementation, this would parse an IOR string
	// For now, we'll just return an error as this isn't implemented yet
	return nil, fmt.Errorf("StringToObject not implemented")
}

// ObjectToString converts an ObjectRef to a stringified object reference (IOR)
func (orb *ORB) ObjectToString(objRef *ObjectRef) (string, error) {
	// In a full implementation, this would generate an IOR string
	// For now, we'll just return an error as this isn't implemented yet
	return "", fmt.Errorf("ObjectToString not implemented")
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
