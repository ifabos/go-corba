// Package main demonstrates the Redis-based synchronization system for CORBA
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ifabos/go-corba/corba"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run sync_example.go <instance_name>")
		fmt.Println("Example: go run sync_example.go instance1")
		os.Exit(1)
	}
	
	instanceName := os.Args[1]
	
	// Set unique instance ID
	corba.SetInstanceID(instanceName)
	
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create sync configuration
	config := corba.DefaultSyncConfiguration()
	config.InstanceID = instanceName
	config.ConsumerName = fmt.Sprintf("consumer_%s", instanceName)
	
	// Check if Redis is available
	log.Printf("Starting CORBA synchronization example for instance: %s", instanceName)
	log.Printf("Redis configuration: %s (DB: %d)", config.RedisAddr, config.RedisDB)
	
	// Create Redis stream synchronizer
	syncManager := corba.NewRedisStreamSynchronizer(config)
	
	// Create ORB
	orb := corba.Init()
	
	// Create synchronized event service
	eventService := corba.NewSynchronizedEventService(orb, syncManager)
	
	// Start the synchronizer
	if err := syncManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start sync manager: %v", err)
	}
	defer syncManager.Stop()
	
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start example operations in a goroutine
	go runExampleOperations(ctx, eventService, instanceName)
	
	// Start Redis queue example in another goroutine
	go runRedisQueueExample(ctx, config, instanceName)
	
	// Wait for shutdown signal
	<-sigChan
	log.Printf("Received shutdown signal, stopping instance %s", instanceName)
	
	// Cancel context to stop all operations
	cancel()
	
	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)
	
	log.Printf("Instance %s stopped", instanceName)
}

func runExampleOperations(ctx context.Context, eventService *corba.SynchronizedEventService, instanceName string) {
	defer log.Printf("Example operations stopped for instance %s", instanceName)
	
	// Wait a bit for system to initialize
	time.Sleep(2 * time.Second)
	
	// Create some test channels
	channelCount := 0
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			channelCount++
			channelName := fmt.Sprintf("test_channel_%s_%d", instanceName, channelCount)
			
			log.Printf("Instance %s: Creating event channel %s", instanceName, channelName)
			
			// Create a channel (this will be synchronized to other instances)
			channel, err := eventService.CreateChannel(channelName, corba.PushChannelType)
			if err != nil {
				log.Printf("Failed to create channel %s: %v", channelName, err)
				continue
			}
			
			log.Printf("Instance %s: Successfully created channel %s (ID: %s)", 
				instanceName, channelName, channel.ID())
			
			// After some time, delete the channel
			go func(name string) {
				time.Sleep(30 * time.Second)
				log.Printf("Instance %s: Deleting event channel %s", instanceName, name)
				
				if err := eventService.DeleteChannel(name); err != nil {
					log.Printf("Failed to delete channel %s: %v", name, err)
				} else {
					log.Printf("Instance %s: Successfully deleted channel %s", instanceName, name)
				}
			}(channelName)
		}
	}
}

func runRedisQueueExample(ctx context.Context, config *corba.SyncConfiguration, instanceName string) {
	defer log.Printf("Redis queue example stopped for instance %s", instanceName)
	
	// Create Redis queue event publisher
	publisher := corba.NewRedisQueueEventPublisher(config)
	defer publisher.Close()
	
	// Wait a bit for system to initialize
	time.Sleep(3 * time.Second)
	
	// Start consuming events in a goroutine
	go func() {
		log.Printf("Instance %s: Starting to consume test events", instanceName)
		
		err := publisher.ConsumeEvents(ctx, "test", func(event corba.Event) error {
			log.Printf("Instance %s: Received event - Type: %s, Source: %s, Data: %v", 
				instanceName, event.Type, event.Source, event.Data)
			return nil
		})
		
		if err != nil && err != context.Canceled {
			log.Printf("Error consuming events: %v", err)
		}
	}()
	
	// Publish test events periodically
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	eventCount := 0
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eventCount++
			
			// Create a test CORBA event
			event := corba.Event{
				Type:   "test",
				Source: instanceName,
				Data: map[string]interface{}{
					"message": fmt.Sprintf("Hello from %s", instanceName),
					"count":   eventCount,
					"time":    time.Now(),
				},
				Headers: map[string]interface{}{
					"instance": instanceName,
				},
			}
			
			log.Printf("Instance %s: Publishing test event %d", instanceName, eventCount)
			
			if err := publisher.PublishEvent(ctx, event); err != nil {
				log.Printf("Failed to publish event: %v", err)
			}
		}
	}
}

// CustomDataLoader demonstrates how to implement a custom data loader
type CustomDataLoader struct {
	name string
}

func NewCustomDataLoader(name string) *CustomDataLoader {
	return &CustomDataLoader{name: name}
}

func (c *CustomDataLoader) LoadData(ctx context.Context, event *corba.ChangeEvent) (interface{}, error) {
	log.Printf("CustomDataLoader %s: Loading data for event %s", c.name, event.EventID)
	
	// Simulate loading data based on the change event
	switch event.Operation {
	case corba.ChangeOperationCreate:
		return map[string]interface{}{
			"loaded_by": c.name,
			"data":      event.Data,
			"timestamp": time.Now(),
		}, nil
	case corba.ChangeOperationUpdate:
		return event.Data, nil
	case corba.ChangeOperationDelete:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", event.Operation)
	}
}

func (c *CustomDataLoader) GetSupportedTypes() []string {
	return []string{"custom_object"}
}

func (c *CustomDataLoader) ValidateData(event *corba.ChangeEvent) error {
	if event.ObjectType != "custom_object" {
		return fmt.Errorf("unsupported object type: %s", event.ObjectType)
	}
	return nil
}

// CustomChangeListener demonstrates how to implement a custom change listener
type CustomChangeListener struct {
	name string
}

func NewCustomChangeListener(name string) *CustomChangeListener {
	return &CustomChangeListener{name: name}
}

func (c *CustomChangeListener) OnChange(ctx context.Context, event *corba.ChangeEvent) error {
	log.Printf("CustomChangeListener %s: Received change event %s for %s:%s (Operation: %s)", 
		c.name, event.EventID, event.ObjectType, event.ObjectID, event.Operation)
	
	// Process the change event
	switch event.Operation {
	case corba.ChangeOperationCreate:
		log.Printf("CustomChangeListener %s: Processing creation of %s", c.name, event.ObjectID)
	case corba.ChangeOperationUpdate:
		log.Printf("CustomChangeListener %s: Processing update of %s", c.name, event.ObjectID)
	case corba.ChangeOperationDelete:
		log.Printf("CustomChangeListener %s: Processing deletion of %s", c.name, event.ObjectID)
	}
	
	return nil
}

func (c *CustomChangeListener) GetSubscribedTypes() []string {
	return []string{"custom_object"}
}

func (c *CustomChangeListener) Start(ctx context.Context) error {
	log.Printf("CustomChangeListener %s: Started", c.name)
	return nil
}

func (c *CustomChangeListener) Stop() error {
	log.Printf("CustomChangeListener %s: Stopped", c.name)
	return nil
}