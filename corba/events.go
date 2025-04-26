// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"fmt"
)

// Common Event Service errors
var (
	ErrEventChannelNotFound = errors.New("event channel not found")
	ErrEventDeliveryFailure = errors.New("event delivery failure")
	ErrEventChannelExists   = errors.New("event channel already exists")
)

// EventServiceName is the standard name for the Event Service
const EventServiceName = "EventService"

// Event represents a basic event in the Event Service
type Event struct {
	Type    string
	Data    interface{}
	Source  string
	Headers map[string]interface{}
}

// EventChannelType defines the type of event channel
type EventChannelType int

const (
	// PushChannelType for push model event channels
	PushChannelType EventChannelType = iota
	// PullChannelType for pull model event channels
	PullChannelType
)

// EventChannel defines the interface for an event channel
type EventChannel interface {
	// Get the ID of this channel
	ID() string

	// Get the name of this channel
	Name() string

	// Get the type of this channel
	Type() EventChannelType

	// Connect a consumer to this channel
	ConnectConsumer(consumer EventConsumer) error

	// Connect a supplier to this channel
	ConnectSupplier(supplier EventSupplier) error

	// Disconnect a consumer from this channel
	DisconnectConsumer(consumer EventConsumer) error

	// Disconnect a supplier from this channel
	DisconnectSupplier(supplier EventSupplier) error

	// Destroy this channel
	Destroy() error
}

// EventConsumer defines the interface for event consumers
type EventConsumer interface {
	// ID returns the unique identifier for this consumer
	ID() string

	// Push is called when an event is pushed to this consumer
	Push(event Event) error

	// Pull retrieves an event from the channel (for pull model)
	Pull() (Event, error)

	// TryPull attempts to retrieve an event without blocking
	TryPull() (Event, bool, error)
}

// EventSupplier defines the interface for event suppliers
type EventSupplier interface {
	// ID returns the unique identifier for this supplier
	ID() string

	// Connect connects this supplier to a channel
	Connect(channel EventChannel) error

	// Disconnect disconnects this supplier from a channel
	Disconnect() error

	// Push pushes an event to the channel (for push model)
	Push(event Event) error

	// Pull is called when a consumer pulls an event (for pull model)
	Pull() (Event, error)

	// TryPull is called when a consumer tries to pull an event without blocking
	TryPull() (Event, bool, error)
}

// EventService defines the interface for the CORBA Event Service
type EventService interface {
	// Create a new event channel
	CreateChannel(name string, channelType EventChannelType) (EventChannel, error)

	// Get an existing event channel by name
	GetChannel(name string) (EventChannel, error)

	// List all event channels
	ListChannels() []EventChannel

	// Delete an event channel
	DeleteChannel(name string) error
}

// ConsumerAdmin defines the interface for managing event consumers
type ConsumerAdmin interface {
	// Get a proxy supplier to receive events (for push model)
	ObtainPushSupplier() ProxyPushSupplier

	// Get a proxy supplier to pull events from (for pull model)
	ObtainPullSupplier() ProxyPullSupplier
}

// SupplierAdmin defines the interface for managing event suppliers
type SupplierAdmin interface {
	// Get a proxy consumer to push events to (for push model)
	ObtainPushConsumer() ProxyPushConsumer

	// Get a proxy consumer to have events pulled from (for pull model)
	ObtainPullConsumer() ProxyPullConsumer
}

// ProxyPushConsumer defines the interface for a proxy that receives pushed events
type ProxyPushConsumer interface {
	EventConsumer

	// Connect a push supplier to this proxy
	Connect(supplier PushSupplier) error

	// Disconnect the supplier from this proxy
	Disconnect() error
}

// ProxyPullConsumer defines the interface for a proxy that pulls events
type ProxyPullConsumer interface {
	// Connect a pull supplier to this proxy
	Connect(supplier PullSupplier) error

	// Disconnect the supplier from this proxy
	Disconnect() error
}

// ProxyPushSupplier defines the interface for a proxy that pushes events
type ProxyPushSupplier interface {
	EventSupplier

	// Disconnect the consumer from this proxy
	Disconnect() error
}

// ProxyPullSupplier defines the interface for a proxy that supplies events to be pulled
type ProxyPullSupplier interface {
	EventSupplier

	// Disconnect the consumer from this proxy
	Disconnect() error
}

// PushConsumer defines the interface for a consumer in the push model
type PushConsumer interface {
	// Push receives an event
	Push(event Event) error

	// Disconnect this consumer
	Disconnect() error
}

