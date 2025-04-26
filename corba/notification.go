// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"fmt"
)

// Common Notification Service errors
var (
	ErrInvalidFilter        = errors.New("invalid filter")
	ErrInvalidEventType     = errors.New("invalid event type")
	ErrQoSNotSupported      = errors.New("quality of service not supported")
	ErrFilterNotSupported   = errors.New("filter not supported")
	ErrInvalidAdminProperty = errors.New("invalid admin property")
	ErrInvalidChannelName   = errors.New("invalid channel name")
	ErrFilterAlreadyExists  = errors.New("filter already exists")
	ErrFilterNotFound       = errors.New("filter not found")
	ErrConstraintNotFound   = errors.New("constraint not found")
)

// NotificationServiceName is the standard name for the Notification Service
const NotificationServiceName = "NotificationService"

// QoSPropertyType defines the Quality of Service property types
type QoSPropertyType string

// QoS property constants
const (
	QoS_EventReliability      QoSPropertyType = "EventReliability"
	QoS_ConnectionReliability QoSPropertyType = "ConnectionReliability"
	QoS_Priority              QoSPropertyType = "Priority"
	QoS_Timeout               QoSPropertyType = "Timeout"
	QoS_StartTimeSupported    QoSPropertyType = "StartTimeSupported"
	QoS_StopTimeSupported     QoSPropertyType = "StopTimeSupported"
	QoS_MaxEventsPerConsumer  QoSPropertyType = "MaxEventsPerConsumer"
	QoS_OrderPolicy           QoSPropertyType = "OrderPolicy"
	QoS_DiscardPolicy         QoSPropertyType = "DiscardPolicy"
	QoS_MaximumBatchSize      QoSPropertyType = "MaximumBatchSize"
	QoS_PacingInterval        QoSPropertyType = "PacingInterval"
)

// Reliability levels
const (
	Reliability_BestEffort     = 0
	Reliability_Persistent     = 1
	Reliability_BestEffortPers = 2 // Best effort with persistent storage as backup
)

// Order policy values
const (
	OrderPolicy_AnyOrder      = 0
	OrderPolicy_FifoOrder     = 1
	OrderPolicy_PriorityOrder = 2
	OrderPolicy_DeadlineOrder = 3
)

// Discard policy values
const (
	DiscardPolicy_AnyOrder      = 0
	DiscardPolicy_FifoOrder     = 1
	DiscardPolicy_PriorityOrder = 2
	DiscardPolicy_DeadlineOrder = 3
	DiscardPolicy_LifoOrder     = 4
)

// StructuredEvent represents a structured event in the Notification Service
type StructuredEvent struct {
	Header     EventHeader
	FilterData map[string]interface{}
	Payload    interface{}
}

// EventHeader contains header information for structured events
type EventHeader struct {
	FixedHeader    FixedEventHeader
	VariableHeader map[string]interface{}
}

// FixedEventHeader contains the fixed part of an event header
type FixedEventHeader struct {
	EventType  EventType
	EventName  string
	DomainName string
}

// EventType represents a structured event type
type EventType struct {
	Domain string
	Type   string
}

// EventBatch is a batch of structured events
type EventBatch []StructuredEvent

// AdminPropertiesType defines properties for notification channel administration
type AdminPropertiesType struct {
	MaxQueueLength  int32
	MaxConsumers    int32
	MaxSuppliers    int32
	RejectNewEvents bool
}

// FilterableEventType defines event filtering options
type FilterableEventType struct {
	EventType           EventType
	FilterableDataNames []string
}

// FilterConstraintType defines a filter constraint
type FilterConstraintType struct {
	ConstraintID string
	Expression   string
}

// Filter defines the interface for notification event filters
type Filter interface {
	// Get the constraint expressions in this filter
	GetConstraints() []FilterConstraintType

	// Add a constraint to this filter
	AddConstraint(constraintID string, expression string) error

	// Remove a constraint from this filter
	RemoveConstraint(constraintID string) error

	// Modify a constraint in this filter
	ModifyConstraint(constraintID string, expression string) error

	// Get a constraint from this filter
	GetConstraint(constraintID string) (FilterConstraintType, error)

	// Match evaluates whether an event matches this filter
	Match(event StructuredEvent) bool
}

