// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Global variables for the event service
var (
	eventServiceInstance *EventServiceImpl
)

// EventServiceImpl implements the CORBA Event Service
type EventServiceImpl struct {
	orb      *ORB
	channels map[string]EventChannel
	mu       sync.RWMutex
}

// NewEventServiceImpl creates a new event service implementation
func NewEventServiceImpl(orb *ORB) *EventServiceImpl {
	return &EventServiceImpl{
		orb:      orb,
		channels: make(map[string]EventChannel),
	}
}

// CreateChannel creates a new event channel
func (es *EventServiceImpl) CreateChannel(name string, channelType EventChannelType) (EventChannel, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Check if a channel with this name already exists
	if _, exists := es.channels[name]; exists {
		return nil, ErrEventChannelExists
	}

	// Create a new channel based on the type
	var channel EventChannel
	switch channelType {
	case PushChannelType:
		channel = newPushEventChannel(name)
	case PullChannelType:
		channel = newPullEventChannel(name)
	default:
		return nil, ErrInvalidEventType
	}

	es.channels[name] = channel
	return channel, nil
}

// GetChannel gets an existing event channel by name
func (es *EventServiceImpl) GetChannel(name string) (EventChannel, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	channel, exists := es.channels[name]
	if !exists {
		return nil, ErrEventChannelNotFound
	}

	return channel, nil
}

// ListChannels lists all event channels
func (es *EventServiceImpl) ListChannels() []EventChannel {
	es.mu.RLock()
	defer es.mu.RUnlock()

	channels := make([]EventChannel, 0, len(es.channels))
	for _, channel := range es.channels {
		channels = append(channels, channel)
	}

	return channels
}

// DeleteChannel deletes an event channel
func (es *EventServiceImpl) DeleteChannel(name string) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	channel, exists := es.channels[name]
	if !exists {
		return ErrEventChannelNotFound
	}

	// Destroy the channel first
	if err := channel.Destroy(); err != nil {
		return err
	}

	delete(es.channels, name)
	return nil
}

// baseEventChannel contains common functionality for all event channels
type baseEventChannel struct {
	id          string
	name        string
	channelType EventChannelType
	consumers   map[string]EventConsumer
	suppliers   map[string]EventSupplier
	mu          sync.RWMutex
}

// ID returns the unique identifier for this channel
func (c *baseEventChannel) ID() string {
	return c.id
}

// Name returns the name of this channel
func (c *baseEventChannel) Name() string {
	return c.name
}

// Type returns the type of this channel
func (c *baseEventChannel) Type() EventChannelType {
	return c.channelType
}

// ConnectConsumer connects a consumer to this channel
func (c *baseEventChannel) ConnectConsumer(consumer EventConsumer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}

	c.consumers[consumer.ID()] = consumer
	return nil
}

// ConnectSupplier connects a supplier to this channel
func (c *baseEventChannel) ConnectSupplier(supplier EventSupplier) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if supplier == nil {
		return fmt.Errorf("supplier cannot be nil")
	}

	c.suppliers[supplier.ID()] = supplier
	return nil
}

// DisconnectConsumer disconnects a consumer from this channel
func (c *baseEventChannel) DisconnectConsumer(consumer EventConsumer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}

	delete(c.consumers, consumer.ID())
	return nil
}

// DisconnectSupplier disconnects a supplier from this channel
func (c *baseEventChannel) DisconnectSupplier(supplier EventSupplier) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if supplier == nil {
		return fmt.Errorf("supplier cannot be nil")
	}

	delete(c.suppliers, supplier.ID())
	return nil
}

// Destroy implements the basic channel destruction behavior
// Concrete implementations should override this for type-specific behavior
func (c *baseEventChannel) Destroy() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear the maps
	c.consumers = make(map[string]EventConsumer)
	c.suppliers = make(map[string]EventSupplier)

	return nil
}

// pushEventChannel implements the push model event channel
type pushEventChannel struct {
	baseEventChannel
	consumerAdmin *pushConsumerAdminImpl
	supplierAdmin *pushSupplierAdminImpl
}

// newPushEventChannel creates a new push event channel
func newPushEventChannel(name string) *pushEventChannel {
	id := uuid.New().String()
	channel := &pushEventChannel{
		baseEventChannel: baseEventChannel{
			id:          id,
			name:        name,
			channelType: PushChannelType,
			consumers:   make(map[string]EventConsumer),
			suppliers:   make(map[string]EventSupplier),
		},
	}

	// Create the admin objects with references to this channel
	channel.consumerAdmin = &pushConsumerAdminImpl{
		channel: channel,
	}
	channel.supplierAdmin = &pushSupplierAdminImpl{
		channel: channel,
	}

	return channel
}

