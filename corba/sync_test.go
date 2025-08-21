package corba

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChangeEvent(t *testing.T) {
	// Test creating a change event
	event := NewChangeEvent(ChangeOperationCreate, "test_object", "obj123", map[string]interface{}{
		"name": "test",
		"value": 42,
	})
	
	if event.EventID == "" {
		t.Error("EventID should not be empty")
	}
	
	if event.Operation != ChangeOperationCreate {
		t.Errorf("Expected operation %s, got %s", ChangeOperationCreate, event.Operation)
	}
	
	if event.ObjectType != "test_object" {
		t.Errorf("Expected object type 'test_object', got %s", event.ObjectType)
	}
	
	if event.ObjectID != "obj123" {
		t.Errorf("Expected object ID 'obj123', got %s", event.ObjectID)
	}
	
	if event.SourceInstance == "" {
		t.Error("SourceInstance should not be empty")
	}
	
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestChangeEventWithOldData(t *testing.T) {
	oldData := map[string]interface{}{"old": "value"}
	newData := map[string]interface{}{"new": "value"}
	
	event := NewChangeEventWithOldData(ChangeOperationUpdate, "test_object", "obj123", newData, oldData)
	
	if event.Operation != ChangeOperationUpdate {
		t.Errorf("Expected operation %s, got %s", ChangeOperationUpdate, event.Operation)
	}
	
	if event.Data == nil {
		t.Error("Data should not be nil")
	}
	
	if event.OldData == nil {
		t.Error("OldData should not be nil")
	}
}

func TestSyncConfiguration(t *testing.T) {
	config := DefaultSyncConfiguration()
	
	if config.InstanceID == "" {
		t.Error("InstanceID should not be empty")
	}
	
	if config.RedisAddr != "localhost:6379" {
		t.Errorf("Expected Redis address 'localhost:6379', got %s", config.RedisAddr)
	}
	
	if config.StreamName != "corba:sync:changes" {
		t.Errorf("Expected stream name 'corba:sync:changes', got %s", config.StreamName)
	}
	
	if !config.EnableIdempotence {
		t.Error("Idempotence should be enabled by default")
	}
}

func TestSetInstanceID(t *testing.T) {
	originalID := getInstanceID()
	
	testID := "test-instance-123"
	SetInstanceID(testID)
	
	if getInstanceID() != testID {
		t.Errorf("Expected instance ID %s, got %s", testID, getInstanceID())
	}
	
	// Restore original ID
	SetInstanceID(originalID)
}

// MockChangeListener for testing
type MockChangeListener struct {
	receivedEvents []*ChangeEvent
	subscribedTypes []string
	onChangeFunc func(ctx context.Context, event *ChangeEvent) error
}

func NewMockChangeListener(types []string) *MockChangeListener {
	return &MockChangeListener{
		receivedEvents: make([]*ChangeEvent, 0),
		subscribedTypes: types,
	}
}

func (m *MockChangeListener) OnChange(ctx context.Context, event *ChangeEvent) error {
	m.receivedEvents = append(m.receivedEvents, event)
	if m.onChangeFunc != nil {
		return m.onChangeFunc(ctx, event)
	}
	return nil
}

func (m *MockChangeListener) GetSubscribedTypes() []string {
	return m.subscribedTypes
}

func (m *MockChangeListener) Start(ctx context.Context) error {
	return nil
}

func (m *MockChangeListener) Stop() error {
	return nil
}

// MockDataLoader for testing
type MockDataLoader struct {
	supportedTypes []string
	loadDataFunc func(ctx context.Context, event *ChangeEvent) (interface{}, error)
}

func NewMockDataLoader(types []string) *MockDataLoader {
	return &MockDataLoader{
		supportedTypes: types,
	}
}

func (m *MockDataLoader) LoadData(ctx context.Context, event *ChangeEvent) (interface{}, error) {
	if m.loadDataFunc != nil {
		return m.loadDataFunc(ctx, event)
	}
	return event.Data, nil
}

func (m *MockDataLoader) GetSupportedTypes() []string {
	return m.supportedTypes
}

func (m *MockDataLoader) ValidateData(event *ChangeEvent) error {
	if event.ObjectType == "" || event.ObjectID == "" {
		return ErrInvalidChangeEvent
	}
	return nil
}

func TestEventChannelDataLoader(t *testing.T) {
	// Create a mock event service
	orb := Init()
	eventService := NewEventServiceImpl(orb)
	
	loader := NewEventChannelDataLoader(eventService)
	
	// Test supported types
	types := loader.GetSupportedTypes()
	expectedTypes := []string{"event_channel", "push_event_channel", "pull_event_channel"}
	
	if len(types) != len(expectedTypes) {
		t.Errorf("Expected %d types, got %d", len(expectedTypes), len(types))
	}
	
	// Test LoadData
	event := NewChangeEvent(ChangeOperationCreate, "event_channel", "test_channel", map[string]interface{}{
		"name": "test_channel",
		"type": int(PushChannelType),
	})
	
	data, err := loader.LoadData(context.Background(), event)
	if err != nil {
		t.Errorf("LoadData failed: %v", err)
	}
	
	if data == nil {
		t.Error("LoadData should return data for CREATE operation")
	}
	
	// Test ValidateData
	err = loader.ValidateData(event)
	if err != nil {
		t.Errorf("ValidateData failed: %v", err)
	}
	
	// Test invalid event
	invalidEvent := NewChangeEvent(ChangeOperationCreate, "", "test_channel", nil)
	err = loader.ValidateData(invalidEvent)
	if err == nil {
		t.Error("ValidateData should fail for empty object_type")
	}
}

func TestRedisStreamSynchronizer_Basic(t *testing.T) {
	// Test basic synchronizer creation and configuration
	config := &SyncConfiguration{
		InstanceID:        "test-instance",
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		StreamName:        "test:stream",
		ConsumerGroup:     "test-group",
		ConsumerName:      "test-consumer",
		BatchSize:         5,
		ReadTimeout:       1 * time.Second,
		EnableIdempotence: true,
		IdempotenceWindow: 1 * time.Hour,
	}
	
	synchronizer := NewRedisStreamSynchronizer(config)
	
	if synchronizer.GetInstanceID() != "test-instance" {
		t.Errorf("Expected instance ID 'test-instance', got %s", synchronizer.GetInstanceID())
	}
	
	// Test subscription
	listener := NewMockChangeListener([]string{"test_object"})
	
	ctx := context.Background()
	err := synchronizer.Subscribe(ctx, []string{"test_object"}, listener)
	if err != nil {
		t.Errorf("Subscribe failed: %v", err)
	}
	
	// Test data loader registration
	loader := NewMockDataLoader([]string{"test_object"})
	
	err = synchronizer.RegisterDataLoader(loader)
	if err != nil {
		t.Errorf("RegisterDataLoader failed: %v", err)
	}
	
	// Test unsubscribe
	err = synchronizer.Unsubscribe(ctx, listener)
	if err != nil {
		t.Errorf("Unsubscribe failed: %v", err)
	}
	
	// Test unregister data loader
	err = synchronizer.UnregisterDataLoader(loader)
	if err != nil {
		t.Errorf("UnregisterDataLoader failed: %v", err)
	}
}

func TestQueueMessage(t *testing.T) {
	// Test creating a queue message
	data := map[string]interface{}{
		"test": "value",
		"num":  42,
	}
	
	message := NewQueueMessage("test_type", data)
	
	if message.ID == "" {
		t.Error("Message ID should not be empty")
	}
	
	if message.Type != "test_type" {
		t.Errorf("Expected message type 'test_type', got %s", message.Type)
	}
	
	if message.Data == nil {
		t.Error("Message data should not be nil")
	}
	
	if message.Timestamp.IsZero() {
		t.Error("Message timestamp should not be zero")
	}
	
	if message.Metadata == nil {
		t.Error("Message metadata should not be nil")
	}
}

// Integration test for synchronized event service (requires running Redis)
func TestSynchronizedEventService_Integration(t *testing.T) {
	// Skip this test if Redis is not available
	config := DefaultSyncConfiguration()
	config.InstanceID = "test-instance-" + uuid.New().String()[:8]
	
	syncManager := NewRedisStreamSynchronizer(config)
	
	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Try to start the synchronizer - if Redis is not available, this will fail
	err := syncManager.Start(ctx)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
		return
	}
	defer syncManager.Stop()
	
	// Create ORB and synchronized event service
	orb := Init()
	eventService := NewSynchronizedEventService(orb, syncManager)
	
	// Create a test channel
	channelName := "integration_test_channel_" + uuid.New().String()[:8]
	channel, err := eventService.CreateChannel(channelName, PushChannelType)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}
	
	if channel.Name() != channelName {
		t.Errorf("Expected channel name %s, got %s", channelName, channel.Name())
	}
	
	// Wait a bit for synchronization
	time.Sleep(500 * time.Millisecond)
	
	// Delete the channel
	err = eventService.DeleteChannel(channelName)
	if err != nil {
		t.Errorf("Failed to delete channel: %v", err)
	}
	
	// Wait a bit for synchronization
	time.Sleep(500 * time.Millisecond)
	
	t.Logf("Integration test completed successfully")
}

