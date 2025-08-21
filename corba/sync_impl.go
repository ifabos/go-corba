// Package corba provides a CORBA implementation in Go
package corba

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// EventChannelDataLoader implements DataLoader for CORBA Event Channels
type EventChannelDataLoader struct {
	eventService *EventServiceImpl
}

// NewEventChannelDataLoader creates a new data loader for Event Channels
func NewEventChannelDataLoader(eventService *EventServiceImpl) *EventChannelDataLoader {
	return &EventChannelDataLoader{
		eventService: eventService,
	}
}

// LoadData loads event channel data based on a change event
func (e *EventChannelDataLoader) LoadData(ctx context.Context, event *ChangeEvent) (interface{}, error) {
	switch event.Operation {
	case ChangeOperationCreate, ChangeOperationUpdate:
		// For CREATE/UPDATE, the data should be in the event
		return event.Data, nil
	case ChangeOperationDelete:
		// For DELETE, we just need to know the channel was deleted
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", event.Operation)
	}
}

// GetSupportedTypes returns the object types this loader can handle
func (e *EventChannelDataLoader) GetSupportedTypes() []string {
	return []string{"event_channel", "push_event_channel", "pull_event_channel"}
}

// ValidateData validates that the data in the change event is valid
func (e *EventChannelDataLoader) ValidateData(event *ChangeEvent) error {
	if event.ObjectType == "" {
		return fmt.Errorf("object_type cannot be empty")
	}
	
	if event.ObjectID == "" {
		return fmt.Errorf("object_id cannot be empty")
	}
	
	switch event.Operation {
	case ChangeOperationCreate, ChangeOperationUpdate:
		if event.Data == nil {
			return fmt.Errorf("data cannot be nil for %s operation", event.Operation)
		}
	case ChangeOperationDelete:
		// DELETE operations don't need data
	default:
		return fmt.Errorf("unsupported operation: %s", event.Operation)
	}
	
	return nil
}

// EventChannelChangeListener implements ChangeListener for Event Channels
type EventChannelChangeListener struct {
	eventService *EventServiceImpl
	syncManager  SyncManager
	mu           sync.RWMutex
}

// NewEventChannelChangeListener creates a new change listener for Event Channels
func NewEventChannelChangeListener(eventService *EventServiceImpl, syncManager SyncManager) *EventChannelChangeListener {
	return &EventChannelChangeListener{
		eventService: eventService,
		syncManager:  syncManager,
	}
}

// OnChange is called when a data change occurs
func (e *EventChannelChangeListener) OnChange(ctx context.Context, event *ChangeEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	log.Printf("EventChannelChangeListener: Received change event %s for %s:%s", 
		event.EventID, event.ObjectType, event.ObjectID)
	
	switch event.Operation {
	case ChangeOperationCreate:
		return e.handleCreate(ctx, event)
	case ChangeOperationUpdate:
		return e.handleUpdate(ctx, event)
	case ChangeOperationDelete:
		return e.handleDelete(ctx, event)
	default:
		return fmt.Errorf("unsupported operation: %s", event.Operation)
	}
}

// handleCreate handles channel creation events
func (e *EventChannelChangeListener) handleCreate(ctx context.Context, event *ChangeEvent) error {
	// Extract channel data from event
	channelData, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid channel data format")
	}
	
	name, ok := channelData["name"].(string)
	if !ok {
		return fmt.Errorf("channel name not found in data")
	}
	
	channelTypeInt, ok := channelData["type"].(float64) // JSON numbers are float64
	if !ok {
		return fmt.Errorf("channel type not found in data")
	}
	
	channelType := EventChannelType(int(channelTypeInt))
	
	// Check if channel already exists (for idempotence)
	_, err := e.eventService.GetChannel(name)
	if err == nil {
		// Channel already exists, this is likely a duplicate event
		log.Printf("Channel %s already exists, skipping creation", name)
		return nil
	}
	
	// Create the channel
	_, err = e.eventService.CreateChannel(name, channelType)
	if err != nil {
		return fmt.Errorf("failed to create channel %s: %w", name, err)
	}
	
	log.Printf("Created event channel %s of type %d", name, channelType)
	return nil
}

