// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Global variables for the notification service
var (
	notificationServiceInstance *NotificationServiceImpl
)

// NotificationServiceImpl implements the CORBA Notification Service
type NotificationServiceImpl struct {
	eventService   *EventServiceImpl
	channelFactory *EventChannelFactoryImpl
	filterFactory  *FilterFactoryImpl
}

// NewNotificationServiceImpl creates a new notification service implementation
func NewNotificationServiceImpl(orb *ORB) *NotificationServiceImpl {
	// Create underlying event service first
	eventService := NewEventServiceImpl(orb)

	// Create the notification service implementation
	ns := &NotificationServiceImpl{
		eventService:  eventService,
		filterFactory: NewFilterFactoryImpl(),
	}

	// Create the channel factory
	ns.channelFactory = NewEventChannelFactoryImpl(ns)

	return ns
}

// Inherit EventService methods
func (ns *NotificationServiceImpl) CreateChannel(name string, channelType EventChannelType) (EventChannel, error) {
	return ns.eventService.CreateChannel(name, channelType)
}

func (ns *NotificationServiceImpl) GetChannel(name string) (EventChannel, error) {
	return ns.eventService.GetChannel(name)
}

func (ns *NotificationServiceImpl) ListChannels() []EventChannel {
	return ns.eventService.ListChannels()
}

func (ns *NotificationServiceImpl) DeleteChannel(name string) error {
	return ns.eventService.DeleteChannel(name)
}

// GetEventChannelFactory returns the notification event channel factory
func (ns *NotificationServiceImpl) GetEventChannelFactory() EventChannelFactory {
	return ns.channelFactory
}

// GetDefaultFilterFactory returns the default filter factory
func (ns *NotificationServiceImpl) GetDefaultFilterFactory() FilterFactory {
	return ns.filterFactory
}

// CreateStructuredEvent creates a structured event
func (ns *NotificationServiceImpl) CreateStructuredEvent(domain, type_, name string,
	filterData map[string]interface{}, payload interface{}) StructuredEvent {

	// Create the event type
	eventType := EventType{
		Domain: domain,
		Type:   type_,
	}

	// Create the fixed header
	fixedHeader := FixedEventHeader{
		EventType:  eventType,
		EventName:  name,
		DomainName: domain,
	}

	// Create the header with variable part
	header := EventHeader{
		FixedHeader:    fixedHeader,
		VariableHeader: make(map[string]interface{}),
	}

	// Add timestamp if not present
	if _, exists := header.VariableHeader["TimeStamp"]; !exists {
		header.VariableHeader["TimeStamp"] = time.Now().UnixNano()
	}

	// Create the structured event
	return StructuredEvent{
		Header:     header,
		FilterData: filterData,
		Payload:    payload,
	}
}

// EventChannelFactoryImpl implements the EventChannelFactory interface
type EventChannelFactoryImpl struct {
	notificationService *NotificationServiceImpl
	channels            map[string]NotificationChannel
	mu                  sync.RWMutex
}

// NewEventChannelFactoryImpl creates a new event channel factory implementation
func NewEventChannelFactoryImpl(ns *NotificationServiceImpl) *EventChannelFactoryImpl {
	return &EventChannelFactoryImpl{
		notificationService: ns,
		channels:            make(map[string]NotificationChannel),
	}
}

// CreateChannel creates a new notification channel
func (ecf *EventChannelFactoryImpl) CreateChannel(name string, initialQoS QoS, initialAdmin AdminPropertiesType) (NotificationChannel, error) {
	ecf.mu.Lock()
	defer ecf.mu.Unlock()

	// Check if channel already exists
	if _, exists := ecf.channels[name]; exists {
		return nil, ErrEventChannelExists
	}

	// Create a new notification channel
	channel := newNotificationChannel(name, ecf.notificationService.filterFactory, initialQoS, initialAdmin)

	// Add the channel to our map
	ecf.channels[name] = channel

	return channel, nil
}

// GetChannel gets an existing notification channel
func (ecf *EventChannelFactoryImpl) GetChannel(name string) (NotificationChannel, error) {
	ecf.mu.RLock()
	defer ecf.mu.RUnlock()

	channel, exists := ecf.channels[name]
	if !exists {
		return nil, ErrEventChannelNotFound
	}

	return channel, nil
}