// Destroy destroys this channel, disconnecting all consumers and suppliers
func (c *pushEventChannel) Destroy() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Disconnect all consumers
	for _, consumer := range c.consumers {
		if pc, ok := consumer.(PushConsumer); ok {
			pc.Disconnect()
		}
	}

	// Disconnect all suppliers
	for _, supplier := range c.suppliers {
		if ps, ok := supplier.(PushSupplier); ok {
			ps.Disconnect()
		}
	}

	// Clear the maps
	c.consumers = make(map[string]EventConsumer)
	c.suppliers = make(map[string]EventSupplier)

	return nil
}

// pullEventChannel implements the pull model event channel
type pullEventChannel struct {
	baseEventChannel
	consumerAdmin *pullConsumerAdminImpl
	supplierAdmin *pullSupplierAdminImpl
	eventQueue    []Event
	queueMu       sync.RWMutex
}

// newPullEventChannel creates a new pull event channel
func newPullEventChannel(name string) *pullEventChannel {
	id := uuid.New().String()
	channel := &pullEventChannel{
		baseEventChannel: baseEventChannel{
			id:          id,
			name:        name,
			channelType: PullChannelType,
			consumers:   make(map[string]EventConsumer),
			suppliers:   make(map[string]EventSupplier),
		},
		eventQueue: make([]Event, 0),
	}

	// Create the admin objects with references to this channel
	channel.consumerAdmin = &pullConsumerAdminImpl{
		channel: channel,
	}
	channel.supplierAdmin = &pullSupplierAdminImpl{
		channel: channel,
	}

	return channel
}

// Destroy destroys this channel, disconnecting all consumers and suppliers
func (c *pullEventChannel) Destroy() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Disconnect all consumers
	for _, consumer := range c.consumers {
		if pc, ok := consumer.(PullConsumer); ok {
			pc.Disconnect()
		}
	}

	// Disconnect all suppliers
	for _, supplier := range c.suppliers {
		if ps, ok := supplier.(PullSupplier); ok {
			ps.Disconnect()
		}
	}

	// Clear the maps and queue
	c.consumers = make(map[string]EventConsumer)
	c.suppliers = make(map[string]EventSupplier)

	c.queueMu.Lock()
	c.eventQueue = make([]Event, 0)
	c.queueMu.Unlock()

	return nil
}

// pushConsumerAdminImpl implements the ConsumerAdmin interface for the push model
type pushConsumerAdminImpl struct {
	channel *pushEventChannel
}

// ObtainPushSupplier creates a proxy supplier for push events
func (ca *pushConsumerAdminImpl) ObtainPushSupplier() ProxyPushSupplier {
	proxy := &proxyPushSupplierImpl{
		id:       uuid.New().String(),
		channel:  ca.channel,
		consumer: nil,
	}
	return proxy
}

// ObtainPullSupplier creates a proxy supplier for pull events (not applicable in push model)
func (ca *pushConsumerAdminImpl) ObtainPullSupplier() ProxyPullSupplier {
	// This is not applicable in push model, but implemented for interface compliance
	return nil
}

// pushSupplierAdminImpl implements the SupplierAdmin interface for the push model
type pushSupplierAdminImpl struct {
	channel *pushEventChannel
}

// ObtainPushConsumer creates a proxy consumer for push events
func (sa *pushSupplierAdminImpl) ObtainPushConsumer() ProxyPushConsumer {
	proxy := &proxyPushConsumerImpl{
		id:       uuid.New().String(),
		channel:  sa.channel,
		supplier: nil,
	}
	return proxy
}

// ObtainPullConsumer creates a proxy consumer for pull events (not applicable in push model)
func (sa *pushSupplierAdminImpl) ObtainPullConsumer() ProxyPullConsumer {
	// This is not applicable in push model, but implemented for interface compliance
	return nil
}

// pullConsumerAdminImpl implements the ConsumerAdmin interface for the pull model
type pullConsumerAdminImpl struct {
	channel *pullEventChannel
}

// ObtainPushSupplier creates a proxy supplier for push events (not applicable in pull model)
func (ca *pullConsumerAdminImpl) ObtainPushSupplier() ProxyPushSupplier {
	// This is not applicable in pull model, but implemented for interface compliance
	return nil
}

