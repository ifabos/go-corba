// Package main provides examples for the Go CORBA SDK
package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// SimpleComponent is a custom component implementation
type SimpleComponent struct {
	*corba.ComponentServant
	Data string
}

// NewSimpleComponent creates a new simple component
func NewSimpleComponent(compType corba.ComponentType) *SimpleComponent {
	return &SimpleComponent{
		ComponentServant: corba.NewComponentServant(compType, corba.BasicComponent),
		Data:             "Initial data",
	}
}

// GetData returns the component data
func (c *SimpleComponent) GetData() string {
	return c.Data
}

// SetData sets the component data
func (c *SimpleComponent) SetData(data string) {
	c.Data = data
}

// Initialize overrides the base Initialize method
func (c *SimpleComponent) Initialize() error {
	// Call the base implementation
	if err := c.ComponentServant.Initialize(); err != nil {
		return err
	}

	fmt.Println("SimpleComponent initialized with ID:", c.GetComponentID())
	return nil
}

// Activate overrides the base Activate method
func (c *SimpleComponent) Activate() error {
	// Call the base implementation
	if err := c.ComponentServant.Activate(); err != nil {
		return err
	}

	fmt.Println("SimpleComponent activated with ID:", c.GetComponentID())
	return nil
}

// PassivateComponent passivates a component
func (c *SimpleComponent) Passivate() error {
	// Call the base implementation
	if err := c.ComponentServant.Passivate(); err != nil {
		return err
	}

	fmt.Println("SimpleComponent passivated with ID:", c.GetComponentID())
	return nil
}

// SimpleEventConsumer is a custom event consumer implementation
// that adapts our sink to the standard CORBA Event Service
type SimpleEventConsumer struct {
	Name string
	sink *SimpleEventSink
}

// ID returns the consumer ID
func (c *SimpleEventConsumer) ID() string {
	return c.Name
}

// Push handles incoming events
func (c *SimpleEventConsumer) Push(event corba.Event) error {
	// Convert standard Event to ComponentEvent
	componentEvent := corba.ComponentEvent{
		Name:      event.Type,
		Payload:   event.Data,
		Timestamp: time.Now().UnixNano(),
	}

	// Forward to the sink
	return c.sink.ConsumeEvent(componentEvent)
}

// Pull is not used in this example
func (c *SimpleEventConsumer) Pull() (corba.Event, error) {
	return corba.Event{}, fmt.Errorf("pull not implemented")
}

// TryPull is not used in this example
func (c *SimpleEventConsumer) TryPull() (corba.Event, bool, error) {
	return corba.Event{}, false, fmt.Errorf("tryPull not implemented")
}

// SimpleEventSink is a custom event sink implementation
type SimpleEventSink struct {
	Name string
}

// GetName returns the sink name
func (s *SimpleEventSink) GetName() string {
	return s.Name
}

// ConsumeEvent handles incoming events
func (s *SimpleEventSink) ConsumeEvent(event corba.ComponentEvent) error {
	fmt.Printf("Sink %s received event %s with payload: %v\n",
		s.Name, event.Name, event.Payload)
	return nil
}

func ExampleCCM() {
	fmt.Println("Starting CCM example...")

	// Initialize the ORB
	orb := corba.Init()

	// Create a CORBA server
	server, err := orb.CreateServer("localhost", 8085)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Activate the component server
	if err := orb.ActivateComponentServer(server); err != nil {
		log.Fatalf("Failed to activate component server: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n--- Creating and using components ---")

	// Get the container manager
	containerManager := orb.GetContainerManager()

	// Get the session container
	sessionContainer, err := containerManager.GetContainer(corba.SessionContainerName)
	if err != nil {
		log.Fatalf("Failed to get session container: %v", err)
	}

	// Create a custom component
	component := NewSimpleComponent(corba.SessionComponent)

	// Install the component into the container
	if err := sessionContainer.InstallComponent(component); err != nil {
		log.Fatalf("Failed to install component: %v", err)
	}

	// Activate the component
	if err := sessionContainer.ActivateComponent(component.GetComponentID()); err != nil {
		log.Fatalf("Failed to activate component: %v", err)
	}

	// Get component context
	context, err := sessionContainer.GetContainerServices().GetComponentContext(component)
	if err != nil {
		log.Fatalf("Failed to get component context: %v", err)
	}

	// Set some attributes in the context
	if err := context.SetAttribute("createdAt", time.Now().String()); err != nil {
		log.Fatalf("Failed to set attribute: %v", err)
	}

	// Demonstrate component operations
	fmt.Println("\n--- Component Operations ---")
	fmt.Println("Component ID:", component.GetComponentID())
	fmt.Println("Component Type:", component.GetType())
	fmt.Println("Component State:", component.GetState())
	component.SetData("Updated data")
	fmt.Println("Component Data:", component.GetData())

	// Create a context attribute
	if err := context.SetAttribute("description", "This is a test component"); err != nil {
		log.Fatalf("Failed to set attribute: %v", err)
	}

	// Get and print context attributes
	fmt.Println("\n--- Context Attributes ---")
	attrs := context.GetAttributes()
	for _, attr := range attrs {
		fmt.Printf("Attribute %s = %v\n", attr.Name, attr.Value)
	}

	// Demonstrate event channels
	fmt.Println("\n--- Event Channels ---")

	// Create an event sink
	sink := &SimpleEventSink{Name: "TestSink"}

	// Create a consumer adapter for the standard event service
	consumer := &SimpleEventConsumer{
		Name: "TestConsumer",
		sink: sink,
	}

	// Create an event channel
	channel, err := sessionContainer.GetContainerServices().CreateEventChannel("TestChannel")
	if err != nil {
		log.Fatalf("Failed to create event channel: %v", err)
	}

	// Connect the consumer to the channel
	if err := channel.ConnectConsumer(consumer); err != nil {
		log.Fatalf("Failed to connect consumer: %v", err)
	}

	// Create a standard Event Service event
	event := corba.Event{
		Type:    "TestEvent",
		Data:    "Hello from CCM",
		Source:  "CCMExample",
		Headers: map[string]interface{}{"timestamp": time.Now().UnixNano()},
	}

	// Push the event to all consumers in the channel
	for _, supplier := range []corba.EventConsumer{consumer} {
		if err := supplier.Push(event); err != nil {
			log.Fatalf("Failed to push event: %v", err)
		}
	}

	// Create a component reference
	fmt.Println("\n--- Component References ---")
	ref, err := orb.CreateComponentReference(component)
	if err != nil {
		log.Fatalf("Failed to create component reference: %v", err)
	}

	// Convert to string
	iorString, err := orb.ObjectToString(ref)
	if err != nil {
		log.Fatalf("Failed to convert reference to string: %v", err)
	}

	fmt.Println("Component IOR:", iorString)

	// Passivate and remove the component when done
	fmt.Println("\n--- Component Cleanup ---")

	// Destroy the event channel
	if err := channel.Destroy(); err != nil {
		log.Fatalf("Failed to destroy event channel: %v", err)
	}

	// Passivate the component
	if err := sessionContainer.PassivateComponent(component.GetComponentID()); err != nil {
		log.Fatalf("Failed to passivate component: %v", err)
	}

	// Uninstall the component
	if err := sessionContainer.UninstallComponent(component.GetComponentID()); err != nil {
		log.Fatalf("Failed to uninstall component: %v", err)
	}

	fmt.Println("Component successfully uninstalled")

	// Shutdown the server
	server.Stop()
	fmt.Println("CCM example completed")
}