// handleUpdate handles channel update events
func (e *EventChannelChangeListener) handleUpdate(ctx context.Context, event *ChangeEvent) error {
	// For event channels, updates might involve configuration changes
	// This would depend on what aspects of channels can be updated
	log.Printf("Update operation for event channel %s not implemented", event.ObjectID)
	return nil
}

// handleDelete handles channel deletion events
func (e *EventChannelChangeListener) handleDelete(ctx context.Context, event *ChangeEvent) error {
	// Extract channel name from object ID or metadata
	channelName := event.ObjectID
	
	// Delete the channel
	err := e.eventService.DeleteChannel(channelName)
	if err != nil {
		// If channel doesn't exist, it might have been already deleted
		if err == ErrEventChannelNotFound {
			log.Printf("Channel %s already deleted, skipping", channelName)
			return nil
		}
		return fmt.Errorf("failed to delete channel %s: %w", channelName, err)
	}
	
	log.Printf("Deleted event channel %s", channelName)
	return nil
}

// GetSubscribedTypes returns the object types this listener is interested in
func (e *EventChannelChangeListener) GetSubscribedTypes() []string {
	return []string{"event_channel", "push_event_channel", "pull_event_channel"}
}

// Start starts the listener
func (e *EventChannelChangeListener) Start(ctx context.Context) error {
	// Subscribe to the sync manager for the types we're interested in
	return e.syncManager.Subscribe(ctx, e.GetSubscribedTypes(), e)
}

// Stop stops the listener
func (e *EventChannelChangeListener) Stop() error {
	// Unsubscribe from the sync manager
	return e.syncManager.Unsubscribe(context.Background(), e)
}

// SynchronizedEventService wraps EventServiceImpl with synchronization capabilities
type SynchronizedEventService struct {
	*EventServiceImpl
	syncManager SyncManager
	mu          sync.RWMutex
}

// NewSynchronizedEventService creates a new synchronized event service
func NewSynchronizedEventService(orb *ORB, syncManager SyncManager) *SynchronizedEventService {
	eventService := NewEventServiceImpl(orb)
	
	syncEventService := &SynchronizedEventService{
		EventServiceImpl: eventService,
		syncManager:      syncManager,
	}
	
	// Register data loader and change listener
	dataLoader := NewEventChannelDataLoader(eventService)
	changeListener := NewEventChannelChangeListener(eventService, syncManager)
	
	syncManager.RegisterDataLoader(dataLoader)
	syncManager.Subscribe(context.Background(), changeListener.GetSubscribedTypes(), changeListener)
	
	return syncEventService
}

// CreateChannel creates a new event channel and publishes a change event
func (s *SynchronizedEventService) CreateChannel(name string, channelType EventChannelType) (EventChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create the channel locally first
	channel, err := s.EventServiceImpl.CreateChannel(name, channelType)
	if err != nil {
		return nil, err
	}
	
	// Prepare channel data for synchronization
	channelData := map[string]interface{}{
		"name": name,
		"type": int(channelType),
		"id":   channel.ID(),
	}
	
	// Create and publish change event
	event := NewChangeEvent(ChangeOperationCreate, "event_channel", name, channelData)
	
	if err := s.syncManager.PublishChange(context.Background(), event); err != nil {
		log.Printf("Failed to publish channel creation event: %v", err)
		// Don't fail the creation if sync fails
	}
	
	return channel, nil
}

// DeleteChannel deletes an event channel and publishes a change event
func (s *SynchronizedEventService) DeleteChannel(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Delete the channel locally first
	err := s.EventServiceImpl.DeleteChannel(name)
	if err != nil {
		return err
	}
	
	// Create and publish change event
	event := NewChangeEvent(ChangeOperationDelete, "event_channel", name, nil)
	
	if err := s.syncManager.PublishChange(context.Background(), event); err != nil {
		log.Printf("Failed to publish channel deletion event: %v", err)
		// Don't fail the deletion if sync fails
	}
	
	return nil
}

// GetSyncManager returns the sync manager
func (s *SynchronizedEventService) GetSyncManager() SyncManager {
	return s.syncManager
}