// ObtainPullSupplier creates a proxy supplier for pull events
func (ca *pullConsumerAdminImpl) ObtainPullSupplier() ProxyPullSupplier {
	proxy := &proxyPullSupplierImpl{
		id:       uuid.New().String(),
		channel:  ca.channel,
		consumer: nil,
	}
	return proxy
}

// pullSupplierAdminImpl implements the SupplierAdmin interface for the pull model
type pullSupplierAdminImpl struct {
	channel *pullEventChannel
}

// ObtainPushConsumer creates a proxy consumer for push events (not applicable in pull model)
func (sa *pullSupplierAdminImpl) ObtainPushConsumer() ProxyPushConsumer {
	// This is not applicable in pull model, but implemented for interface compliance
	return nil
}

// ObtainPullConsumer creates a proxy consumer for pull events
func (sa *pullSupplierAdminImpl) ObtainPullConsumer() ProxyPullConsumer {
	proxy := &proxyPullConsumerImpl{
		id:       uuid.New().String(),
		channel:  sa.channel,
		supplier: nil,
	}
	return proxy
}

// proxyPushConsumerImpl implements the ProxyPushConsumer interface
type proxyPushConsumerImpl struct {
	id       string
	channel  *pushEventChannel
	supplier PushSupplier
	mu       sync.RWMutex
}

// ID returns the unique identifier for this consumer
func (p *proxyPushConsumerImpl) ID() string {
	return p.id
}

// Push is called when an event is pushed to this consumer
func (p *proxyPushConsumerImpl) Push(event Event) error {
	// Forward the event to all connected suppliers in the channel
	p.channel.mu.RLock()
	defer p.channel.mu.RUnlock()

	// Push to all consumers except self
	for _, consumer := range p.channel.consumers {
		if consumer.ID() != p.id {
			if err := consumer.Push(event); err != nil {
				// Log the error but continue with other consumers
				fmt.Printf("Error pushing event to consumer %s: %v\n", consumer.ID(), err)
			}
		}
	}

	return nil
}

// Pull is not applicable for push model but implemented for interface compliance
func (p *proxyPushConsumerImpl) Pull() (Event, error) {
	return Event{}, fmt.Errorf("pull not supported in push model")
}

// TryPull is not applicable for push model but implemented for interface compliance
func (p *proxyPushConsumerImpl) TryPull() (Event, bool, error) {
	return Event{}, false, fmt.Errorf("tryPull not supported in push model")
}

// Connect connects a push supplier to this proxy
func (p *proxyPushConsumerImpl) Connect(supplier PushSupplier) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if supplier == nil {
		return fmt.Errorf("supplier cannot be nil")
	}

	p.supplier = supplier
	return nil
}

// Disconnect disconnects the supplier from this proxy
func (p *proxyPushConsumerImpl) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.supplier != nil {
		p.supplier.Disconnect()
		p.supplier = nil
	}
	return nil
}

// proxyPullConsumerImpl implements the ProxyPullConsumer interface
type proxyPullConsumerImpl struct {
	id       string
	channel  *pullEventChannel
	supplier PullSupplier
	mu       sync.RWMutex
}

// Connect connects a pull supplier to this proxy
func (p *proxyPullConsumerImpl) Connect(supplier PullSupplier) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if supplier == nil {
		return fmt.Errorf("supplier cannot be nil")
	}

	p.supplier = supplier
	return nil
}

// Disconnect disconnects the supplier from this proxy
func (p *proxyPullConsumerImpl) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.supplier != nil {
		p.supplier.Disconnect()
		p.supplier = nil
	}
	return nil
}

// proxyPushSupplierImpl implements the ProxyPushSupplier interface
type proxyPushSupplierImpl struct {
	id       string
	channel  *pushEventChannel
	consumer PushConsumer
	mu       sync.RWMutex
}

// ID returns the unique identifier for this supplier
func (p *proxyPushSupplierImpl) ID() string {
	return p.id
}

// Push pushes an event to the channel
func (p *proxyPushSupplierImpl) Push(event Event) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.consumer != nil {
		return p.consumer.Push(event)
	}

	return fmt.Errorf("no consumer connected")
}

// Pull is not applicable for push model but implemented for interface compliance
func (p *proxyPushSupplierImpl) Pull() (Event, error) {
	return Event{}, fmt.Errorf("pull not supported in push model")
}

// TryPull is not applicable for push model but implemented for interface compliance
func (p *proxyPushSupplierImpl) TryPull() (Event, bool, error) {
	return Event{}, false, fmt.Errorf("tryPull not supported in push model")
}

