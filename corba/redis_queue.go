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

// RedisQueue implements a simple message queue using Redis lists
type RedisQueue struct {
	client    *redis.Client
	queueName string
	mu        sync.RWMutex
}

// NewRedisQueue creates a new Redis-based queue
func NewRedisQueue(redisAddr, password string, db int, queueName string) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       db,
	})
	
	return &RedisQueue{
		client:    client,
		queueName: queueName,
	}
}

// QueueMessage represents a message in the queue
type QueueMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewQueueMessage creates a new queue message
func NewQueueMessage(msgType string, data interface{}) *QueueMessage {
	return &QueueMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// Enqueue adds a message to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, message *QueueMessage) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Push to Redis list (right side)
	err = q.client.RPush(ctx, q.queueName, string(data)).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}
	
	log.Printf("Enqueued message %s of type %s to queue %s", message.ID, message.Type, q.queueName)
	return nil
}

// Dequeue removes and returns a message from the queue (blocking)
func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*QueueMessage, error) {
	// Pop from Redis list (left side) with timeout
	result, err := q.client.BLPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Timeout occurred
		}
		return nil, fmt.Errorf("failed to dequeue message: %w", err)
	}
	
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid result from Redis")
	}
	
	// Deserialize message
	var message QueueMessage
	err = json.Unmarshal([]byte(result[1]), &message)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	log.Printf("Dequeued message %s of type %s from queue %s", message.ID, message.Type, q.queueName)
	return &message, nil
}

// TryDequeue attempts to dequeue a message without blocking
func (q *RedisQueue) TryDequeue(ctx context.Context) (*QueueMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Pop from Redis list (left side) without blocking
	result, err := q.client.LPop(ctx, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Queue is empty
		}
		return nil, fmt.Errorf("failed to dequeue message: %w", err)
	}
	
	// Deserialize message
	var message QueueMessage
	err = json.Unmarshal([]byte(result), &message)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	log.Printf("Dequeued message %s of type %s from queue %s", message.ID, message.Type, q.queueName)
	return &message, nil
}

// Length returns the current length of the queue
func (q *RedisQueue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}

// Peek returns the next message without removing it from the queue
func (q *RedisQueue) Peek(ctx context.Context) (*QueueMessage, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	// Get the first element without removing it
	result, err := q.client.LIndex(ctx, q.queueName, 0).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Queue is empty
		}
		return nil, fmt.Errorf("failed to peek message: %w", err)
	}
	
	// Deserialize message
	var message QueueMessage
	err = json.Unmarshal([]byte(result), &message)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	return &message, nil
}

// Clear removes all messages from the queue
func (q *RedisQueue) Clear(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	err := q.client.Del(ctx, q.queueName).Err()
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}
	
	log.Printf("Cleared queue %s", q.queueName)
	return nil
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// RedisQueueManager manages multiple Redis queues for different message types
type RedisQueueManager struct {
	config *SyncConfiguration
	client *redis.Client
	queues map[string]*RedisQueue
	mu     sync.RWMutex
}

// NewRedisQueueManager creates a new Redis queue manager
func NewRedisQueueManager(config *SyncConfiguration) *RedisQueueManager {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	
	return &RedisQueueManager{
		config: config,
		client: client,
		queues: make(map[string]*RedisQueue),
	}
}

// GetQueue returns a queue for the specified message type
func (qm *RedisQueueManager) GetQueue(messageType string) *RedisQueue {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	queue, exists := qm.queues[messageType]
	if !exists {
		queueName := fmt.Sprintf("corba:queue:%s", messageType)
		queue = NewRedisQueue(qm.config.RedisAddr, qm.config.RedisPassword, qm.config.RedisDB, queueName)
		qm.queues[messageType] = queue
	}
	
	return queue
}

// PublishMessage publishes a message to the appropriate queue
func (qm *RedisQueueManager) PublishMessage(ctx context.Context, messageType string, data interface{}) error {
	queue := qm.GetQueue(messageType)
	message := NewQueueMessage(messageType, data)
	return queue.Enqueue(ctx, message)
}

// ConsumeMessages starts consuming messages from a queue
func (qm *RedisQueueManager) ConsumeMessages(ctx context.Context, messageType string, handler func(*QueueMessage) error) error {
	queue := qm.GetQueue(messageType)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				log.Printf("Error dequeuing message: %v", err)
				continue
			}
			
			if message == nil {
				// Timeout, continue
				continue
			}
			
			// Handle the message
			if err := handler(message); err != nil {
				log.Printf("Error handling message %s: %v", message.ID, err)
				// Could implement retry logic here
			}
		}
	}
}

// Close closes all queues and the Redis connection
func (qm *RedisQueueManager) Close() error {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	for _, queue := range qm.queues {
		if err := queue.Close(); err != nil {
			log.Printf("Error closing queue: %v", err)
		}
	}
	
	return qm.client.Close()
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), getInstanceID()[:8])
}

// RedisQueueEventPublisher publishes CORBA events to Redis queues
type RedisQueueEventPublisher struct {
	queueManager *RedisQueueManager
}

// NewRedisQueueEventPublisher creates a new Redis queue event publisher
func NewRedisQueueEventPublisher(config *SyncConfiguration) *RedisQueueEventPublisher {
	return &RedisQueueEventPublisher{
		queueManager: NewRedisQueueManager(config),
	}
}

// PublishEvent publishes a CORBA event to the appropriate queue
func (p *RedisQueueEventPublisher) PublishEvent(ctx context.Context, event Event) error {
	// Determine queue based on event type
	queueType := fmt.Sprintf("corba_event_%s", event.Type)
	
	// Wrap the CORBA event in additional metadata
	eventData := map[string]interface{}{
		"corba_event":     event,
		"source_instance": getInstanceID(),
		"timestamp":       time.Now(),
	}
	
	return p.queueManager.PublishMessage(ctx, queueType, eventData)
}

// ConsumeEvents starts consuming CORBA events from queues
func (p *RedisQueueEventPublisher) ConsumeEvents(ctx context.Context, eventType string, handler func(Event) error) error {
	queueType := fmt.Sprintf("corba_event_%s", eventType)
	
	return p.queueManager.ConsumeMessages(ctx, queueType, func(message *QueueMessage) error {
		// Extract the CORBA event from the message data
		eventData, ok := message.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid event data format")
		}
		
		// Skip events from our own instance
		sourceInstance, ok := eventData["source_instance"].(string)
		if ok && sourceInstance == getInstanceID() {
			return nil
		}
		
		corbaEventData, ok := eventData["corba_event"]
		if !ok {
			return fmt.Errorf("corba_event not found in message data")
		}
		
		// Convert back to Event struct
		// This would need proper unmarshaling logic
		event, ok := corbaEventData.(Event)
		if !ok {
			return fmt.Errorf("invalid corba event format")
		}
		
		return handler(event)
	})
}

// Close closes the queue manager
func (p *RedisQueueEventPublisher) Close() error {
	return p.queueManager.Close()
}