// ListChannels lists all notification channels
func (ecf *EventChannelFactoryImpl) ListChannels() []NotificationChannel {
	ecf.mu.RLock()
	defer ecf.mu.RUnlock()

	channels := make([]NotificationChannel, 0, len(ecf.channels))
	for _, channel := range ecf.channels {
		channels = append(channels, channel)
	}

	return channels
}

// DeleteChannel deletes a notification channel
func (ecf *EventChannelFactoryImpl) DeleteChannel(name string) error {
	ecf.mu.Lock()
	defer ecf.mu.Unlock()

	channel, exists := ecf.channels[name]
	if !exists {
		return ErrEventChannelNotFound
	}

	// Destroy the channel first
	if err := channel.Destroy(); err != nil {
		return err
	}

	delete(ecf.channels, name)
	return nil
}

// notificationChannelImpl implements the NotificationChannel interface
type notificationChannelImpl struct {
	baseEventChannel
	qos                  QoS
	adminProperties      AdminPropertiesType
	defaultConsumerAdmin *notificationConsumerAdminImpl
	defaultSupplierAdmin *notificationSupplierAdminImpl
	consumerAdmins       map[string]*notificationConsumerAdminImpl
	supplierAdmins       map[string]*notificationSupplierAdminImpl
	filterFactory        FilterFactory
	filtersById          map[string]Filter
	nextFilterId         int
	pushEventChannel     pushEventChannel
	pullEventChannel     pullEventChannel
	mu                   sync.RWMutex
}

// newNotificationChannel creates a new notification channel
func newNotificationChannel(name string, filterFactory FilterFactory, initialQoS QoS, initialAdmin AdminPropertiesType) *notificationChannelImpl {
	id := uuid.New().String()
	channel := &notificationChannelImpl{
		baseEventChannel: baseEventChannel{
			id:          id,
			name:        name,
			channelType: PushChannelType, // Notification channels are push by default
			consumers:   make(map[string]EventConsumer),
			suppliers:   make(map[string]EventSupplier),
		},
		qos:             initialQoS,
		adminProperties: initialAdmin,
		consumerAdmins:  make(map[string]*notificationConsumerAdminImpl),
		supplierAdmins:  make(map[string]*notificationSupplierAdminImpl),
		filterFactory:   filterFactory,
		filtersById:     make(map[string]Filter),
		nextFilterId:    1,
	}

	// Initialize the push and pull channel components
	channel.pushEventChannel = pushEventChannel{
		baseEventChannel: baseEventChannel{
			id:          id + "_push",
			name:        name + "_push",
			channelType: PushChannelType,
			consumers:   make(map[string]EventConsumer),
			suppliers:   make(map[string]EventSupplier),
		},
		consumerAdmin: &pushConsumerAdminImpl{},
		supplierAdmin: &pushSupplierAdminImpl{},
	}

	channel.pushEventChannel.consumerAdmin.channel = &channel.pushEventChannel
	channel.pushEventChannel.supplierAdmin.channel = &channel.pushEventChannel

	channel.pullEventChannel = pullEventChannel{
		baseEventChannel: baseEventChannel{
			id:          id + "_pull",
			name:        name + "_pull",
			channelType: PullChannelType,
			consumers:   make(map[string]EventConsumer),
			suppliers:   make(map[string]EventSupplier),
		},
		consumerAdmin: &pullConsumerAdminImpl{},
		supplierAdmin: &pullSupplierAdminImpl{},
		eventQueue:    make([]Event, 0),
	}

	channel.pullEventChannel.consumerAdmin.channel = &channel.pullEventChannel
	channel.pullEventChannel.supplierAdmin.channel = &channel.pullEventChannel

	// Create default admin objects
	channel.defaultConsumerAdmin = newNotificationConsumerAdmin(channel, filterFactory)
	channel.defaultSupplierAdmin = newNotificationSupplierAdmin(channel, filterFactory)

	return channel
}

// GetQoS returns the QoS properties for this channel
func (c *notificationChannelImpl) GetQoS() QoS {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.qos
}

