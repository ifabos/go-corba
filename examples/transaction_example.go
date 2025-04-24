// Package examples provides example code for the Go CORBA SDK
package examples

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// SimpleResource implements a basic transactional resource
type SimpleResource struct {
	name     string
	data     string
	tempData string
	mu       sync.RWMutex
}

// NewSimpleResource creates a new simple resource
func NewSimpleResource(name string) *SimpleResource {
	return &SimpleResource{
		name: name,
		data: "",
	}
}

// Prepare prepares the resource for committing
func (r *SimpleResource) Prepare() (int, error) {
	log.Printf("Resource %s: Preparing", r.name)
	r.mu.Lock()
	defer r.mu.Unlock()

	// In a real implementation, this would write to a transaction log
	// For this example, we'll just store the data in a temporary field
	r.tempData = r.data

	// Always vote commit in this example
	return corba.VoteCommit, nil
}

// Commit commits the resource changes
func (r *SimpleResource) Commit() error {
	log.Printf("Resource %s: Committing", r.name)
	r.mu.Lock()
	defer r.mu.Unlock()

	// In a real implementation, this would make the changes permanent
	// and clean up any transaction logs
	// For this example, we don't need to do anything as the data is already set

	return nil
}

// Rollback rolls back the resource changes
func (r *SimpleResource) Rollback() error {
	log.Printf("Resource %s: Rolling back", r.name)
	r.mu.Lock()
	defer r.mu.Unlock()

	// In a real implementation, this would restore from transaction logs
	// For this example, we'll restore from the temporary field
	r.data = r.tempData

	return nil
}

// CommitOnePhase performs a one-phase commit
func (r *SimpleResource) CommitOnePhase() error {
	log.Printf("Resource %s: One-phase commit", r.name)
	r.mu.Lock()
	defer r.mu.Unlock()

	// In a real implementation, this would make the changes permanent
	// without the prepare phase

	return nil
}

// Forget forgets a heuristic decision
func (r *SimpleResource) Forget() error {
	log.Printf("Resource %s: Forgetting heuristic", r.name)
	return nil
}

// UpdateData updates the resource data within a transaction
func (r *SimpleResource) UpdateData(newData string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data = newData
}

// GetData gets the resource data
func (r *SimpleResource) GetData() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.data
}

// ExportableSimpleResource makes SimpleResource exportable as a CORBA object
type ExportableSimpleResource struct {
	*SimpleResource
	objRef *corba.ObjectRef
}

// NewExportableSimpleResource creates a new exportable simple resource
func NewExportableSimpleResource(name string, orb *corba.ORB) *ExportableSimpleResource {
	resource := &ExportableSimpleResource{
		SimpleResource: NewSimpleResource(name),
	}

	// Create an ObjectRef for this resource
	objRef := &corba.ObjectRef{
		Name:       name,
		ServerHost: "localhost",
		ServerPort: 9000, // Example port
	}
	resource.objRef = objRef

	return resource
}

// ExportReference exports the resource as an ObjectRef
func (r *ExportableSimpleResource) ExportReference() *corba.ObjectRef {
	return r.objRef
}

// SimpleSynchronization implements a basic synchronization callback
type SimpleSynchronization struct {
	name string
}

// NewSimpleSynchronization creates a new simple synchronization
func NewSimpleSynchronization(name string) *SimpleSynchronization {
	return &SimpleSynchronization{
		name: name,
	}
}

// BeforeCompletion is called before the transaction completes
func (s *SimpleSynchronization) BeforeCompletion() {
	log.Printf("Sync %s: Before completion", s.name)
}

// AfterCompletion is called after the transaction completes
func (s *SimpleSynchronization) AfterCompletion(status int) {
	log.Printf("Sync %s: After completion, status: %d", s.name, status)
}

// ExportableSynchronization makes SimpleSynchronization exportable as a CORBA object
type ExportableSynchronization struct {
	*SimpleSynchronization
	objRef *corba.ObjectRef
}

// NewExportableSynchronization creates a new exportable synchronization
func NewExportableSynchronization(name string, orb *corba.ORB) *ExportableSynchronization {
	sync := &ExportableSynchronization{
		SimpleSynchronization: NewSimpleSynchronization(name),
	}

	// Create an ObjectRef for this synchronization
	objRef := &corba.ObjectRef{
		Name:       fmt.Sprintf("sync-%s", name),
		ServerHost: "localhost",
		ServerPort: 9000, // Example port
	}
	sync.objRef = objRef

	return sync
}

// ExportReference exports the synchronization as an ObjectRef
func (s *ExportableSynchronization) ExportReference() *corba.ObjectRef {
	return s.objRef
}