// FilterFactory creates filters for notification consumers and suppliers
type FilterFactory interface {
	// Create a new filter
	CreateFilter(constraintLanguage string) (Filter, error)
}

// QoS defines Quality of Service settings
type QoS struct {
	Properties map[QoSPropertyType]interface{}
}

// StructuredPushConsumer extends PushConsumer for structured events
type StructuredPushConsumer interface {
	PushConsumer

	// Push a structured event to this consumer
	PushStructuredEvent(event StructuredEvent) error

	// Push a batch of structured events to this consumer
	PushEventBatch(events EventBatch) error
}

// StructuredPullConsumer extends PullConsumer for structured events
type StructuredPullConsumer interface {
	PullConsumer

	// Connect this consumer to the notification channel
	ConnectStructuredPullSupplier(supplier StructuredPullSupplier) error
}

// StructuredPushSupplier extends PushSupplier for structured events
type StructuredPushSupplier interface {
	PushSupplier

	// Connect this supplier to the notification channel
	ConnectStructuredPushConsumer(consumer StructuredPushConsumer) error
}

// StructuredPullSupplier extends PullSupplier for structured events
type StructuredPullSupplier interface {
	PullSupplier

	// Pull a structured event
	PullStructuredEvent() (StructuredEvent, error)

	// Try to pull a structured event
	TryPullStructuredEvent() (StructuredEvent, bool, error)
}

// NotificationConsumerAdmin extends ConsumerAdmin with notification specific features
type NotificationConsumerAdmin interface {
	ConsumerAdmin

	// Get the QoS properties for this admin
	GetQoS() QoS

	// Set the QoS properties for this admin
	SetQoS(qos QoS) error

	// Get the filter factory
	GetFilterFactory() FilterFactory

	// Get the list of all filters
	GetAllFilters() []Filter

	// Get a filter by ID
	GetFilter(filterID string) (Filter, error)

	// Add a filter
	AddFilter(filter Filter) string

	// Remove a filter
	RemoveFilter(filterID string) error

	// Obtain a proxy for structured push supplier
	ObtainStructuredPushSupplier() (StructuredPushSupplier, error)

	// Obtain a proxy for structured pull supplier
	ObtainStructuredPullSupplier() (StructuredPullSupplier, error)
}

// NotificationSupplierAdmin extends SupplierAdmin with notification specific features
type NotificationSupplierAdmin interface {
	SupplierAdmin

	// Get the QoS properties for this admin
	GetQoS() QoS

	// Set the QoS properties for this admin
	SetQoS(qos QoS) error

	// Get the filter factory
	GetFilterFactory() FilterFactory

	// Get the list of all filters
	GetAllFilters() []Filter

	// Get a filter by ID
	GetFilter(filterID string) (Filter, error)

	// Add a filter
	AddFilter(filter Filter) string

	// Remove a filter
	RemoveFilter(filterID string) error

	// Obtain a proxy for structured push consumer
	ObtainStructuredPushConsumer() (StructuredPushConsumer, error)

	// Obtain a proxy for structured pull consumer
	ObtainStructuredPullConsumer() (StructuredPullConsumer, error)
}

// EventChannelFactory creates notification event channels
type EventChannelFactory interface {
	// Create a new notification channel
	CreateChannel(name string, initialQoS QoS, initialAdmin AdminPropertiesType) (NotificationChannel, error)

	// Get an existing notification channel
	GetChannel(name string) (NotificationChannel, error)

	// List all notification channels
	ListChannels() []NotificationChannel

	// Delete a notification channel
	DeleteChannel(name string) error
}

