// Package naming provides a CORBA Naming Service implementation
package corba

import (
	"fmt"
)

// NamingServiceClient provides a client interface to the CORBA Naming Service
type NamingServiceClient struct {
	client     *Client
	objectRef  *ObjectRef
	serverHost string
	serverPort int
}

// ConnectToNameService connects to a naming service running on the specified host and port
func ConnectToNameService(orb *ORB, host string, port int) (*NamingServiceClient, error) {
	client := orb.CreateClient()

	// Connect to the NameService
	err := client.Connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to naming service: %w", err)
	}

	// Get a reference to the NameService object
	objRef, err := client.GetObject("NameService", host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to get naming service reference: %w", err)
	}

	return &NamingServiceClient{
		client:     client,
		objectRef:  objRef,
		serverHost: host,
		serverPort: port,
	}, nil
}

// Bind associates a name with an object in the naming service
func (nsc *NamingServiceClient) Bind(name string, obj interface{}) error {
	_, err := nsc.objectRef.Invoke("bind", name, obj)
	return err
}

// Rebind binds a name to an object, overwriting any existing binding
func (nsc *NamingServiceClient) Rebind(name string, obj interface{}) error {
	_, err := nsc.objectRef.Invoke("rebind", name, obj)
	return err
}

// Resolve returns the object bound to the specified name
func (nsc *NamingServiceClient) Resolve(name string) (interface{}, error) {
	return nsc.objectRef.Invoke("resolve", name)
}

// Unbind removes the binding for the specified name
func (nsc *NamingServiceClient) Unbind(name string) error {
	_, err := nsc.objectRef.Invoke("unbind", name)
	return err
}

// NewContext creates a new naming context that is not bound to the naming tree
func (nsc *NamingServiceClient) NewContext() (*ObjectRef, error) {
	result, err := nsc.objectRef.Invoke("new_context")
	if err != nil {
		return nil, err
	}

	// The result should be a reference to the new naming context
	contextRef, ok := result.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid context reference returned")
	}

	return contextRef, nil
}

// BindNewContext creates a new context and binds it to the specified name
func (nsc *NamingServiceClient) BindNewContext(name string) (*ObjectRef, error) {
	result, err := nsc.objectRef.Invoke("bind_new_context", name)
	if err != nil {
		return nil, err
	}

	// The result should be a reference to the new naming context
	contextRef, ok := result.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid context reference returned")
	}

	return contextRef, nil
}

// List returns a list of all bindings in the root context
func (nsc *NamingServiceClient) List() ([]*Binding, error) {
	result, err := nsc.objectRef.Invoke("list")
	if err != nil {
		return nil, err
	}

	// The result should be a list of bindings
	bindings, ok := result.([]*Binding)
	if !ok {
		return nil, fmt.Errorf("invalid binding list returned")
	}

	return bindings, nil
}

// Close closes the connection to the naming service
func (nsc *NamingServiceClient) Close() error {
	return nsc.client.Disconnect(nsc.serverHost, nsc.serverPort)
}
