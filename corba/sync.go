// Package corba provides a CORBA implementation in Go
package corba

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common synchronization errors
var (
	ErrSyncTimeout       = errors.New("synchronization timeout")
	ErrInvalidChangeEvent = errors.New("invalid change event")
	ErrDataNotFound      = errors.New("data not found")
	ErrDuplicateEvent    = errors.New("duplicate event ID (idempotence check)")
)

// ChangeOperation defines the type of change operation
type ChangeOperation string

const (
	// ChangeOperationCreate for data creation
	ChangeOperationCreate ChangeOperation = "CREATE"
	// ChangeOperationUpdate for data updates
	ChangeOperationUpdate ChangeOperation = "UPDATE"
	// ChangeOperationDelete for data deletion
	ChangeOperationDelete ChangeOperation = "DELETE"
)

// ChangeEvent represents a data change event with unique ID for idempotence
type ChangeEvent struct {
	// EventID is a unique identifier for this event, used for idempotence
	EventID string `json:"event_id"`
	
	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`
	
	// Operation type (CREATE, UPDATE, DELETE)
	Operation ChangeOperation `json:"operation"`
	
	// ObjectType represents the type of object being changed
	ObjectType string `json:"object_type"`
	
	// ObjectID is the unique identifier of the changed object
	ObjectID string `json:"object_id"`
	
	// Data contains the actual change data (new data for CREATE/UPDATE, nil for DELETE)
	Data interface{} `json:"data,omitempty"`
	
	// OldData contains the previous data (for UPDATE operations)
	OldData interface{} `json:"old_data,omitempty"`
	
	// SourceInstance identifies which instance generated this event
	SourceInstance string `json:"source_instance"`
	
	// Metadata for additional event information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewChangeEvent creates a new change event with a unique ID
func NewChangeEvent(operation ChangeOperation, objectType, objectID string, data interface{}) *ChangeEvent {
	return &ChangeEvent{
		EventID:        uuid.New().String(),
		Timestamp:      time.Now(),
		Operation:      operation,
		ObjectType:     objectType,
		ObjectID:       objectID,
		Data:          data,
		SourceInstance: getInstanceID(),
		Metadata:      make(map[string]interface{}),
	}
}

// NewChangeEventWithOldData creates a new change event with old data (for UPDATE operations)
func NewChangeEventWithOldData(operation ChangeOperation, objectType, objectID string, data, oldData interface{}) *ChangeEvent {
	event := NewChangeEvent(operation, objectType, objectID, data)
	event.OldData = oldData
	return event
}

// ChangeListener defines the interface for listening to data changes
type ChangeListener interface {
	// OnChange is called when a data change occurs
	OnChange(ctx context.Context, event *ChangeEvent) error
	
	// GetSubscribedTypes returns the object types this listener is interested in
	GetSubscribedTypes() []string
	
	// Start starts the listener
	Start(ctx context.Context) error
	
	// Stop stops the listener
	Stop() error
}

// DataLoader defines the interface for loading data based on change events
type DataLoader interface {
	// LoadData loads data for a specific object based on a change event
	LoadData(ctx context.Context, event *ChangeEvent) (interface{}, error)
	
	// GetSupportedTypes returns the object types this loader can handle
	GetSupportedTypes() []string
	
	// ValidateData validates that the data in the change event is valid
	ValidateData(event *ChangeEvent) error
}

// SyncManager defines the interface for managing synchronization between instances
type SyncManager interface {
	// PublishChange publishes a change event to other instances
	PublishChange(ctx context.Context, event *ChangeEvent) error
	
	// Subscribe subscribes to changes for specific object types
	Subscribe(ctx context.Context, objectTypes []string, listener ChangeListener) error
	
	// Unsubscribe unsubscribes from changes
	Unsubscribe(ctx context.Context, listener ChangeListener) error
	
	// RegisterDataLoader registers a data loader for specific object types
	RegisterDataLoader(loader DataLoader) error
	
	// UnregisterDataLoader unregisters a data loader
	UnregisterDataLoader(loader DataLoader) error
	
	// Start starts the synchronization manager
	Start(ctx context.Context) error
	
	// Stop stops the synchronization manager
	Stop() error
	
	// GetInstanceID returns the unique identifier for this instance
	GetInstanceID() string
}

// SyncConfiguration holds configuration for synchronization
type SyncConfiguration struct {
	// InstanceID uniquely identifies this instance
	InstanceID string
	
	// RedisAddr is the address of the Redis server
	RedisAddr string
	
	// RedisPassword for Redis authentication
	RedisPassword string
	
	// RedisDB is the Redis database number
	RedisDB int
	
	// StreamName is the Redis stream name for synchronization
	StreamName string
	
	// ConsumerGroup is the Redis consumer group name
	ConsumerGroup string
	
	// ConsumerName is the unique name for this consumer
	ConsumerName string
	
	// BatchSize for processing events in batches
	BatchSize int64
	
	// ReadTimeout for Redis stream reads
	ReadTimeout time.Duration
	
	// EnableIdempotence enables duplicate event detection
	EnableIdempotence bool
	
	// IdempotenceWindow defines how long to keep track of processed events
	IdempotenceWindow time.Duration
}

// DefaultSyncConfiguration returns a default configuration
func DefaultSyncConfiguration() *SyncConfiguration {
	return &SyncConfiguration{
		InstanceID:        uuid.New().String(),
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		StreamName:        "corba:sync:changes",
		ConsumerGroup:     "corba-sync-group",
		ConsumerName:      uuid.New().String(),
		BatchSize:         10,
		ReadTimeout:       5 * time.Second,
		EnableIdempotence: true,
		IdempotenceWindow: 24 * time.Hour,
	}
}

// Global instance ID for this process
var globalInstanceID = uuid.New().String()

// getInstanceID returns the global instance ID
func getInstanceID() string {
	return globalInstanceID
}

// SetInstanceID sets the global instance ID (should be called early in application startup)
func SetInstanceID(instanceID string) {
	globalInstanceID = instanceID
}