// NotificationChannel extends EventChannel with notification specific features
type NotificationChannel interface {
	EventChannel

	// Get the QoS properties for this channel
	GetQoS() QoS

	// Set the QoS properties for this channel
	SetQoS(qos QoS) error

	// Get the admin properties for this channel
	GetAdminProperties() AdminPropertiesType

	// Set the admin properties for this channel
	SetAdminProperties(props AdminPropertiesType) error

	// Get the default consumer admin
	DefaultConsumerAdmin() NotificationConsumerAdmin

	// Get the default supplier admin
	DefaultSupplierAdmin() NotificationSupplierAdmin

	// Create a new consumer admin
	NewConsumerAdmin() (NotificationConsumerAdmin, error)

	// Create a new supplier admin
	NewSupplierAdmin() (NotificationSupplierAdmin, error)

	// Get the filter factory
	GetFilterFactory() FilterFactory
}

// NotificationService provides access to the notification service
type NotificationService interface {
	EventService

	// Get the event channel factory
	GetEventChannelFactory() EventChannelFactory

	// Get the default filter factory
	GetDefaultFilterFactory() FilterFactory

	// Create a structured event
	CreateStructuredEvent(domain, type_, name string, filterData map[string]interface{}, payload interface{}) StructuredEvent
}

// NotificationServiceClient is a client proxy for interacting with a remote Notification Service
type NotificationServiceClient struct {
	objectRef          *ObjectRef
	eventServiceClient *EventServiceClient
}

// CreateChannel creates a new event channel (inherits from Event Service)
func (c *NotificationServiceClient) CreateChannel(name string, channelType EventChannelType) (*EventChannelClient, error) {
	// Initialize the event service client if needed
	if c.eventServiceClient == nil {
		c.eventServiceClient = &EventServiceClient{objectRef: c.objectRef}
	}
	return c.eventServiceClient.CreateChannel(name, channelType)
}

// GetChannel gets an existing event channel by name (inherits from Event Service)
func (c *NotificationServiceClient) GetChannel(name string) (*EventChannelClient, error) {
	// Initialize the event service client if needed
	if c.eventServiceClient == nil {
		c.eventServiceClient = &EventServiceClient{objectRef: c.objectRef}
	}
	return c.eventServiceClient.GetChannel(name)
}

// ListChannels lists all event channels (inherits from Event Service)
func (c *NotificationServiceClient) ListChannels() ([]*EventChannelClient, error) {
	// Initialize the event service client if needed
	if c.eventServiceClient == nil {
		c.eventServiceClient = &EventServiceClient{objectRef: c.objectRef}
	}
	return c.eventServiceClient.ListChannels()
}

// DeleteChannel deletes an event channel (inherits from Event Service)
func (c *NotificationServiceClient) DeleteChannel(name string) error {
	// Initialize the event service client if needed
	if c.eventServiceClient == nil {
		c.eventServiceClient = &EventServiceClient{objectRef: c.objectRef}
	}
	return c.eventServiceClient.DeleteChannel(name)
}