// SetQoS sets the QoS properties for this channel
func (c *notificationChannelImpl) SetQoS(qos QoS) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate QoS settings
	for property, value := range qos.Properties {
		switch property {
		case QoS_EventReliability, QoS_ConnectionReliability:
			reliability, ok := value.(int)
			if !ok || reliability < Reliability_BestEffort || reliability > Reliability_BestEffortPers {
				return ErrQoSNotSupported
			}
		case QoS_Priority:
			priority, ok := value.(int32)
			if !ok || priority < -32767 || priority > 32767 {
				return ErrQoSNotSupported
			}
		case QoS_Timeout:
			_, ok := value.(int64)
			if !ok {
				return ErrQoSNotSupported
			}
		case QoS_StartTimeSupported, QoS_StopTimeSupported:
			_, ok := value.(bool)
			if !ok {
				return ErrQoSNotSupported
			}
		case QoS_MaxEventsPerConsumer:
			count, ok := value.(int32)
			if !ok || count < 0 {
				return ErrQoSNotSupported
			}
		case QoS_OrderPolicy:
			policy, ok := value.(int)
			if !ok || policy < OrderPolicy_AnyOrder || policy > OrderPolicy_DeadlineOrder {
				return ErrQoSNotSupported
			}
		case QoS_DiscardPolicy:
			policy, ok := value.(int)
			if !ok || policy < DiscardPolicy_AnyOrder || policy > DiscardPolicy_LifoOrder {
				return ErrQoSNotSupported
			}
		case QoS_MaximumBatchSize:
			size, ok := value.(int32)
			if !ok || size <= 0 {
				return ErrQoSNotSupported
			}
		case QoS_PacingInterval:
			interval, ok := value.(int64)
			if !ok || interval < 0 {
				return ErrQoSNotSupported
			}
		default:
			return ErrQoSNotSupported
		}
	}

	// Update QoS
	if c.qos.Properties == nil {
		c.qos.Properties = make(map[QoSPropertyType]interface{})
	}
	for property, value := range qos.Properties {
		c.qos.Properties[property] = value
	}

	return nil
}

// GetAdminProperties returns the admin properties for this channel
func (c *notificationChannelImpl) GetAdminProperties() AdminPropertiesType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.adminProperties
}

// SetAdminProperties sets the admin properties for this channel
func (c *notificationChannelImpl) SetAdminProperties(props AdminPropertiesType) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate admin properties
	if props.MaxQueueLength < 0 || props.MaxConsumers < 0 || props.MaxSuppliers < 0 {
		return ErrInvalidAdminProperty
	}

	c.adminProperties = props
	return nil
}

// DefaultConsumerAdmin returns the default consumer admin
func (c *notificationChannelImpl) DefaultConsumerAdmin() NotificationConsumerAdmin {
	return c.defaultConsumerAdmin
}

// DefaultSupplierAdmin returns the default supplier admin
func (c *notificationChannelImpl) DefaultSupplierAdmin() NotificationSupplierAdmin {
	return c.defaultSupplierAdmin
}

// NewConsumerAdmin creates a new consumer admin
func (c *notificationChannelImpl) NewConsumerAdmin() (NotificationConsumerAdmin, error) {
	admin := newNotificationConsumerAdmin(c, c.filterFactory)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.consumerAdmins[admin.id] = admin
	return admin, nil
}

// NewSupplierAdmin creates a new supplier admin
func (c *notificationChannelImpl) NewSupplierAdmin() (NotificationSupplierAdmin, error) {
	admin := newNotificationSupplierAdmin(c, c.filterFactory)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.supplierAdmins[admin.id] = admin
	return admin, nil
}

// GetFilterFactory returns the filter factory
func (c *notificationChannelImpl) GetFilterFactory() FilterFactory {
	return c.filterFactory
}

// Destroy destroys this channel
func (c *notificationChannelImpl) Destroy() error {
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

	// Clear maps
	c.consumers = make(map[string]EventConsumer)
	c.suppliers = make(map[string]EventSupplier)
	c.consumerAdmins = make(map[string]*notificationConsumerAdminImpl)
	c.supplierAdmins = make(map[string]*notificationSupplierAdminImpl)
	c.filtersById = make(map[string]Filter)

	return nil
}