// PullConsumer defines the interface for a consumer in the pull model
type PullConsumer interface {
	// Disconnect this consumer
	Disconnect() error
}

// PushSupplier defines the interface for a supplier in the push model
type PushSupplier interface {
	// Disconnect this supplier
	Disconnect() error
}

// PullSupplier defines the interface for a supplier in the pull model
type PullSupplier interface {
	// Pull retrieves an event
	Pull() (Event, error)

	// TryPull attempts to retrieve an event without blocking
	TryPull() (Event, bool, error)

	// Disconnect this supplier
	Disconnect() error
}

// EventServiceClient is a client proxy for interacting with a remote Event Service
type EventServiceClient struct {
	objectRef *ObjectRef
}

// CreateChannel creates a new event channel
func (c *EventServiceClient) CreateChannel(name string, channelType EventChannelType) (*EventChannelClient, error) {
	result, err := c.objectRef.Invoke("CreateChannel", name, int(channelType))
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to an event channel
	if objRef, ok := result.(*ObjectRef); ok {
		return &EventChannelClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from CreateChannel")
}

// GetChannel gets an existing event channel by name
func (c *EventServiceClient) GetChannel(name string) (*EventChannelClient, error) {
	result, err := c.objectRef.Invoke("GetChannel", name)
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to an event channel
	if objRef, ok := result.(*ObjectRef); ok {
		return &EventChannelClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetChannel")
}

// ListChannels lists all event channels
func (c *EventServiceClient) ListChannels() ([]*EventChannelClient, error) {
	result, err := c.objectRef.Invoke("ListChannels")
	if err != nil {
		return nil, err
	}

	// The result should be a slice of object references
	if objRefs, ok := result.([]interface{}); ok {
		channels := make([]*EventChannelClient, 0, len(objRefs))
		for _, ref := range objRefs {
			if objRef, ok := ref.(*ObjectRef); ok {
				channels = append(channels, &EventChannelClient{objectRef: objRef})
			}
		}
		return channels, nil
	}

	return nil, fmt.Errorf("unexpected result type from ListChannels")
}

// DeleteChannel deletes an event channel
func (c *EventServiceClient) DeleteChannel(name string) error {
	_, err := c.objectRef.Invoke("DeleteChannel", name)
	return err
}

// EventChannelClient is a client proxy for interacting with a remote Event Channel
type EventChannelClient struct {
	objectRef *ObjectRef
}

// ID returns the channel's ID
func (c *EventChannelClient) ID() string {
	result, err := c.objectRef.Invoke("ID")
	if err != nil {
		return ""
	}

	if id, ok := result.(string); ok {
		return id
	}
	return ""
}

// Name returns the channel's name
func (c *EventChannelClient) Name() string {
	result, err := c.objectRef.Invoke("Name")
	if err != nil {
		return ""
	}

	if name, ok := result.(string); ok {
		return name
	}
	return ""
}

// Type returns the channel's type
func (c *EventChannelClient) Type() EventChannelType {
	result, err := c.objectRef.Invoke("Type")
	if err != nil {
		return PushChannelType // Default
	}

	if typeVal, ok := result.(int); ok {
		return EventChannelType(typeVal)
	}
	return PushChannelType
}

// ConnectConsumer connects a consumer to this channel
func (c *EventChannelClient) ConnectConsumer(consumer EventConsumer) error {
	// This would require a callback mechanism to handle remote consumers
	return fmt.Errorf("connecting remote consumers not implemented in client")
}

// ConnectSupplier connects a supplier to this channel
func (c *EventChannelClient) ConnectSupplier(supplier EventSupplier) error {
	// This would require a callback mechanism to handle remote suppliers
	return fmt.Errorf("connecting remote suppliers not implemented in client")
}

// DisconnectConsumer disconnects a consumer from this channel
func (c *EventChannelClient) DisconnectConsumer(consumer EventConsumer) error {
	// This would require a callback mechanism to handle remote consumers
	return fmt.Errorf("disconnecting remote consumers not implemented in client")
}

// DisconnectSupplier disconnects a supplier from this channel
func (c *EventChannelClient) DisconnectSupplier(supplier EventSupplier) error {
	// This would require a callback mechanism to handle remote suppliers
	return fmt.Errorf("disconnecting remote suppliers not implemented in client")
}

// Destroy destroys this channel
func (c *EventChannelClient) Destroy() error {
	_, err := c.objectRef.Invoke("Destroy")
	return err
}
