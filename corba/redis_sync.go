// Package corba provides a CORBA implementation in Go
package corba

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStreamSynchronizer implements SyncManager using Redis Streams
type RedisStreamSynchronizer struct {
	config      *SyncConfiguration
	client      *redis.Client
	listeners   map[string][]ChangeListener
	dataLoaders map[string]DataLoader
	
	// For idempotence tracking
	processedEvents map[string]time.Time
	
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
}

// NewRedisStreamSynchronizer creates a new Redis-based synchronizer
func NewRedisStreamSynchronizer(config *SyncConfiguration) *RedisStreamSynchronizer {
	if config == nil {
		config = DefaultSyncConfiguration()
	}
	
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	
	return &RedisStreamSynchronizer{
		config:          config,
		client:          client,
		listeners:       make(map[string][]ChangeListener),
		dataLoaders:     make(map[string]DataLoader),
		processedEvents: make(map[string]time.Time),
		stopChan:        make(chan struct{}),
	}
}

// PublishChange publishes a change event to other instances
func (r *RedisStreamSynchronizer) PublishChange(ctx context.Context, event *ChangeEvent) error {
	// Set source instance if not already set
	if event.SourceInstance == "" {
		event.SourceInstance = r.config.InstanceID
	}
	
	// Convert event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal change event: %w", err)
	}
	
	// Publish to Redis stream
	args := &redis.XAddArgs{
		Stream: r.config.StreamName,
		Values: map[string]interface{}{
			"event_id":      event.EventID,
			"object_type":   event.ObjectType,
			"operation":     string(event.Operation),
			"event_data":    string(eventData),
			"source_instance": event.SourceInstance,
		},
	}
	
	_, err = r.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish change event to Redis stream: %w", err)
	}
	
	log.Printf("Published change event %s for %s:%s", event.EventID, event.ObjectType, event.ObjectID)
	return nil
}

// Subscribe subscribes to changes for specific object types
func (r *RedisStreamSynchronizer) Subscribe(ctx context.Context, objectTypes []string, listener ChangeListener) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, objectType := range objectTypes {
		if r.listeners[objectType] == nil {
			r.listeners[objectType] = make([]ChangeListener, 0)
		}
		r.listeners[objectType] = append(r.listeners[objectType], listener)
	}
	
	log.Printf("Subscribed listener to object types: %v", objectTypes)
	return nil
}

// Unsubscribe unsubscribes from changes
func (r *RedisStreamSynchronizer) Unsubscribe(ctx context.Context, listener ChangeListener) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for objectType, listeners := range r.listeners {
		for i, l := range listeners {
			if l == listener {
				// Remove listener from slice
				r.listeners[objectType] = append(listeners[:i], listeners[i+1:]...)
				break
			}
		}
	}
	
	log.Printf("Unsubscribed listener")
	return nil
}

// RegisterDataLoader registers a data loader for specific object types
func (r *RedisStreamSynchronizer) RegisterDataLoader(loader DataLoader) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, objectType := range loader.GetSupportedTypes() {
		r.dataLoaders[objectType] = loader
	}
	
	log.Printf("Registered data loader for types: %v", loader.GetSupportedTypes())
	return nil
}

// UnregisterDataLoader unregisters a data loader
func (r *RedisStreamSynchronizer) UnregisterDataLoader(loader DataLoader) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, objectType := range loader.GetSupportedTypes() {
		delete(r.dataLoaders, objectType)
	}
	
	log.Printf("Unregistered data loader for types: %v", loader.GetSupportedTypes())
	return nil
}

// Start starts the synchronization manager
func (r *RedisStreamSynchronizer) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.started {
		return fmt.Errorf("synchronizer already started")
	}
	
	// Test Redis connection
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	// Create consumer group if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, r.config.StreamName, r.config.ConsumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	
	r.started = true
	
	// Start event processing goroutine
	r.wg.Add(1)
	go r.processEvents(ctx)
	
	// Start idempotence cleanup goroutine if enabled
	if r.config.EnableIdempotence {
		r.wg.Add(1)
		go r.cleanupProcessedEvents(ctx)
	}
	
	log.Printf("Redis stream synchronizer started for instance %s", r.config.InstanceID)
	return nil
}

// Stop stops the synchronization manager
func (r *RedisStreamSynchronizer) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.started {
		return nil
	}
	
	r.started = false
	close(r.stopChan)
	r.wg.Wait()
	
	// Close Redis client
	if err := r.client.Close(); err != nil {
		log.Printf("Error closing Redis client: %v", err)
	}
	
	log.Printf("Redis stream synchronizer stopped")
	return nil
}

// GetInstanceID returns the unique identifier for this instance
func (r *RedisStreamSynchronizer) GetInstanceID() string {
	return r.config.InstanceID
}