// FilterFactoryImpl implements the FilterFactory interface
type FilterFactoryImpl struct {
}

// NewFilterFactoryImpl creates a new filter factory
func NewFilterFactoryImpl() *FilterFactoryImpl {
	return &FilterFactoryImpl{}
}

// CreateFilter creates a new filter with the given constraint language
func (ff *FilterFactoryImpl) CreateFilter(constraintLanguage string) (Filter, error) {
	if constraintLanguage != "CQL" && constraintLanguage != "SQL92" {
		return nil, ErrFilterNotSupported
	}

	return NewFilter(constraintLanguage), nil
}

// FilterImpl implements the Filter interface
type FilterImpl struct {
	constraintLanguage string
	constraints        map[string]FilterConstraintType
	mu                 sync.RWMutex
}

// NewFilter creates a new filter
func NewFilter(constraintLanguage string) *FilterImpl {
	return &FilterImpl{
		constraintLanguage: constraintLanguage,
		constraints:        make(map[string]FilterConstraintType),
	}
}

// GetConstraints returns all constraints
func (f *FilterImpl) GetConstraints() []FilterConstraintType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	constraints := make([]FilterConstraintType, 0, len(f.constraints))
	for _, constraint := range f.constraints {
		constraints = append(constraints, constraint)
	}

	return constraints
}

// AddConstraint adds a constraint
func (f *FilterImpl) AddConstraint(constraintID string, expression string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constraints[constraintID]; exists {
		return ErrFilterAlreadyExists
	}

	// Validate the expression
	if err := validateFilterExpression(expression, f.constraintLanguage); err != nil {
		return err
	}

	f.constraints[constraintID] = FilterConstraintType{
		ConstraintID: constraintID,
		Expression:   expression,
	}

	return nil
}

// RemoveConstraint removes a constraint
func (f *FilterImpl) RemoveConstraint(constraintID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constraints[constraintID]; !exists {
		return ErrConstraintNotFound
	}

	delete(f.constraints, constraintID)
	return nil
}

// ModifyConstraint modifies a constraint
func (f *FilterImpl) ModifyConstraint(constraintID string, expression string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constraints[constraintID]; !exists {
		return ErrConstraintNotFound
	}

	// Validate the expression
	if err := validateFilterExpression(expression, f.constraintLanguage); err != nil {
		return err
	}

	f.constraints[constraintID] = FilterConstraintType{
		ConstraintID: constraintID,
		Expression:   expression,
	}

	return nil
}

// GetConstraint gets a constraint
func (f *FilterImpl) GetConstraint(constraintID string) (FilterConstraintType, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	constraint, exists := f.constraints[constraintID]
	if !exists {
		return FilterConstraintType{}, ErrConstraintNotFound
	}

	return constraint, nil
}

// Match evaluates whether an event matches this filter
func (f *FilterImpl) Match(event StructuredEvent) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// If no constraints, match everything
	if len(f.constraints) == 0 {
		return true
	}

	// Check each constraint, if any match, the event matches
	for _, constraint := range f.constraints {
		if matchConstraint(constraint.Expression, event, f.constraintLanguage) {
			return true
		}
	}

	return false
}

// validateFilterExpression validates a filter expression
func validateFilterExpression(expression string, language string) error {
	if expression == "" {
		return ErrInvalidFilter
	}

	// Basic validation depending on language
	if language == "CQL" {
		// CQL is a simplified language, just check it's not empty
		return nil
	} else if language == "SQL92" {
		// For SQL92, check for WHERE keyword
		if matched, _ := regexp.MatchString(`(?i)where`, expression); !matched {
			return ErrInvalidFilter
		}
	}

	return nil
}