func main() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server for hosting the Transaction Service
	server, err := corba.Init().CreateServer("localhost", 9000)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Give the server time to start up
	time.Sleep(1 * time.Second)

	// Activate the Transaction Service on the server
	if err := orb.ActivateTransactionService(server); err != nil {
		log.Fatalf("Failed to activate Transaction Service: %v", err)
	}

	log.Println("Transaction Service activated")

	// Get the Transaction Service
	txService, err := orb.GetTransactionService()
	if err != nil {
		log.Fatalf("Failed to get Transaction Service: %v", err)
	}

	// Get the Transaction Factory
	factory := txService.GetFactory()

	// Create a transaction
	control, err := factory.Create()
	if err != nil {
		log.Fatalf("Failed to create transaction: %v", err)
	}

	log.Println("Transaction created")

	// Get the coordinator for the transaction
	coordinator, err := control.GetCoordinator()
	if err != nil {
		log.Fatalf("Failed to get coordinator: %v", err)
	}

	// Create some resources
	resource1 := NewExportableSimpleResource("Resource1", orb)
	resource2 := NewExportableSimpleResource("Resource2", orb)

	// Register the resources with the transaction
	recoveryCoord1, err := coordinator.RegisterResource(resource1)
	if err != nil {
		log.Fatalf("Failed to register Resource1: %v", err)
	}
	log.Printf("Resource1 registered with recovery coordinator: %v", recoveryCoord1)

	recoveryCoord2, err := coordinator.RegisterResource(resource2)
	if err != nil {
		log.Fatalf("Failed to register Resource2: %v", err)
	}
	log.Printf("Resource2 registered with recovery coordinator: %v", recoveryCoord2)

	// Register a synchronization callback
	sync := NewExportableSynchronization("Sync1", orb)
	if err := coordinator.RegisterSynchronization(sync); err != nil {
		log.Fatalf("Failed to register synchronization: %v", err)
	}
	log.Println("Synchronization registered")

	// Update the resources within the transaction
	resource1.UpdateData("Resource 1 Data")
	resource2.UpdateData("Resource 2 Data")

	log.Printf("Resource1 data: %s", resource1.GetData())
	log.Printf("Resource2 data: %s", resource2.GetData())

	// Get the transaction status
	status, err := coordinator.GetStatus()
	if err != nil {
		log.Fatalf("Failed to get transaction status: %v", err)
	}
	log.Printf("Transaction status: %d", status)

	// Get the transaction name
	name := coordinator.GetTransactionName()
	log.Printf("Transaction name: %s", name)

	// Get the terminator for the transaction
	terminator, err := control.GetTerminator()
	if err != nil {
		log.Fatalf("Failed to get terminator: %v", err)
	}

	// Commit the transaction
	log.Println("Committing transaction...")
	if err := terminator.Commit(true); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("Transaction committed")

	// Check the resources after the transaction
	log.Printf("Resource1 data after commit: %s", resource1.GetData())
	log.Printf("Resource2 data after commit: %s", resource2.GetData())

	// Example of using the Current interface
	log.Println("\nTesting Current interface...")

	// Get the Current interface
	current := txService.GetCurrent()

	ctrl, err := current.Begin()

	// Begin a new transaction
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	log.Println("Transaction begun via Current")

	// Get the transaction status
	status, err = current.GetStatus(ctrl)
	if err != nil {
		log.Fatalf("Failed to get current transaction status: %v", err)
	}
	log.Printf("Current transaction status: %d", status)

	// Create a new resource
	resource3 := NewExportableSimpleResource("Resource3", orb)

	// Get the control for the current transaction
	currentControl, err := current.GetControl(ctrl)
	if err != nil {
		log.Fatalf("Failed to get current transaction control: %v", err)
	}

	// Get the coordinator for the current transaction
	currentCoordinator, err := currentControl.GetCoordinator()
	if err != nil {
		log.Fatalf("Failed to get current transaction coordinator: %v", err)
	}

	// Register the resource with the transaction
	_, err = currentCoordinator.RegisterResource(resource3)
	if err != nil {
		log.Fatalf("Failed to register Resource3: %v", err)
	}
	log.Println("Resource3 registered with current transaction")

	// Update the resource
	resource3.UpdateData("Resource 3 Data")
	log.Printf("Resource3 data: %s", resource3.GetData())

	// Rollback the transaction
	log.Println("Rolling back current transaction...")
	if err := current.Rollback(ctrl); err != nil {
		log.Fatalf("Failed to rollback transaction: %v", err)
	}

	log.Println("Transaction rolled back")

	// Show how to create a subtransaction
	log.Println("\nTesting subtransactions...")

	// Create a parent transaction
	parentControl, err := factory.Create()
	if err != nil {
		log.Fatalf("Failed to create parent transaction: %v", err)
	}
	log.Println("Parent transaction created")

	// Get the coordinator for the parent transaction
	parentCoordinator, err := parentControl.GetCoordinator()
	if err != nil {
		log.Fatalf("Failed to get parent coordinator: %v", err)
	}

	// Create a subtransaction
	childControl, err := parentCoordinator.CreateSubtransaction()
	if err != nil {
		log.Fatalf("Failed to create subtransaction: %v", err)
	}
	log.Println("Subtransaction created")

	// Get the coordinator for the subtransaction
	childCoordinator, err := childControl.GetCoordinator()
	if err != nil {
		log.Fatalf("Failed to get child coordinator: %v", err)
	}

	// Check if the transactions are related
	childID := childCoordinator.GetTransactionID()
	// parentID := parentCoordinator.GetTransactionID()

	related := parentCoordinator.IsRelatedTransaction(childID)
	log.Printf("Child is related to parent: %v", related)

	// Get the terminator for the subtransaction
	childTerminator, err := childControl.GetTerminator()
	if err != nil {
		log.Fatalf("Failed to get child terminator: %v", err)
	}

	// Commit the subtransaction
	log.Println("Committing subtransaction...")
	if err := childTerminator.Commit(true); err != nil {
		log.Fatalf("Failed to commit subtransaction: %v", err)
	}
	log.Println("Subtransaction committed")

	// Get the terminator for the parent transaction
	parentTerminator, err := parentControl.GetTerminator()
	if err != nil {
		log.Fatalf("Failed to get parent terminator: %v", err)
	}

	// Commit the parent transaction
	log.Println("Committing parent transaction...")
	if err := parentTerminator.Commit(true); err != nil {
		log.Fatalf("Failed to commit parent transaction: %v", err)
	}
	log.Println("Parent transaction committed")

	log.Println("\nTransaction Service example completed successfully")

	// Shutdown the server
	server.Shutdown()
}