func TestSynchronizedEventService_Local(t *testing.T) {
	// Test without Redis using a mock sync manager
	mockSyncManager := &MockSyncManager{}
	
	orb := Init()
	eventService := NewSynchronizedEventService(orb, mockSyncManager)
	
	// Create a test channel
	channelName := "local_test_channel"
	channel, err := eventService.CreateChannel(channelName, PushChannelType)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}
	
	if channel.Name() != channelName {
		t.Errorf("Expected channel name %s, got %s", channelName, channel.Name())
	}
	
	// Check that a change event was published
	if len(mockSyncManager.publishedEvents) != 1 {
		t.Errorf("Expected 1 published event, got %d", len(mockSyncManager.publishedEvents))
	}
	
	publishedEvent := mockSyncManager.publishedEvents[0]
	if publishedEvent.Operation != ChangeOperationCreate {
		t.Errorf("Expected CREATE operation, got %s", publishedEvent.Operation)
	}
	
	if publishedEvent.ObjectType != "event_channel" {
		t.Errorf("Expected object type 'event_channel', got %s", publishedEvent.ObjectType)
	}
	
	// Delete the channel
	err = eventService.DeleteChannel(channelName)
	if err != nil {
		t.Errorf("Failed to delete channel: %v", err)
	}
	
	// Check that a delete event was published
	if len(mockSyncManager.publishedEvents) != 2 {
		t.Errorf("Expected 2 published events, got %d", len(mockSyncManager.publishedEvents))
	}
	
	deleteEvent := mockSyncManager.publishedEvents[1]
	if deleteEvent.Operation != ChangeOperationDelete {
		t.Errorf("Expected DELETE operation, got %s", deleteEvent.Operation)
	}
}