// matchConstraint checks if an event matches a constraint
func matchConstraint(expression string, event StructuredEvent, language string) bool {
	// For simplicity, implement very basic filtering logic
	// In a real implementation, this would be a full expression evaluator

	if language == "CQL" {
		// Very simple domain/type matching for CQL
		// Format: "$domain_name == 'domain' AND $type_name == 'type'"
		if matched, _ := regexp.MatchString(fmt.Sprintf(`\$domain_name\s*==\s*['"]%s['"]`, event.Header.FixedHeader.EventType.Domain), expression); matched {
			if matched, _ := regexp.MatchString(fmt.Sprintf(`\$type_name\s*==\s*['"]%s['"]`, event.Header.FixedHeader.EventType.Type), expression); matched {
				return true
			}
		}
	} else if language == "SQL92" {
		// Very simple filter data matching for SQL92
		// This is extremely simplified - a real implementation would parse and evaluate SQL expressions
		for key, value := range event.FilterData {
			strValue := fmt.Sprintf("%v", value)
			if matched, _ := regexp.MatchString(fmt.Sprintf(`(?i)where\s+%s\s*=\s*['"]%s['"]`, key, strValue), expression); matched {
				return true
			}
		}
	}

	return false
}

// notificationConsumerAdminImpl implements the NotificationConsumerAdmin interface
type notificationConsumerAdminImpl struct {
	id            string
	channel       *notificationChannelImpl
	qos           QoS
	filterFactory FilterFactory
	filters       map[string]Filter
	nextFilterId  int
	mu            sync.RWMutex
}

// newNotificationConsumerAdmin creates a new notification consumer admin
func newNotificationConsumerAdmin(channel *notificationChannelImpl, filterFactory FilterFactory) *notificationConsumerAdminImpl {
	return &notificationConsumerAdminImpl{
		id:            uuid.New().String(),
		channel:       channel,
		qos:           QoS{Properties: make(map[QoSPropertyType]interface{})},
		filterFactory: filterFactory,
		filters:       make(map[string]Filter),
		nextFilterId:  1,
	}
}

// ObtainPushSupplier creates a proxy supplier for push events
func (ca *notificationConsumerAdminImpl) ObtainPushSupplier() ProxyPushSupplier {
	// Use the underlying event channel's consumer admin directly instead of the notification channel's defaultConsumerAdmin
	// to avoid infinite recursion
	proxy := &proxyPushSupplierImpl{
		id:       uuid.New().String(),
		channel:  &ca.channel.pushEventChannel,
		consumer: nil,
	}
	return proxy
}

// ObtainPullSupplier creates a proxy supplier for pull events
func (ca *notificationConsumerAdminImpl) ObtainPullSupplier() ProxyPullSupplier {
	// Use the underlying event channel's consumer admin directly instead of the notification channel's defaultConsumerAdmin
	// to avoid infinite recursion
	proxy := &proxyPullSupplierImpl{
		id:       uuid.New().String(),
		channel:  &ca.channel.pullEventChannel,
		consumer: nil,
	}
	return proxy
}

// GetQoS returns the QoS properties for this admin
func (ca *notificationConsumerAdminImpl) GetQoS() QoS {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.qos
}

// SetQoS sets the QoS properties for this admin
func (ca *notificationConsumerAdminImpl) SetQoS(qos QoS) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Validate QoS settings (reuse channel validation)
	return ca.channel.SetQoS(qos)
}

// GetFilterFactory returns the filter factory
func (ca *notificationConsumerAdminImpl) GetFilterFactory() FilterFactory {
	return ca.filterFactory
}

// GetAllFilters returns all filters
func (ca *notificationConsumerAdminImpl) GetAllFilters() []Filter {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	filters := make([]Filter, 0, len(ca.filters))
	for _, filter := range ca.filters {
		filters = append(filters, filter)
	}

	return filters
}

// GetFilter gets a filter by ID
func (ca *notificationConsumerAdminImpl) GetFilter(filterID string) (Filter, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	filter, exists := ca.filters[filterID]
	if !exists {
		return nil, ErrFilterNotFound
	}

	return filter, nil
}

// AddFilter adds a filter
func (ca *notificationConsumerAdminImpl) AddFilter(filter Filter) string {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	filterID := fmt.Sprintf("Filter_%d", ca.nextFilterId)
	ca.nextFilterId++

	ca.filters[filterID] = filter
	return filterID
}