// ConnectPushConsumer connects a push consumer to this proxy
func (p *proxyPushSupplierImpl) ConnectPushConsumer(consumer PushConsumer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}

	p.consumer = consumer
	return nil
}

// Connect connects this supplier to a channel (implementation of EventSupplier interface)
func (p *proxyPushSupplierImpl) Connect(channel EventChannel) error {
	if channel == nil {
		return fmt.Errorf("channel cannot be nil")
	}

	return channel.ConnectSupplier(p)
}

// Disconnect disconnects this supplier from its channel
func (p *proxyPushSupplierImpl) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.DisconnectSupplier(p)
	}

	if p.consumer != nil {
		p.consumer.Disconnect()
		p.consumer = nil
	}

	return nil
}

// proxyPullSupplierImpl implements the ProxyPullSupplier interface
type proxyPullSupplierImpl struct {
	id       string
	channel  *pullEventChannel
	consumer PullConsumer
	mu       sync.RWMutex
}

// ID returns the unique identifier for this supplier
func (p *proxyPullSupplierImpl) ID() string {
	return p.id
}

// Push pushes an event to the channel (not applicable in pull model)
func (p *proxyPullSupplierImpl) Push(event Event) error {
	// In pull model, we store the event in the channel's queue
	p.channel.queueMu.Lock()
	defer p.channel.queueMu.Unlock()

	p.channel.eventQueue = append(p.channel.eventQueue, event)
	return nil
}

// Pull retrieves an event from the channel
func (p *proxyPullSupplierImpl) Pull() (Event, error) {
	p.channel.queueMu.Lock()
	defer p.channel.queueMu.Unlock()

	if len(p.channel.eventQueue) == 0 {
		// Block until an event is available or timeout
		p.channel.queueMu.Unlock()
		time.Sleep(100 * time.Millisecond) // Polling mechanism
		p.channel.queueMu.Lock()

		if len(p.channel.eventQueue) == 0 {
			return Event{}, fmt.Errorf("no events available")
		}
	}

	event := p.channel.eventQueue[0]
	p.channel.eventQueue = p.channel.eventQueue[1:]
	return event, nil
}

// TryPull attempts to retrieve an event without blocking
func (p *proxyPullSupplierImpl) TryPull() (Event, bool, error) {
	p.channel.queueMu.Lock()
	defer p.channel.queueMu.Unlock()

	if len(p.channel.eventQueue) == 0 {
		return Event{}, false, nil
	}

	event := p.channel.eventQueue[0]
	p.channel.eventQueue = p.channel.eventQueue[1:]
	return event, true, nil
}

// ConnectPullConsumer connects a pull consumer to this proxy
func (p *proxyPullSupplierImpl) ConnectPullConsumer(consumer PullConsumer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}

	p.consumer = consumer
	return nil
}

// Connect connects this supplier to a channel (implementation of EventSupplier interface)
func (p *proxyPullSupplierImpl) Connect(channel EventChannel) error {
	if channel == nil {
		return fmt.Errorf("channel cannot be nil")
	}

	return channel.ConnectSupplier(p)
}

// Disconnect disconnects this supplier from its channel
func (p *proxyPullSupplierImpl) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.DisconnectSupplier(p)
	}

	if p.consumer != nil {
		p.consumer.Disconnect()
		p.consumer = nil
	}

	return nil
}

// EventServiceServant is a CORBA servant for the Event Service
type EventServiceServant struct {
	service *EventServiceImpl
}

// NewEventServiceServant creates a new servant for the Event Service
func NewEventServiceServant(service *EventServiceImpl) *EventServiceServant {
	return &EventServiceServant{
		service: service,
	}
}

// Dispatch handles incoming CORBA method calls to the Event Service
func (ess *EventServiceServant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "CreateChannel":
		if len(args) != 2 {
			return nil, fmt.Errorf("CreateChannel requires 2 arguments")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be a string")
		}
		channelType, ok := args[1].(int)
		if !ok {
			return nil, fmt.Errorf("second argument must be an int")
		}
		return ess.service.CreateChannel(name, EventChannelType(channelType))

	case "GetChannel":
		if len(args) != 1 {
			return nil, fmt.Errorf("GetChannel requires 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		return ess.service.GetChannel(name)

	case "ListChannels":
		return ess.service.ListChannels(), nil

	case "DeleteChannel":
		if len(args) != 1 {
			return nil, fmt.Errorf("DeleteChannel requires 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		return nil, ess.service.DeleteChannel(name)

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}
