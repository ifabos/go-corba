# Redis-based Synchronization for Go-CORBA

This document describes the Redis-based synchronization system for the Go-CORBA implementation, which allows multiple instances to synchronize their state using Redis Streams and queues instead of traditional message queues.

## Overview

The Redis synchronization system provides:

- **Change Events**: Unique events with IDs for idempotent processing
- **Change Listeners**: Interfaces for monitoring data changes
- **Data Loaders**: Interfaces for loading data based on change events
- **Redis Stream Synchronization**: Primary synchronization mechanism using Redis Streams
- **Redis Queue Support**: Alternative queue-based messaging system

## Architecture

### Core Components

1. **ChangeEvent**: Represents a data change with unique ID for idempotence
2. **ChangeListener**: Interface for handling change events
3. **DataLoader**: Interface for loading/validating data from change events
4. **SyncManager**: Manages synchronization between instances
5. **RedisStreamSynchronizer**: Redis Streams implementation of SyncManager
6. **RedisQueue**: Simple message queue using Redis lists

### Event Flow

```
Instance A: Data Change → ChangeEvent → Redis Stream → Instance B: Process Event
```

## Usage Examples

### Basic Setup

```go
package main

import (
    "context"
    "github.com/ifabos/go-corba/corba"
)

func main() {
    // Create sync configuration
    config := corba.DefaultSyncConfiguration()
    config.InstanceID = "instance1"
    config.RedisAddr = "localhost:6379"
    
    // Create synchronizer
    syncManager := corba.NewRedisStreamSynchronizer(config)
    
    // Start synchronization
    ctx := context.Background()
    if err := syncManager.Start(ctx); err != nil {
        panic(err)
    }
    defer syncManager.Stop()
    
    // Create ORB and synchronized event service
    orb := corba.Init()
    eventService := corba.NewSynchronizedEventService(orb, syncManager)
    
    // Operations on eventService will now be synchronized across instances
    channel, err := eventService.CreateChannel("test", corba.PushChannelType)
    // This will be synchronized to other instances
}
```

### Custom Change Listener

```go
type CustomListener struct {
    name string
}

func (c *CustomListener) OnChange(ctx context.Context, event *corba.ChangeEvent) error {
    log.Printf("Received change: %s for %s:%s", 
        event.Operation, event.ObjectType, event.ObjectID)
    
    switch event.Operation {
    case corba.ChangeOperationCreate:
        // Handle creation
    case corba.ChangeOperationUpdate:
        // Handle update
    case corba.ChangeOperationDelete:
        // Handle deletion
    }
    return nil
}

func (c *CustomListener) GetSubscribedTypes() []string {
    return []string{"my_object_type"}
}

func (c *CustomListener) Start(ctx context.Context) error { return nil }
func (c *CustomListener) Stop() error { return nil }

// Register the listener
listener := &CustomListener{name: "my-listener"}
syncManager.Subscribe(ctx, listener.GetSubscribedTypes(), listener)
```

### Custom Data Loader

```go
type CustomDataLoader struct{}

func (c *CustomDataLoader) LoadData(ctx context.Context, event *corba.ChangeEvent) (interface{}, error) {
    // Load data based on the change event
    switch event.Operation {
    case corba.ChangeOperationCreate:
        return event.Data, nil
    case corba.ChangeOperationUpdate:
        return event.Data, nil
    case corba.ChangeOperationDelete:
        return nil, nil
    }
    return nil, fmt.Errorf("unsupported operation")
}

func (c *CustomDataLoader) GetSupportedTypes() []string {
    return []string{"my_object_type"}
}

func (c *CustomDataLoader) ValidateData(event *corba.ChangeEvent) error {
    if event.ObjectType == "" || event.ObjectID == "" {
        return corba.ErrInvalidChangeEvent
    }
    return nil
}

// Register the data loader
loader := &CustomDataLoader{}
syncManager.RegisterDataLoader(loader)
```

### Using Redis Queues

```go
// Create Redis queue manager
queueManager := corba.NewRedisQueueManager(config)

// Publish messages
ctx := context.Background()
err := queueManager.PublishMessage(ctx, "my_message_type", map[string]interface{}{
    "key": "value",
    "timestamp": time.Now(),
})

// Consume messages
err = queueManager.ConsumeMessages(ctx, "my_message_type", func(message *corba.QueueMessage) error {
    log.Printf("Received message: %s", message.ID)
    // Process message
    return nil
})
```

## Configuration

### SyncConfiguration Options

```go
config := &corba.SyncConfiguration{
    InstanceID:        "unique-instance-id",     // Unique identifier for this instance
    RedisAddr:         "localhost:6379",         // Redis server address
    RedisPassword:     "",                       // Redis password (if any)
    RedisDB:           0,                        // Redis database number
    StreamName:        "corba:sync:changes",     // Redis stream name
    ConsumerGroup:     "corba-sync-group",       // Consumer group name
    ConsumerName:      "consumer-1",             // Unique consumer name
    BatchSize:         10,                       // Batch size for processing events
    ReadTimeout:       5 * time.Second,         // Timeout for Redis reads
    EnableIdempotence: true,                     // Enable duplicate detection
    IdempotenceWindow: 24 * time.Hour,          // How long to track processed events
}
```

## Features

### Idempotence

The system provides idempotence through unique event IDs:

```go
event := corba.NewChangeEvent(corba.ChangeOperationCreate, "object_type", "object_id", data)
// event.EventID is automatically generated and unique
```

Processed events are tracked to prevent duplicate processing.

### Error Handling

- Failed message processing can be retried
- Pending messages can be reprocessed
- Dead letter handling for permanently failed messages

### Scalability

- Multiple consumer instances can process events in parallel
- Redis Streams provide built-in partitioning and consumer groups
- Horizontal scaling through multiple instances

## Redis Requirements

- Redis 5.0+ (for Redis Streams support)
- Recommended: Redis Cluster for high availability
- Memory sizing based on message volume and retention requirements

## Monitoring

Monitor the following metrics:

- Stream length: `XLEN corba:sync:changes`
- Consumer group lag: `XPENDING corba:sync:changes corba-sync-group`
- Failed message count
- Processing throughput

## Example: Running Multiple Instances

```bash
# Terminal 1 - Start first instance
cd examples/sync
go run main.go instance1

# Terminal 2 - Start second instance  
cd examples/sync
go run main.go instance2

# Terminal 3 - Start third instance
cd examples/sync
go run main.go instance3
```

Each instance will:
- Create event channels periodically
- Synchronize changes with other instances
- Show received change events from other instances
- Demonstrate Redis queue messaging

## Best Practices

1. **Unique Instance IDs**: Ensure each instance has a unique ID
2. **Error Handling**: Implement proper error handling in listeners
3. **Graceful Shutdown**: Stop synchronizers properly on application exit
4. **Monitoring**: Monitor Redis metrics and consumer lag
5. **Testing**: Test with Redis failures and network partitions
6. **Security**: Use Redis AUTH and SSL in production

## Integration with CORBA Event Service

The synchronization system integrates seamlessly with the existing CORBA Event Service:

- Event channels are automatically synchronized across instances
- Channel creation/deletion events are propagated
- Maintains CORBA Event Service semantics
- Transparent to existing CORBA applications

This provides a robust foundation for building distributed CORBA applications with automatic state synchronization.