// RemoveFilter removes a filter
func (ca *notificationConsumerAdminImpl) RemoveFilter(filterID string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if _, exists := ca.filters[filterID]; !exists {
		return ErrFilterNotFound
	}

	delete(ca.filters, filterID)
	return nil
}

// ObtainStructuredPushSupplier returns a proxy for structured push supplier
func (ca *notificationConsumerAdminImpl) ObtainStructuredPushSupplier() (StructuredPushSupplier, error) {
	proxy := newStructuredProxyPushSupplierImpl(ca.channel)
	return proxy, nil
}

// ObtainStructuredPullSupplier returns a proxy for structured pull supplier
func (ca *notificationConsumerAdminImpl) ObtainStructuredPullSupplier() (StructuredPullSupplier, error) {
	proxy := newStructuredProxyPullSupplierImpl(ca.channel)
	return proxy, nil
}

// notificationSupplierAdminImpl implements the NotificationSupplierAdmin interface
type notificationSupplierAdminImpl struct {
	id            string
	channel       *notificationChannelImpl
	qos           QoS
	filterFactory FilterFactory
	filters       map[string]Filter
	nextFilterId  int
	mu            sync.RWMutex
}

// newNotificationSupplierAdmin creates a new notification supplier admin
func newNotificationSupplierAdmin(channel *notificationChannelImpl, filterFactory FilterFactory) *notificationSupplierAdminImpl {
	return &notificationSupplierAdminImpl{
		id:            uuid.New().String(),
		channel:       channel,
		qos:           QoS{Properties: make(map[QoSPropertyType]interface{})},
		filterFactory: filterFactory,
		filters:       make(map[string]Filter),
		nextFilterId:  1,
	}
}

// ObtainPushConsumer creates a proxy consumer for push events
func (sa *notificationSupplierAdminImpl) ObtainPushConsumer() ProxyPushConsumer {
	// Create a new proxy directly to avoid infinite recursion through defaultSupplierAdmin
	proxy := &proxyPushConsumerImpl{
		id:       uuid.New().String(),
		channel:  &sa.channel.pushEventChannel,
		supplier: nil,
	}
	return proxy
}

// ObtainPullConsumer creates a proxy consumer for pull events
func (sa *notificationSupplierAdminImpl) ObtainPullConsumer() ProxyPullConsumer {
	// Create a new proxy directly to avoid infinite recursion through defaultSupplierAdmin
	proxy := &proxyPullConsumerImpl{
		id:       uuid.New().String(),
		channel:  &sa.channel.pullEventChannel,
		supplier: nil,
	}
	return proxy
}

// GetQoS returns the QoS properties for this admin
func (sa *notificationSupplierAdminImpl) GetQoS() QoS {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.qos
}

// SetQoS sets the QoS properties for this admin
func (sa *notificationSupplierAdminImpl) SetQoS(qos QoS) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// Validate QoS settings (reuse channel validation)
	return sa.channel.SetQoS(qos)
}

// GetFilterFactory returns the filter factory
func (sa *notificationSupplierAdminImpl) GetFilterFactory() FilterFactory {
	return sa.filterFactory
}

// GetAllFilters returns all filters
func (sa *notificationSupplierAdminImpl) GetAllFilters() []Filter {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	filters := make([]Filter, 0, len(sa.filters))
	for _, filter := range sa.filters {
		filters = append(filters, filter)
	}

	return filters
}

// GetFilter gets a filter by ID
func (sa *notificationSupplierAdminImpl) GetFilter(filterID string) (Filter, error) {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	filter, exists := sa.filters[filterID]
	if !exists {
		return nil, ErrFilterNotFound
	}

	return filter, nil
}

// AddFilter adds a filter
func (sa *notificationSupplierAdminImpl) AddFilter(filter Filter) string {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	filterID := fmt.Sprintf("Filter_%d", sa.nextFilterId)
	sa.nextFilterId++

	sa.filters[filterID] = filter
	return filterID
}

// RemoveFilter removes a filter
func (sa *notificationSupplierAdminImpl) RemoveFilter(filterID string) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if _, exists := sa.filters[filterID]; !exists {
		return ErrFilterNotFound
	}

	delete(sa.filters, filterID)
	return nil
}