// GetEventChannelFactory returns the event channel factory
func (c *NotificationServiceClient) GetEventChannelFactory() (*EventChannelFactoryClient, error) {
	result, err := c.objectRef.Invoke("GetEventChannelFactory")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to an event channel factory
	if objRef, ok := result.(*ObjectRef); ok {
		return &EventChannelFactoryClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetEventChannelFactory")
}

// GetDefaultFilterFactory returns the default filter factory
func (c *NotificationServiceClient) GetDefaultFilterFactory() (*FilterFactoryClient, error) {
	result, err := c.objectRef.Invoke("GetDefaultFilterFactory")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a filter factory
	if objRef, ok := result.(*ObjectRef); ok {
		return &FilterFactoryClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetDefaultFilterFactory")
}

// CreateStructuredEvent creates a structured event
func (c *NotificationServiceClient) CreateStructuredEvent(domain, type_, name string,
	filterData map[string]interface{}, payload interface{}) (StructuredEvent, error) {

	result, err := c.objectRef.Invoke("CreateStructuredEvent", domain, type_, name, filterData, payload)
	if err != nil {
		return StructuredEvent{}, err
	}

	// The result should be a structured event
	if event, ok := result.(StructuredEvent); ok {
		return event, nil
	}

	// Try to convert from map to struct if needed
	if eventMap, ok := result.(map[string]interface{}); ok {
		event := StructuredEvent{}

		// Try to extract header
		if headerMap, ok := eventMap["Header"].(map[string]interface{}); ok {
			// Extract fixed header
			if fixedHeaderMap, ok := headerMap["FixedHeader"].(map[string]interface{}); ok {
				// Extract event type
				if eventTypeMap, ok := fixedHeaderMap["EventType"].(map[string]interface{}); ok {
					event.Header.FixedHeader.EventType.Domain = fmt.Sprintf("%v", eventTypeMap["Domain"])
					event.Header.FixedHeader.EventType.Type = fmt.Sprintf("%v", eventTypeMap["Type"])
				}

				event.Header.FixedHeader.EventName = fmt.Sprintf("%v", fixedHeaderMap["EventName"])
				event.Header.FixedHeader.DomainName = fmt.Sprintf("%v", fixedHeaderMap["DomainName"])
			}

			// Extract variable header
			if varHeader, ok := headerMap["VariableHeader"].(map[string]interface{}); ok {
				event.Header.VariableHeader = varHeader
			}
		}

		// Extract filter data
		if filterMap, ok := eventMap["FilterData"].(map[string]interface{}); ok {
			event.FilterData = filterMap
		}

		// Extract payload
		event.Payload = eventMap["Payload"]

		return event, nil
	}

	return StructuredEvent{}, fmt.Errorf("unexpected result type from CreateStructuredEvent")
}

// EventChannelFactoryClient is a client proxy for interacting with a remote Event Channel Factory
type EventChannelFactoryClient struct {
	objectRef *ObjectRef
}

// CreateChannel creates a new notification channel
func (c *EventChannelFactoryClient) CreateChannel(name string, initialQoS QoS, initialAdmin AdminPropertiesType) (*NotificationChannelClient, error) {
	result, err := c.objectRef.Invoke("CreateChannel", name, initialQoS, initialAdmin)
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a notification channel
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationChannelClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from CreateChannel")
}

// GetChannel gets an existing notification channel
func (c *EventChannelFactoryClient) GetChannel(name string) (*NotificationChannelClient, error) {
	result, err := c.objectRef.Invoke("GetChannel", name)
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a notification channel
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationChannelClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetChannel")
}

// ListChannels lists all notification channels
func (c *EventChannelFactoryClient) ListChannels() ([]*NotificationChannelClient, error) {
	result, err := c.objectRef.Invoke("ListChannels")
	if err != nil {
		return nil, err
	}

	// The result should be a slice of object references
	if objRefs, ok := result.([]interface{}); ok {
		channels := make([]*NotificationChannelClient, 0, len(objRefs))
		for _, ref := range objRefs {
			if objRef, ok := ref.(*ObjectRef); ok {
				channels = append(channels, &NotificationChannelClient{objectRef: objRef})
			}
		}
		return channels, nil
	}

	return nil, fmt.Errorf("unexpected result type from ListChannels")
}

// DeleteChannel deletes a notification channel
func (c *EventChannelFactoryClient) DeleteChannel(name string) error {
	_, err := c.objectRef.Invoke("DeleteChannel", name)
	return err
}

// FilterFactoryClient is a client proxy for interacting with a remote Filter Factory
type FilterFactoryClient struct {
	objectRef *ObjectRef
}

// CreateFilter creates a new filter with the given constraint language
func (c *FilterFactoryClient) CreateFilter(constraintLanguage string) (*FilterClient, error) {
	result, err := c.objectRef.Invoke("CreateFilter", constraintLanguage)
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a filter
	if objRef, ok := result.(*ObjectRef); ok {
		return &FilterClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from CreateFilter")
}

// FilterClient is a client proxy for interacting with a remote Filter
type FilterClient struct {
	objectRef *ObjectRef
}

// GetConstraints returns all constraints
func (c *FilterClient) GetConstraints() ([]FilterConstraintType, error) {
	result, err := c.objectRef.Invoke("GetConstraints")
	if err != nil {
		return nil, err
	}

	// Convert the result to constraints
	constraints := []FilterConstraintType{}
	if constArray, ok := result.([]interface{}); ok {
		for _, constItem := range constArray {
			if constMap, ok := constItem.(map[string]interface{}); ok {
				constraint := FilterConstraintType{
					ConstraintID: fmt.Sprintf("%v", constMap["ConstraintID"]),
					Expression:   fmt.Sprintf("%v", constMap["Expression"]),
				}
				constraints = append(constraints, constraint)
			}
		}
		return constraints, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetConstraints")
}

// AddConstraint adds a constraint
func (c *FilterClient) AddConstraint(constraintID string, expression string) error {
	_, err := c.objectRef.Invoke("AddConstraint", constraintID, expression)
	return err
}

// RemoveConstraint removes a constraint
func (c *FilterClient) RemoveConstraint(constraintID string) error {
	_, err := c.objectRef.Invoke("RemoveConstraint", constraintID)
	return err
}

// ModifyConstraint modifies a constraint
func (c *FilterClient) ModifyConstraint(constraintID string, expression string) error {
	_, err := c.objectRef.Invoke("ModifyConstraint", constraintID, expression)
	return err
}

// GetConstraint gets a constraint
func (c *FilterClient) GetConstraint(constraintID string) (FilterConstraintType, error) {
	result, err := c.objectRef.Invoke("GetConstraint", constraintID)
	if err != nil {
		return FilterConstraintType{}, err
	}

	// Convert the result to a constraint
	if constMap, ok := result.(map[string]interface{}); ok {
		constraint := FilterConstraintType{
			ConstraintID: fmt.Sprintf("%v", constMap["ConstraintID"]),
			Expression:   fmt.Sprintf("%v", constMap["Expression"]),
		}
		return constraint, nil
	}

	return FilterConstraintType{}, fmt.Errorf("unexpected result type from GetConstraint")
}

// Match evaluates whether an event matches this filter
func (c *FilterClient) Match(event StructuredEvent) bool {
	result, err := c.objectRef.Invoke("Match", event)
	if err != nil {
		return false
	}

	if match, ok := result.(bool); ok {
		return match
	}

	return false
}

// NotificationChannelClient is a client proxy for interacting with a remote Notification Channel
type NotificationChannelClient struct {
	objectRef          *ObjectRef
	eventChannelClient *EventChannelClient
}

// ID returns the channel's ID (inherits from EventChannel)
func (c *NotificationChannelClient) ID() string {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.ID()
}

// Name returns the channel's name (inherits from EventChannel)
func (c *NotificationChannelClient) Name() string {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.Name()
}

// Type returns the channel's type (inherits from EventChannel)
func (c *NotificationChannelClient) Type() EventChannelType {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.Type()
}

// ConnectConsumer connects a consumer to this channel (inherits from EventChannel)
func (c *NotificationChannelClient) ConnectConsumer(consumer EventConsumer) error {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.ConnectConsumer(consumer)
}

// ConnectSupplier connects a supplier to this channel (inherits from EventChannel)
func (c *NotificationChannelClient) ConnectSupplier(supplier EventSupplier) error {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.ConnectSupplier(supplier)
}

// DisconnectConsumer disconnects a consumer from this channel (inherits from EventChannel)
func (c *NotificationChannelClient) DisconnectConsumer(consumer EventConsumer) error {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.DisconnectConsumer(consumer)
}

// DisconnectSupplier disconnects a supplier from this channel (inherits from EventChannel)
func (c *NotificationChannelClient) DisconnectSupplier(supplier EventSupplier) error {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.DisconnectSupplier(supplier)
}

// Destroy destroys this channel (inherits from EventChannel)
func (c *NotificationChannelClient) Destroy() error {
	// Initialize the event channel client if needed
	if c.eventChannelClient == nil {
		c.eventChannelClient = &EventChannelClient{objectRef: c.objectRef}
	}
	return c.eventChannelClient.Destroy()
}

// GetQoS returns the QoS properties for this channel
func (c *NotificationChannelClient) GetQoS() (QoS, error) {
	result, err := c.objectRef.Invoke("GetQoS")
	if err != nil {
		return QoS{}, err
	}

	// Convert the result to QoS
	if qosMap, ok := result.(map[string]interface{}); ok {
		qos := QoS{
			Properties: make(map[QoSPropertyType]interface{}),
		}

		// Extract properties
		if propsMap, ok := qosMap["Properties"].(map[string]interface{}); ok {
			for k, v := range propsMap {
				qos.Properties[QoSPropertyType(k)] = v
			}
		}

		return qos, nil
	}

	return QoS{}, fmt.Errorf("unexpected result type from GetQoS")
}

// SetQoS sets the QoS properties for this channel
func (c *NotificationChannelClient) SetQoS(qos QoS) error {
	_, err := c.objectRef.Invoke("SetQoS", qos)
	return err
}

// GetAdminProperties returns the admin properties for this channel
func (c *NotificationChannelClient) GetAdminProperties() (AdminPropertiesType, error) {
	result, err := c.objectRef.Invoke("GetAdminProperties")
	if err != nil {
		return AdminPropertiesType{}, err
	}

	// Convert the result to AdminPropertiesType
	if propsMap, ok := result.(map[string]interface{}); ok {
		props := AdminPropertiesType{}

		if val, ok := propsMap["MaxQueueLength"].(int32); ok {
			props.MaxQueueLength = val
		}
		if val, ok := propsMap["MaxConsumers"].(int32); ok {
			props.MaxConsumers = val
		}
		if val, ok := propsMap["MaxSuppliers"].(int32); ok {
			props.MaxSuppliers = val
		}
		if val, ok := propsMap["RejectNewEvents"].(bool); ok {
			props.RejectNewEvents = val
		}

		return props, nil
	}

	return AdminPropertiesType{}, fmt.Errorf("unexpected result type from GetAdminProperties")
}

// SetAdminProperties sets the admin properties for this channel
func (c *NotificationChannelClient) SetAdminProperties(props AdminPropertiesType) error {
	_, err := c.objectRef.Invoke("SetAdminProperties", props)
	return err
}

// DefaultConsumerAdmin returns the default consumer admin
func (c *NotificationChannelClient) DefaultConsumerAdmin() (*NotificationConsumerAdminClient, error) {
	result, err := c.objectRef.Invoke("DefaultConsumerAdmin")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a consumer admin
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationConsumerAdminClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from DefaultConsumerAdmin")
}

// DefaultSupplierAdmin returns the default supplier admin
func (c *NotificationChannelClient) DefaultSupplierAdmin() (*NotificationSupplierAdminClient, error) {
	result, err := c.objectRef.Invoke("DefaultSupplierAdmin")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a supplier admin
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationSupplierAdminClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from DefaultSupplierAdmin")
}

// NewConsumerAdmin creates a new consumer admin
func (c *NotificationChannelClient) NewConsumerAdmin() (*NotificationConsumerAdminClient, error) {
	result, err := c.objectRef.Invoke("NewConsumerAdmin")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a consumer admin
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationConsumerAdminClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from NewConsumerAdmin")
}

// NewSupplierAdmin creates a new supplier admin
func (c *NotificationChannelClient) NewSupplierAdmin() (*NotificationSupplierAdminClient, error) {
	result, err := c.objectRef.Invoke("NewSupplierAdmin")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a supplier admin
	if objRef, ok := result.(*ObjectRef); ok {
		return &NotificationSupplierAdminClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from NewSupplierAdmin")
}

// GetFilterFactory returns the filter factory
func (c *NotificationChannelClient) GetFilterFactory() (*FilterFactoryClient, error) {
	result, err := c.objectRef.Invoke("GetFilterFactory")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a filter factory
	if objRef, ok := result.(*ObjectRef); ok {
		return &FilterFactoryClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetFilterFactory")
}

// NotificationConsumerAdminClient is a client proxy for interacting with a remote Notification Consumer Admin
type NotificationConsumerAdminClient struct {
	objectRef *ObjectRef
}

// ObtainPushSupplier creates a proxy supplier for push events
func (c *NotificationConsumerAdminClient) ObtainPushSupplier() (*ProxyPushSupplierClient, error) {
	result, err := c.objectRef.Invoke("ObtainPushSupplier")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a proxy push supplier
	if objRef, ok := result.(*ObjectRef); ok {
		return &ProxyPushSupplierClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from ObtainPushSupplier")
}

// ObtainPullSupplier creates a proxy supplier for pull events
func (c *NotificationConsumerAdminClient) ObtainPullSupplier() (*ProxyPullSupplierClient, error) {
	result, err := c.objectRef.Invoke("ObtainPullSupplier")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a proxy pull supplier
	if objRef, ok := result.(*ObjectRef); ok {
		return &ProxyPullSupplierClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from ObtainPullSupplier")
}

// GetQoS returns the QoS properties for this admin
func (c *NotificationConsumerAdminClient) GetQoS() (QoS, error) {
	result, err := c.objectRef.Invoke("GetQoS")
	if err != nil {
		return QoS{}, err
	}

	// Convert the result to QoS
	if qosMap, ok := result.(map[string]interface{}); ok {
		qos := QoS{
			Properties: make(map[QoSPropertyType]interface{}),
		}

		// Extract properties
		if propsMap, ok := qosMap["Properties"].(map[string]interface{}); ok {
			for k, v := range propsMap {
				qos.Properties[QoSPropertyType(k)] = v
			}
		}

		return qos, nil
	}

	return QoS{}, fmt.Errorf("unexpected result type from GetQoS")
}

// SetQoS sets the QoS properties for this admin
func (c *NotificationConsumerAdminClient) SetQoS(qos QoS) error {
	_, err := c.objectRef.Invoke("SetQoS", qos)
	return err
}

// GetFilterFactory returns the filter factory
func (c *NotificationConsumerAdminClient) GetFilterFactory() (*FilterFactoryClient, error) {
	result, err := c.objectRef.Invoke("GetFilterFactory")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a filter factory
	if objRef, ok := result.(*ObjectRef); ok {
		return &FilterFactoryClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetFilterFactory")
}

// NotificationSupplierAdminClient is a client proxy for interacting with a remote Notification Supplier Admin
type NotificationSupplierAdminClient struct {
	objectRef *ObjectRef
}

// ObtainPushConsumer creates a proxy consumer for push events
func (c *NotificationSupplierAdminClient) ObtainPushConsumer() (*ProxyPushConsumerClient, error) {
	result, err := c.objectRef.Invoke("ObtainPushConsumer")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a proxy push consumer
	if objRef, ok := result.(*ObjectRef); ok {
		return &ProxyPushConsumerClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from ObtainPushConsumer")
}

// ObtainPullConsumer creates a proxy consumer for pull events
func (c *NotificationSupplierAdminClient) ObtainPullConsumer() (*ProxyPullConsumerClient, error) {
	result, err := c.objectRef.Invoke("ObtainPullConsumer")
	if err != nil {
		return nil, err
	}

	// The result should be a remote reference to a proxy pull consumer
	if objRef, ok := result.(*ObjectRef); ok {
		return &ProxyPullConsumerClient{objectRef: objRef}, nil
	}

	return nil, fmt.Errorf("unexpected result type from ObtainPullConsumer")
}

// ProxyPushSupplierClient is a client proxy for interacting with a remote Proxy Push Supplier
type ProxyPushSupplierClient struct {
	objectRef *ObjectRef
}

// ProxyPullSupplierClient is a client proxy for interacting with a remote Proxy Pull Supplier
type ProxyPullSupplierClient struct {
	objectRef *ObjectRef
}

// ProxyPushConsumerClient is a client proxy for interacting with a remote Proxy Push Consumer
type ProxyPushConsumerClient struct {
	objectRef *ObjectRef
}

// ProxyPullConsumerClient is a client proxy for interacting with a remote Proxy Pull Consumer
type ProxyPullConsumerClient struct {
	objectRef *ObjectRef
}