// processEvents processes events from the Redis stream
func (r *RedisStreamSynchronizer) processEvents(ctx context.Context) {
	defer r.wg.Done()
	
	for {
		select {
		case <-r.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			// Read from stream
			streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    r.config.ConsumerGroup,
				Consumer: r.config.ConsumerName,
				Streams:  []string{r.config.StreamName, ">"},
				Count:    r.config.BatchSize,
				Block:    r.config.ReadTimeout,
			}).Result()
			
			if err != nil {
				if err == redis.Nil {
					// No messages, continue
					continue
				}
				log.Printf("Error reading from Redis stream: %v", err)
				time.Sleep(time.Second)
				continue
			}
			
			// Process each stream
			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := r.processMessage(ctx, message); err != nil {
						log.Printf("Error processing message %s: %v", message.ID, err)
					}
				}
			}
		}
	}
}

// processMessage processes a single message from the stream
func (r *RedisStreamSynchronizer) processMessage(ctx context.Context, message redis.XMessage) error {
	// Extract event data
	eventDataStr, ok := message.Values["event_data"].(string)
	if !ok {
		return fmt.Errorf("invalid event_data in message")
	}
	
	// Parse event
	var event ChangeEvent
	if err := json.Unmarshal([]byte(eventDataStr), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	
	// Skip events from our own instance
	if event.SourceInstance == r.config.InstanceID {
		// Acknowledge the message
		r.client.XAck(ctx, r.config.StreamName, r.config.ConsumerGroup, message.ID)
		return nil
	}
	
	// Check for idempotence if enabled
	if r.config.EnableIdempotence {
		if r.isEventProcessed(event.EventID) {
			log.Printf("Skipping duplicate event %s", event.EventID)
			// Acknowledge the message
			r.client.XAck(ctx, r.config.StreamName, r.config.ConsumerGroup, message.ID)
			return nil
		}
		r.markEventProcessed(event.EventID)
	}
	
	// Find listeners for this object type
	r.mu.RLock()
	listeners, exists := r.listeners[event.ObjectType]
	r.mu.RUnlock()
	
	if exists {
		// Notify all listeners
		for _, listener := range listeners {
			if err := listener.OnChange(ctx, &event); err != nil {
				log.Printf("Error notifying listener of event %s: %v", event.EventID, err)
				// Don't acknowledge the message if listener fails
				return err
			}
		}
	}
	
	// Acknowledge the message
	if err := r.client.XAck(ctx, r.config.StreamName, r.config.ConsumerGroup, message.ID).Err(); err != nil {
		log.Printf("Error acknowledging message %s: %v", message.ID, err)
	}
	
	log.Printf("Processed change event %s for %s:%s", event.EventID, event.ObjectType, event.ObjectID)
	return nil
}

// isEventProcessed checks if an event has already been processed
func (r *RedisStreamSynchronizer) isEventProcessed(eventID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, exists := r.processedEvents[eventID]
	return exists
}

// markEventProcessed marks an event as processed
func (r *RedisStreamSynchronizer) markEventProcessed(eventID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.processedEvents[eventID] = time.Now()
}

// cleanupProcessedEvents periodically removes old processed events
func (r *RedisStreamSynchronizer) cleanupProcessedEvents(ctx context.Context) {
	defer r.wg.Done()
	
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-r.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			cutoff := time.Now().Add(-r.config.IdempotenceWindow)
			for eventID, timestamp := range r.processedEvents {
				if timestamp.Before(cutoff) {
					delete(r.processedEvents, eventID)
				}
			}
			r.mu.Unlock()
			
			log.Printf("Cleaned up old processed events")
		}
	}
}

// GetUnprocessedMessages returns pending messages for this consumer
func (r *RedisStreamSynchronizer) GetUnprocessedMessages(ctx context.Context) ([]redis.XMessage, error) {
	// Get pending messages for this consumer
	pendingResult, err := r.client.XPending(ctx, r.config.StreamName, r.config.ConsumerGroup).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	
	if pendingResult.Count == 0 {
		return nil, nil
	}
	
	// Get detailed pending info
	pendingExt, err := r.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   r.config.StreamName,
		Group:    r.config.ConsumerGroup,
		Consumer: r.config.ConsumerName,
		Start:    "-",
		End:      "+",
		Count:    pendingResult.Count,
	}).Result()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get pending message details: %w", err)
	}
	
	// Get the actual messages
	messageIDs := make([]string, len(pendingExt))
	for i, msg := range pendingExt {
		messageIDs[i] = msg.ID
	}
	
	if len(messageIDs) == 0 {
		return nil, nil
	}
	
	// Read the messages by ID
	streams := append([]string{r.config.StreamName}, messageIDs...)
	result, err := r.client.XRead(ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   int64(len(messageIDs)),
	}).Result()
	
	if err != nil {
		return nil, fmt.Errorf("failed to read pending messages: %w", err)
	}
	
	if len(result) == 0 {
		return nil, nil
	}
	
	return result[0].Messages, nil
}

// ReprocessFailedMessages reprocesses failed/pending messages
func (r *RedisStreamSynchronizer) ReprocessFailedMessages(ctx context.Context) error {
	messages, err := r.GetUnprocessedMessages(ctx)
	if err != nil {
		return err
	}
	
	for _, message := range messages {
		if err := r.processMessage(ctx, message); err != nil {
			log.Printf("Error reprocessing message %s: %v", message.ID, err)
		}
	}
	
	return nil
}