// ObtainStructuredPushConsumer returns a proxy for structured push consumer
func (sa *notificationSupplierAdminImpl) ObtainStructuredPushConsumer() (StructuredPushConsumer, error) {
	proxy := newStructuredProxyPushConsumerImpl(sa.channel)
	return proxy, nil
}

// ObtainStructuredPullConsumer returns a proxy for structured pull consumer
func (sa *notificationSupplierAdminImpl) ObtainStructuredPullConsumer() (StructuredPullConsumer, error) {
	proxy := newStructuredProxyPullConsumerImpl(sa.channel)
	return proxy, nil
}

// structuredProxyPushConsumerImpl implements the StructuredPushConsumer interface
type structuredProxyPushConsumerImpl struct {
	proxyPushConsumerImpl
	channel *notificationChannelImpl
}

// newStructuredProxyPushConsumerImpl creates a new structured proxy push consumer
func newStructuredProxyPushConsumerImpl(channel *notificationChannelImpl) *structuredProxyPushConsumerImpl {
	return &structuredProxyPushConsumerImpl{
		proxyPushConsumerImpl: proxyPushConsumerImpl{
			id:      uuid.New().String(),
			channel: &channel.pushEventChannel,
		},
		channel: channel,
	}
}

// PushStructuredEvent pushes a structured event to this consumer
func (p *structuredProxyPushConsumerImpl) PushStructuredEvent(event StructuredEvent) error {
	// Convert the structured event to a regular event
	simpleEvent := Event{
		Type:    event.Header.FixedHeader.EventType.Type,
		Data:    event.Payload,
		Source:  event.Header.FixedHeader.DomainName,
		Headers: event.Header.VariableHeader,
	}

	// Apply filtering
	if !applyFilters(p.channel, event) {
		return nil // Event filtered out
	}

	// Forward to all consumers
	return p.Push(simpleEvent)
}

// PushEventBatch pushes a batch of structured events to this consumer
func (p *structuredProxyPushConsumerImpl) PushEventBatch(events EventBatch) error {
	for _, event := range events {
		if err := p.PushStructuredEvent(event); err != nil {
			return err
		}
	}
	return nil
}

// structuredProxyPullConsumerImpl implements the StructuredPullConsumer interface
type structuredProxyPullConsumerImpl struct {
	id       string
	channel  *notificationChannelImpl
	supplier StructuredPullSupplier
	mu       sync.RWMutex
}

// newStructuredProxyPullConsumerImpl creates a new structured proxy pull consumer
func newStructuredProxyPullConsumerImpl(channel *notificationChannelImpl) *structuredProxyPullConsumerImpl {
	return &structuredProxyPullConsumerImpl{
		id:      uuid.New().String(),
		channel: channel,
	}
}

// ConnectStructuredPullSupplier connects this consumer to a structured pull supplier
func (p *structuredProxyPullConsumerImpl) ConnectStructuredPullSupplier(supplier StructuredPullSupplier) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if supplier == nil {
		return fmt.Errorf("supplier cannot be nil")
	}

	p.supplier = supplier
	return nil
}

// Disconnect disconnects this consumer
func (p *structuredProxyPullConsumerImpl) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.supplier = nil
	return nil
}

// structuredProxyPushSupplierImpl implements the StructuredPushSupplier interface
type structuredProxyPushSupplierImpl struct {
	proxyPushSupplierImpl
	consumer StructuredPushConsumer
	channel  *notificationChannelImpl
}

// newStructuredProxyPushSupplierImpl creates a new structured proxy push supplier
func newStructuredProxyPushSupplierImpl(channel *notificationChannelImpl) *structuredProxyPushSupplierImpl {
	return &structuredProxyPushSupplierImpl{
		proxyPushSupplierImpl: proxyPushSupplierImpl{
			id:      uuid.New().String(),
			channel: &channel.pushEventChannel,
		},
		channel: channel,
	}
}

// ConnectStructuredPushConsumer connects this supplier to a structured push consumer
func (p *structuredProxyPushSupplierImpl) ConnectStructuredPushConsumer(consumer StructuredPushConsumer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}

	p.consumer = consumer
	return nil
}