// MockSyncManager for testing
type MockSyncManager struct {
	publishedEvents []*ChangeEvent
	listeners       map[string][]ChangeListener
	dataLoaders     map[string]DataLoader
}

func (m *MockSyncManager) PublishChange(ctx context.Context, event *ChangeEvent) error {
	if m.publishedEvents == nil {
		m.publishedEvents = make([]*ChangeEvent, 0)
	}
	m.publishedEvents = append(m.publishedEvents, event)
	return nil
}

func (m *MockSyncManager) Subscribe(ctx context.Context, objectTypes []string, listener ChangeListener) error {
	if m.listeners == nil {
		m.listeners = make(map[string][]ChangeListener)
	}
	for _, objectType := range objectTypes {
		if m.listeners[objectType] == nil {
			m.listeners[objectType] = make([]ChangeListener, 0)
		}
		m.listeners[objectType] = append(m.listeners[objectType], listener)
	}
	return nil
}

func (m *MockSyncManager) Unsubscribe(ctx context.Context, listener ChangeListener) error {
	return nil
}

func (m *MockSyncManager) RegisterDataLoader(loader DataLoader) error {
	if m.dataLoaders == nil {
		m.dataLoaders = make(map[string]DataLoader)
	}
	for _, objectType := range loader.GetSupportedTypes() {
		m.dataLoaders[objectType] = loader
	}
	return nil
}

func (m *MockSyncManager) UnregisterDataLoader(loader DataLoader) error {
	return nil
}

func (m *MockSyncManager) Start(ctx context.Context) error {
	return nil
}

func (m *MockSyncManager) Stop() error {
	return nil
}

func (m *MockSyncManager) GetInstanceID() string {
	return "mock-instance"
}