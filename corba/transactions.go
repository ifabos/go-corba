// Package corba provides a CORBA implementation in Go
package corba

import (
	"errors"
	"time"
)

// Constants for Transaction Service
const (
	TransactionServiceName = "TransactionService"

	// Transaction status values per CORBA OTS specification
	StatusActive         = 0
	StatusMarkedRollback = 1
	StatusPrepared       = 2
	StatusCommitted      = 3
	StatusRolledBack     = 4
	StatusUnknown        = 5
	StatusNoTransaction  = 6
	StatusPreparing      = 7
	StatusCommitting     = 8
	StatusRollingBack    = 9

	// Vote values for resource participants
	VoteCommit   = 0
	VoteRollback = 1
	VoteReadOnly = 2
)

// Common transaction errors
var (
	ErrTransactionUnavailable     = errors.New("transaction service unavailable")
	ErrInvalidTransaction         = errors.New("invalid transaction")
	ErrSubtransactionsUnavailable = errors.New("subtransactions unavailable")
	ErrTransactionRolledBack      = errors.New("transaction rolled back")
	ErrHeuristicRollback          = errors.New("heuristic rollback")
	ErrHeuristicCommit            = errors.New("heuristic commit")
	ErrHeuristicMixed             = errors.New("heuristic mixed")
	ErrHeuristicHazard            = errors.New("heuristic hazard")
	ErrNoPermission               = errors.New("no permission")
)

// TransactionPropagation defines how transactions are propagated
type TransactionPropagation int

const (
	// PropagationRequired - Support a current transaction; create a new one if none exists
	PropagationRequired TransactionPropagation = iota

	// PropagationSupports - Support a current transaction; execute non-transactionally if none exists
	PropagationSupports

	// PropagationMandatory - Support a current transaction; throw an exception if no current transaction exists
	PropagationMandatory

	// PropagationRequiresNew - Create a new transaction, suspending the current transaction if one exists
	PropagationRequiresNew

	// PropagationNotSupported - Do not support a current transaction; execute non-transactionally
	PropagationNotSupported

	// PropagationNever - Do not support a current transaction; throw an exception if a current transaction exists
	PropagationNever
)

// TransactionIsolation defines the isolation level for transactions
type TransactionIsolation int

const (
	// IsolationDefault - Use the default isolation level of the underlying resource
	IsolationDefault TransactionIsolation = iota

	// IsolationReadUncommitted - Dirty reads, non-repeatable reads and phantom reads can occur
	IsolationReadUncommitted

	// IsolationReadCommitted - Dirty reads are prevented; non-repeatable reads and phantom reads can occur
	IsolationReadCommitted

	// IsolationRepeatableRead - Dirty reads and non-repeatable reads are prevented; phantom reads can occur
	IsolationRepeatableRead

	// IsolationSerializable - Dirty reads, non-repeatable reads, and phantom reads are prevented
	IsolationSerializable
)

// TransactionID uniquely identifies a transaction in the system
type TransactionID string

// Control is the interface for managing a transaction
type Control interface {
	// Get the Coordinator object for the transaction
	GetCoordinator() (Coordinator, error)

	// Get the Terminator object for the transaction
	GetTerminator() (Terminator, error)
}

// Coordinator is the interface for enrolling resources in a transaction
type Coordinator interface {
	// Register a resource as a participant in the transaction
	RegisterResource(resource Resource) (RecoveryCoordinator, error)

	// Register a synchronization callback
	RegisterSynchronization(sync Synchronization) error

	// Get the transaction status
	GetStatus() (int, error)

	// Get the transaction name
	GetTransactionName() string

	// Is the transaction related to the given transaction?
	IsRelatedTransaction(xid TransactionID) bool

	// Does the transaction have the same id as the given transaction?
	IsSameTransaction(xid TransactionID) bool

	// Get the transaction ID
	GetTransactionID() TransactionID

	// Create a subtransaction
	CreateSubtransaction() (Control, error)
}

// Terminator is the interface for completing a transaction
type Terminator interface {
	// Commit the transaction
	Commit(reportHeuristics bool) error

	// Roll back the transaction
	Rollback() error
}

// Resource is the interface for a resource that can participate in a transaction
type Resource interface {
	// Prepare to commit - first phase
	Prepare() (int, error)

	// Commit the resource - second phase
	Commit() error

	// Rollback the resource
	Rollback() error

	// Commit one-phase
	CommitOnePhase() error

	// Forget a heuristic decision
	Forget() error
}

// RecoveryCoordinator is the interface for recovering from failures
type RecoveryCoordinator interface {
	// Replay completion of a transaction for a resource after a failure
	ReplayCompletion(resource Resource) (int, error)
}

// Synchronization is the interface for objects that need to be notified
// before and after a transaction completes
type Synchronization interface {
	// Before completion of the transaction
	BeforeCompletion()

	// After completion of the transaction
	AfterCompletion(status int)
}

// TransactionFactory is the interface for creating new transactions
type TransactionFactory interface {
	// Create a new transaction with default timeout
	Create() (Control, error)

	// Create a new transaction with specific timeout in seconds
	CreateWithTimeout(seconds uint32) (Control, error)
}

// Current provides access to the transaction associated with the current thread
type Current interface {
	// Begin a new transaction
	Begin() (string, error)

	// Begin a new transaction with timeout
	BeginWithTimeout(seconds uint32) (string, error)

	// Commit the current transaction
	Commit(ctrl string, reportHeuristics bool) error

	// Rollback the current transaction
	Rollback(ctrl string) error

	// Set the rollback-only status of the current transaction
	SetRollbackOnly(ctrl string) error

	// Get the transaction status
	GetStatus(ctrl string) (int, error)

	// Get the transaction associated with the current thread
	GetControl(ctrl string) (Control, error)

	// Suspend the current transaction
	Suspend(ctrl string) (Control, error)

	// Resume a suspended transaction
	Resume(ctrl string, control Control) error

	// Set the transaction timeout (in seconds)
	SetTimeout(seconds uint32) error
}

// TransactionContext represents the transaction context that is propagated
// between distributed objects
type TransactionContext struct {
	// Transaction ID
	XID TransactionID

	// Transaction timeout
	Timeout time.Duration

	// Transaction status
	Status int
}