// structuredProxyPullSupplierImpl implements the StructuredPullSupplier interface
type structuredProxyPullSupplierImpl struct {
	proxyPullSupplierImpl
	eventQueue []StructuredEvent
	channel    *notificationChannelImpl
}

// newStructuredProxyPullSupplierImpl creates a new structured proxy pull supplier
func newStructuredProxyPullSupplierImpl(channel *notificationChannelImpl) *structuredProxyPullSupplierImpl {
	return &structuredProxyPullSupplierImpl{
		proxyPullSupplierImpl: proxyPullSupplierImpl{
			id:      uuid.New().String(),
			channel: &channel.pullEventChannel,
		},
		eventQueue: make([]StructuredEvent, 0),
		channel:    channel,
	}
}

// PullStructuredEvent pulls a structured event
func (p *structuredProxyPullSupplierImpl) PullStructuredEvent() (StructuredEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.eventQueue) == 0 {
		// Block until an event is available or timeout
		p.mu.Unlock()
		time.Sleep(100 * time.Millisecond) // Polling mechanism
		p.mu.Lock()

		if len(p.eventQueue) == 0 {
			return StructuredEvent{}, fmt.Errorf("no events available")
		}
	}

	// Get the first event
	event := p.eventQueue[0]
	p.eventQueue = p.eventQueue[1:]

	// Apply filtering
	if !applyFilters(p.channel, event) {
		// If filtered out, try next event recursively
		return p.PullStructuredEvent()
	}

	return event, nil
}

// TryPullStructuredEvent attempts to pull a structured event without blocking
func (p *structuredProxyPullSupplierImpl) TryPullStructuredEvent() (StructuredEvent, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.eventQueue) == 0 {
		return StructuredEvent{}, false, nil
	}

	// Get the first event
	event := p.eventQueue[0]
	p.eventQueue = p.eventQueue[1:]

	// Apply filtering
	if !applyFilters(p.channel, event) {
		// If filtered out and we have more events, try next event recursively
		if len(p.eventQueue) > 0 {
			return p.TryPullStructuredEvent()
		}
		return StructuredEvent{}, false, nil
	}

	return event, true, nil
}

// applyFilters applies filters to an event
func applyFilters(channel *notificationChannelImpl, event StructuredEvent) bool {
	channel.mu.RLock()
	defer channel.mu.RUnlock()

	// If no filters, allow the event
	if len(channel.filtersById) == 0 {
		return true
	}

	// Check filters
	for _, filter := range channel.filtersById {
		// If any filter matches, allow the event
		if filter.Match(event) {
			return true
		}
	}

	return false
}

// NotificationServiceServant is a CORBA servant for the Notification Service
type NotificationServiceServant struct {
	service *NotificationServiceImpl
}

// NewNotificationServiceServant creates a new servant for the Notification Service
func NewNotificationServiceServant(service *NotificationServiceImpl) *NotificationServiceServant {
	return &NotificationServiceServant{
		service: service,
	}
}

// Dispatch handles incoming CORBA method calls to the Notification Service
func (nss *NotificationServiceServant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
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
		return nss.service.CreateChannel(name, EventChannelType(channelType))

	case "GetChannel":
		if len(args) != 1 {
			return nil, fmt.Errorf("GetChannel requires 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		return nss.service.GetChannel(name)

	case "ListChannels":
		return nss.service.ListChannels(), nil

	case "DeleteChannel":
		if len(args) != 1 {
			return nil, fmt.Errorf("DeleteChannel requires 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		return nil, nss.service.DeleteChannel(name)

	case "GetEventChannelFactory":
		return nss.service.GetEventChannelFactory(), nil

	case "GetDefaultFilterFactory":
		return nss.service.GetDefaultFilterFactory(), nil

	case "CreateStructuredEvent":
		if len(args) != 5 {
			return nil, fmt.Errorf("CreateStructuredEvent requires 5 arguments")
		}
		domain, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be a string")
		}
		type_, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("second argument must be a string")
		}
		name, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("third argument must be a string")
		}
		filterData, ok := args[3].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("fourth argument must be a map")
		}
		payload := args[4]
		return nss.service.CreateStructuredEvent(domain, type_, name, filterData, payload), nil

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}
