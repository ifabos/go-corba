// Package corba provides a CORBA implementation in Go
package corba

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Global variables for the transaction service
var (
	transactionServiceInstance *TransactionServiceImpl
	transactionMutex           sync.RWMutex
	activeTransactions         map[TransactionID]*TransactionImpl
	defaultTransactionTimeout  = uint32(300) // 5 minutes default timeout
)

func init() {
	activeTransactions = make(map[TransactionID]*TransactionImpl)
}

// TransactionServiceImpl implements the CORBA Transaction Service
type TransactionServiceImpl struct {
	orb            *ORB
	defaultTimeout uint32
	mu             sync.RWMutex
	factory        *TransactionFactoryImpl
	current        *TransactionCurrentImpl
}

// NewTransactionServiceImpl creates a new transaction service implementation
func NewTransactionServiceImpl(orb *ORB) *TransactionServiceImpl {
	service := &TransactionServiceImpl{
		orb:            orb,
		defaultTimeout: defaultTransactionTimeout,
	}

	service.factory = NewTransactionFactoryImpl(service)
	service.current = NewTransactionCurrentImpl(service)

	return service
}

// GetFactory returns the transaction factory
func (ts *TransactionServiceImpl) GetFactory() TransactionFactory {
	return ts.factory
}

// GetCurrent returns the Current interface for the transaction service
func (ts *TransactionServiceImpl) GetCurrent() Current {
	return ts.current
}

// SetDefaultTimeout sets the default transaction timeout in seconds
func (ts *TransactionServiceImpl) SetDefaultTimeout(seconds uint32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.defaultTimeout = seconds
}

// GetDefaultTimeout gets the default transaction timeout in seconds
func (ts *TransactionServiceImpl) GetDefaultTimeout() uint32 {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.defaultTimeout
}

// GetORB returns the ORB associated with this transaction service
func (ts *TransactionServiceImpl) GetORB() *ORB {
	return ts.orb
}

// TransactionFactoryImpl implements the TransactionFactory interface
type TransactionFactoryImpl struct {
	service *TransactionServiceImpl
}

// NewTransactionFactoryImpl creates a new transaction factory
func NewTransactionFactoryImpl(service *TransactionServiceImpl) *TransactionFactoryImpl {
	return &TransactionFactoryImpl{
		service: service,
	}
}

// Create creates a new transaction with the default timeout
func (factory *TransactionFactoryImpl) Create() (Control, error) {
	return factory.CreateWithTimeout(factory.service.GetDefaultTimeout())
}

// CreateWithTimeout creates a new transaction with the specified timeout
func (factory *TransactionFactoryImpl) CreateWithTimeout(seconds uint32) (Control, error) {
	// Create a unique transaction ID
	xid, err := generateTransactionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	// Create a new transaction implementation
	tx := &TransactionImpl{
		xid:          xid,
		status:       StatusActive,
		creationTime: time.Now(),
		timeout:      time.Duration(seconds) * time.Second,
		resources:    make([]Resource, 0),
		syncs:        make([]Synchronization, 0),
		subtx:        make([]Control, 0),
		factory:      factory,
		coordinator:  nil,
		terminator:   nil,
		transactionContext: &TransactionContext{
			XID:     xid,
			Timeout: time.Duration(seconds) * time.Second,
			Status:  StatusActive,
		},
	}

	tx.coordinator = &CoordinatorImpl{tx: tx}
	tx.terminator = &TerminatorImpl{tx: tx}

	// Register the transaction in the active transactions map
	transactionMutex.Lock()
	activeTransactions[xid] = tx
	transactionMutex.Unlock()

	// Set up a timeout if specified
	if seconds > 0 {
		go func(txid TransactionID, timeout time.Duration) {
			time.Sleep(timeout)
			handleTransactionTimeout(txid)
		}(xid, tx.timeout)
	}

	control := &ControlImpl{tx: tx}
	return control, nil
}

// TransactionCurrentImpl implements the Current interface
type TransactionCurrentImpl struct {
	service  *TransactionServiceImpl
	threadTx sync.Map // Maps thread IDs to transaction controls
	timeout  uint32   // Default timeout in seconds for new transactions
	mu       sync.RWMutex
}

// NewTransactionCurrentImpl creates a new transaction current implementation
func NewTransactionCurrentImpl(service *TransactionServiceImpl) *TransactionCurrentImpl {
	return &TransactionCurrentImpl{
		service: service,
		timeout: service.GetDefaultTimeout(),
	}
}

// Begin starts a new transaction
func (current *TransactionCurrentImpl) Begin() (string, error) {
	return current.BeginWithTimeout(current.timeout)
}

// BeginWithTimeout starts a new transaction with a specific timeout
func (current *TransactionCurrentImpl) BeginWithTimeout(seconds uint32) (string, error) {
	ctrl := uuid.New().String() // Simulate a thread ID using UUID

	// Create a new transaction
	factory := current.service.GetFactory()
	control, err := factory.CreateWithTimeout(seconds)
	if err != nil {
		return "", err
	}

	// Associate the transaction with the current thread
	current.threadTx.Store(ctrl, control)

	return ctrl, nil
}

// Commit commits the current transaction
func (current *TransactionCurrentImpl) Commit(ctrl string, reportHeuristics bool) error {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return ErrTransactionUnavailable
	}

	control := ctrlIface.(Control)

	// Remove the transaction from the current thread
	current.threadTx.Delete(ctrl)

	// Get the terminator and commit
	terminator, err := control.GetTerminator()
	if err != nil {
		return err
	}

	return terminator.Commit(reportHeuristics)
}

// Rollback rolls back the current transaction
func (current *TransactionCurrentImpl) Rollback(ctrl string) error {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return ErrTransactionUnavailable
	}

	control := ctrlIface.(Control)

	// Remove the transaction from the current thread
	current.threadTx.Delete(ctrl)

	// Get the terminator and rollback
	terminator, err := control.GetTerminator()
	if err != nil {
		return err
	}

	return terminator.Rollback()
}

// SetRollbackOnly marks the current transaction for rollback only
func (current *TransactionCurrentImpl) SetRollbackOnly(ctrl string) error {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return ErrTransactionUnavailable
	}

	control := ctrlIface.(Control)

	// Get the coordinator
	coordinator, err := control.GetCoordinator()
	if err != nil {
		return err
	}

	// Get the transaction ID
	xid := coordinator.GetTransactionID()

	transactionMutex.RLock()
	tx, exists := activeTransactions[xid]
	transactionMutex.RUnlock()

	if !exists {
		return ErrInvalidTransaction
	}

	// Mark the transaction for rollback only
	tx.mu.Lock()
	if tx.status == StatusActive {
		tx.status = StatusMarkedRollback
	}
	tx.mu.Unlock()

	return nil
}

// GetStatus returns the status of the current transaction
func (current *TransactionCurrentImpl) GetStatus(ctrl string) (int, error) {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return StatusNoTransaction, nil
	}

	control := ctrlIface.(Control)

	// Get the coordinator and status
	coordinator, err := control.GetCoordinator()
	if err != nil {
		return StatusUnknown, err
	}

	return coordinator.GetStatus()
}

// GetControl returns the Control for the current transaction
func (current *TransactionCurrentImpl) GetControl(ctrl string) (Control, error) {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return nil, ErrTransactionUnavailable
	}

	return ctrlIface.(Control), nil
}

// Suspend suspends the current transaction
func (current *TransactionCurrentImpl) Suspend(ctrl string) (Control, error) {
	// Get the control for the current transaction
	ctrlIface, exists := current.threadTx.Load(ctrl)
	if !exists {
		return nil, ErrTransactionUnavailable
	}

	// Remove the transaction from the current thread
	current.threadTx.Delete(ctrl)

	return ctrlIface.(Control), nil
}

// Resume resumes a suspended transaction
func (current *TransactionCurrentImpl) Resume(ctrl string, control Control) error {
	if control == nil {
		return ErrInvalidTransaction
	}

	// Check if a transaction is already active on this thread
	if _, exists := current.threadTx.Load(ctrl); exists {
		return ErrTransactionUnavailable
	}

	// Associate the transaction with the current thread
	current.threadTx.Store(ctrl, control)

	return nil
}

// SetTimeout sets the transaction timeout
func (current *TransactionCurrentImpl) SetTimeout(seconds uint32) error {
	current.mu.Lock()
	defer current.mu.Unlock()

	current.timeout = seconds
	return nil
}

// TransactionImpl implements a transaction
type TransactionImpl struct {
	xid                TransactionID
	status             int
	creationTime       time.Time
	timeout            time.Duration
	resources          []Resource
	syncs              []Synchronization
	subtx              []Control
	factory            *TransactionFactoryImpl
	coordinator        *CoordinatorImpl
	terminator         *TerminatorImpl
	parent             *TransactionImpl
	transactionContext *TransactionContext
	mu                 sync.RWMutex
}

// getStatus gets the current status of the transaction
func (tx *TransactionImpl) getStatus() int {
	tx.mu.RLock()
	defer tx.mu.RUnlock()

	return tx.status
}

// setStatus sets the status of the transaction
func (tx *TransactionImpl) setStatus(status int) {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.status = status
	tx.transactionContext.Status = status
}

// ControlImpl implements the Control interface
type ControlImpl struct {
	tx *TransactionImpl
}

// GetCoordinator returns the coordinator for this transaction
func (c *ControlImpl) GetCoordinator() (Coordinator, error) {
	if c.tx == nil {
		return nil, ErrInvalidTransaction
	}

	return c.tx.coordinator, nil
}

// GetTerminator returns the terminator for this transaction
func (c *ControlImpl) GetTerminator() (Terminator, error) {
	if c.tx == nil {
		return nil, ErrInvalidTransaction
	}

	return c.tx.terminator, nil
}

// CoordinatorImpl implements the Coordinator interface
type CoordinatorImpl struct {
	tx *TransactionImpl
}

// RegisterResource registers a resource as a participant in the transaction
func (c *CoordinatorImpl) RegisterResource(resource Resource) (RecoveryCoordinator, error) {
	if c.tx == nil {
		return nil, ErrInvalidTransaction
	}

	c.tx.mu.Lock()
	defer c.tx.mu.Unlock()

	// Check if the transaction is active
	if c.tx.status != StatusActive {
		return nil, fmt.Errorf("transaction is not active: %d", c.tx.status)
	}

	// Add the resource to the transaction
	c.tx.resources = append(c.tx.resources, resource)

	// Create a recovery coordinator for this resource
	rc := &RecoveryCoordinatorImpl{
		tx: c.tx,
	}

	return rc, nil
}

// RegisterSynchronization registers a synchronization callback
func (c *CoordinatorImpl) RegisterSynchronization(sync Synchronization) error {
	if c.tx == nil {
		return ErrInvalidTransaction
	}

	c.tx.mu.Lock()
	defer c.tx.mu.Unlock()

	// Check if the transaction is active
	if c.tx.status != StatusActive {
		return fmt.Errorf("transaction is not active: %d", c.tx.status)
	}

	// Add the synchronization to the transaction
	c.tx.syncs = append(c.tx.syncs, sync)

	return nil
}

// GetStatus returns the transaction status
func (c *CoordinatorImpl) GetStatus() (int, error) {
	if c.tx == nil {
		return StatusNoTransaction, ErrInvalidTransaction
	}

	return c.tx.getStatus(), nil
}

// GetTransactionName returns the name of the transaction
func (c *CoordinatorImpl) GetTransactionName() string {
	if c.tx == nil {
		return ""
	}

	return string(c.tx.xid)
}

// IsRelatedTransaction checks if the transaction is related to the given transaction
func (c *CoordinatorImpl) IsRelatedTransaction(xid TransactionID) bool {
	if c.tx == nil {
		return false
	}

	// Current implementation only checks for direct parent-child relationships
	// A more complete implementation would traverse the entire transaction tree
	current := c.tx

	for current != nil {
		if current.xid == xid {
			return true
		}
		current = current.parent
	}

	return false
}

// IsSameTransaction checks if the transaction has the same id
func (c *CoordinatorImpl) IsSameTransaction(xid TransactionID) bool {
	if c.tx == nil {
		return false
	}

	return c.tx.xid == xid
}

// GetTransactionID returns the transaction ID
func (c *CoordinatorImpl) GetTransactionID() TransactionID {
	if c.tx == nil {
		return ""
	}

	return c.tx.xid
}

// CreateSubtransaction creates a nested transaction
func (c *CoordinatorImpl) CreateSubtransaction() (Control, error) {
	if c.tx == nil {
		return nil, ErrInvalidTransaction
	}

	// Create a new transaction
	xid, err := generateTransactionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	// Create a new transaction implementation
	subtx := &TransactionImpl{
		xid:          xid,
		status:       StatusActive,
		creationTime: time.Now(),
		timeout:      c.tx.timeout, // Inherit timeout from parent
		resources:    make([]Resource, 0),
		syncs:        make([]Synchronization, 0),
		subtx:        make([]Control, 0),
		factory:      c.tx.factory,
		parent:       c.tx,
		transactionContext: &TransactionContext{
			XID:     xid,
			Timeout: c.tx.timeout,
			Status:  StatusActive,
		},
	}

	subtx.coordinator = &CoordinatorImpl{tx: subtx}
	subtx.terminator = &TerminatorImpl{tx: subtx}

	// Register the subtransaction with the parent
	c.tx.mu.Lock()
	control := &ControlImpl{tx: subtx}
	c.tx.subtx = append(c.tx.subtx, control)
	c.tx.mu.Unlock()

	// Register the transaction in the active transactions map
	transactionMutex.Lock()
	activeTransactions[xid] = subtx
	transactionMutex.Unlock()

	return control, nil
}

// TerminatorImpl implements the Terminator interface
type TerminatorImpl struct {
	tx *TransactionImpl
}

// Commit commits the transaction
func (t *TerminatorImpl) Commit(reportHeuristics bool) error {
	if t.tx == nil {
		return ErrInvalidTransaction
	}

	t.tx.mu.Lock()

	// Check if the transaction is active
	if t.tx.status == StatusMarkedRollback {
		t.tx.mu.Unlock()
		return t.Rollback()
	}

	if t.tx.status != StatusActive {
		t.tx.mu.Unlock()
		return fmt.Errorf("transaction is not active: %d", t.tx.status)
	}

	// First, run the before completion callbacks
	syncs := t.tx.syncs
	t.tx.mu.Unlock()

	for _, sync := range syncs {
		sync.BeforeCompletion()
	}

	// Set the transaction status to preparing
	t.tx.setStatus(StatusPreparing)

	// Start the 2PC protocol if there are multiple resources
	resources := t.tx.resources

	if len(resources) == 0 {
		// No resources, just commit
		t.tx.setStatus(StatusCommitted)

		// Run the after completion callbacks
		for _, sync := range syncs {
			sync.AfterCompletion(StatusCommitted)
		}

		// Remove the transaction from the active transactions map
		transactionMutex.Lock()
		delete(activeTransactions, t.tx.xid)
		transactionMutex.Unlock()

		return nil
	} else if len(resources) == 1 {
		// Only one resource, use one-phase commit
		err := resources[0].CommitOnePhase()

		if err != nil {
			// Rollback the transaction
			t.tx.setStatus(StatusRolledBack)

			// Run the after completion callbacks
			for _, sync := range syncs {
				sync.AfterCompletion(StatusRolledBack)
			}

			// Remove the transaction from the active transactions map
			transactionMutex.Lock()
			delete(activeTransactions, t.tx.xid)
			transactionMutex.Unlock()

			return err
		}

		// Commit succeeded
		t.tx.setStatus(StatusCommitted)

		// Run the after completion callbacks
		for _, sync := range syncs {
			sync.AfterCompletion(StatusCommitted)
		}

		// Remove the transaction from the active transactions map
		transactionMutex.Lock()
		delete(activeTransactions, t.tx.xid)
		transactionMutex.Unlock()

		return nil
	}

	// Multiple resources, use two-phase commit

	// Phase 1: Prepare
	var readOnlyCount int
	var voteRollback bool

	// Keep track of prepare responses for each resource
	prepareResponses := make([]int, len(resources))

	// Prepare phase
	for i, resource := range resources {
		vote, err := resource.Prepare()

		if err != nil || vote == VoteRollback {
			voteRollback = true
			// We can break here as one vote to rollback means we must rollback
			break
		}

		prepareResponses[i] = vote

		if vote == VoteReadOnly {
			readOnlyCount++
		}
	}

	// If any resource voted rollback or if all resources are read-only, we're done
	if voteRollback {
		t.tx.setStatus(StatusRollingBack)

		// Rollback all resources that did not vote read-only
		for i, resource := range resources {
			if i < len(prepareResponses) && prepareResponses[i] != VoteReadOnly {
				resource.Rollback()
			}
		}

		t.tx.setStatus(StatusRolledBack)

		// Run the after completion callbacks
		for _, sync := range syncs {
			sync.AfterCompletion(StatusRolledBack)
		}

		// Remove the transaction from the active transactions map
		transactionMutex.Lock()
		delete(activeTransactions, t.tx.xid)
		transactionMutex.Unlock()

		return ErrTransactionRolledBack
	}

	// If all resources voted read-only, we're done
	if readOnlyCount == len(resources) {
		t.tx.setStatus(StatusCommitted)

		// Run the after completion callbacks
		for _, sync := range syncs {
			sync.AfterCompletion(StatusCommitted)
		}

		// Remove the transaction from the active transactions map
		transactionMutex.Lock()
		delete(activeTransactions, t.tx.xid)
		transactionMutex.Unlock()

		return nil
	}

	// Phase 2: Commit
	t.tx.setStatus(StatusCommitting)

	// Commit all resources that voted commit (not read-only)
	var commitErrors bool

	for i, resource := range resources {
		if prepareResponses[i] == VoteCommit {
			if err := resource.Commit(); err != nil {
				commitErrors = true
				// Continue with other resources even if one fails
			}
		}
	}

	// Determine the final status
	finalStatus := StatusCommitted
	var finalError error

	if commitErrors {
		finalStatus = StatusUnknown
		if reportHeuristics {
			finalError = ErrHeuristicHazard
		}
	}

	t.tx.setStatus(finalStatus)

	// Run the after completion callbacks
	for _, sync := range syncs {
		sync.AfterCompletion(finalStatus)
	}

	// Remove the transaction from the active transactions map
	transactionMutex.Lock()
	delete(activeTransactions, t.tx.xid)
	transactionMutex.Unlock()

	return finalError
}

// Rollback rolls back the transaction
func (t *TerminatorImpl) Rollback() error {
	if t.tx == nil {
		return ErrInvalidTransaction
	}

	t.tx.mu.Lock()

	// Check if the transaction is active or marked for rollback
	if t.tx.status != StatusActive && t.tx.status != StatusMarkedRollback {
		t.tx.mu.Unlock()
		return fmt.Errorf("transaction cannot be rolled back: %d", t.tx.status)
	}

	// Set the transaction status to rolling back
	t.tx.status = StatusRollingBack

	// Get the resources and synchronizations
	resources := t.tx.resources
	syncs := t.tx.syncs
	t.tx.mu.Unlock()

	// First, run the before completion callbacks
	for _, sync := range syncs {
		sync.BeforeCompletion()
	}

	// Rollback all resources
	for _, resource := range resources {
		resource.Rollback()
	}

	// Set the transaction status to rolled back
	t.tx.setStatus(StatusRolledBack)

	// Run the after completion callbacks
	for _, sync := range syncs {
		sync.AfterCompletion(StatusRolledBack)
	}

	// Remove the transaction from the active transactions map
	transactionMutex.Lock()
	delete(activeTransactions, t.tx.xid)
	transactionMutex.Unlock()

	return nil
}

// RecoveryCoordinatorImpl implements the RecoveryCoordinator interface
type RecoveryCoordinatorImpl struct {
	tx *TransactionImpl
}

// ReplayCompletion replays the completion process for a resource
func (rc *RecoveryCoordinatorImpl) ReplayCompletion(resource Resource) (int, error) {
	if rc.tx == nil {
		return StatusUnknown, ErrInvalidTransaction
	}

	status := rc.tx.getStatus()

	switch status {
	case StatusCommitted:
		// The transaction was committed, so commit the resource
		if err := resource.Commit(); err != nil {
			return StatusUnknown, err
		}
		return StatusCommitted, nil

	case StatusRolledBack:
		// The transaction was rolled back, so roll back the resource
		if err := resource.Rollback(); err != nil {
			return StatusUnknown, err
		}
		return StatusRolledBack, nil

	default:
		// The transaction is still in progress or in an unknown state
		return status, nil
	}
}

// TransactionServiceServant is a servant for the Transaction Service
type TransactionServiceServant struct {
	service *TransactionServiceImpl
}

// GetFactory returns the transaction factory
func (servant *TransactionServiceServant) GetFactory() (TransactionFactory, error) {
	return servant.service.GetFactory(), nil
}

// GetCurrent returns the Current interface for the transaction service
func (servant *TransactionServiceServant) GetCurrent() (Current, error) {
	return servant.service.GetCurrent(), nil
}

// SetDefaultTimeout sets the default transaction timeout in seconds
func (servant *TransactionServiceServant) SetDefaultTimeout(seconds uint32) error {
	servant.service.SetDefaultTimeout(seconds)
	return nil
}

// GetDefaultTimeout gets the default transaction timeout in seconds
func (servant *TransactionServiceServant) GetDefaultTimeout() (uint32, error) {
	return servant.service.GetDefaultTimeout(), nil
}

// TransactionServiceClient implements a client for the Transaction Service
type TransactionServiceClient struct {
	objRef *ObjectRef
}

// NewTransactionServiceClient creates a new Transaction Service client
func NewTransactionServiceClient(objRef *ObjectRef) *TransactionServiceClient {
	return &TransactionServiceClient{
		objRef: objRef,
	}
}

// GetFactory returns a factory client for the transaction service
func (client *TransactionServiceClient) GetFactory() (*TransactionFactoryClient, error) {
	// Create a request to get the factory
	request := NewRequest(client.objRef, "GetFactory")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction factory: %w", err)
	}

	// Extract the factory object reference
	factoryRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction factory")
	}

	return NewTransactionFactoryClient(factoryRef), nil
}

// GetCurrent returns a Current client for the transaction service
func (client *TransactionServiceClient) GetCurrent() (*TransactionCurrentClient, error) {
	// Create a request to get the current
	request := NewRequest(client.objRef, "GetCurrent")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction current: %w", err)
	}

	// Extract the current object reference
	currentRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction current")
	}

	return NewTransactionCurrentClient(currentRef), nil
}

// SetDefaultTimeout sets the default transaction timeout
func (client *TransactionServiceClient) SetDefaultTimeout(seconds uint32) error {
	// Create a request to set the default timeout
	request := NewRequest(client.objRef, "SetDefaultTimeout")

	// Add the timeout parameter
	request.AddParameter("seconds", seconds, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to set default timeout: %w", err)
	}

	return nil
}

// GetDefaultTimeout gets the default transaction timeout
func (client *TransactionServiceClient) GetDefaultTimeout() (uint32, error) {
	// Create a request to get the default timeout
	request := NewRequest(client.objRef, "GetDefaultTimeout")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return 0, fmt.Errorf("failed to get default timeout: %w", err)
	}

	// Extract the timeout value
	timeout, ok := response.(uint32)
	if !ok {
		return 0, fmt.Errorf("invalid response type for default timeout")
	}

	return timeout, nil
}

// TransactionFactoryClient implements a client for the Transaction Factory
type TransactionFactoryClient struct {
	objRef *ObjectRef
}

// NewTransactionFactoryClient creates a new Transaction Factory client
func NewTransactionFactoryClient(objRef *ObjectRef) *TransactionFactoryClient {
	return &TransactionFactoryClient{
		objRef: objRef,
	}
}

// Create creates a new transaction with the default timeout
func (client *TransactionFactoryClient) Create() (Control, error) {
	// Create a request to create a transaction
	request := NewRequest(client.objRef, "Create")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Extract the control object reference
	controlRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction control")
	}

	return NewControlClient(controlRef), nil
}

// CreateWithTimeout creates a new transaction with the specified timeout
func (client *TransactionFactoryClient) CreateWithTimeout(seconds uint32) (Control, error) {
	// Create a request to create a transaction with timeout
	request := NewRequest(client.objRef, "CreateWithTimeout")

	// Add the timeout parameter
	request.AddParameter("seconds", seconds, FlagIn)

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction with timeout: %w", err)
	}

	// Extract the control object reference
	controlRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction control")
	}

	return NewControlClient(controlRef), nil
}

// TransactionCurrentClient implements a client for the Transaction Current
type TransactionCurrentClient struct {
	objRef *ObjectRef
}

// NewTransactionCurrentClient creates a new Transaction Current client
func NewTransactionCurrentClient(objRef *ObjectRef) *TransactionCurrentClient {
	return &TransactionCurrentClient{
		objRef: objRef,
	}
}

// Begin starts a new transaction
func (client *TransactionCurrentClient) Begin() error {
	// Create a request to begin a transaction
	request := NewRequest(client.objRef, "Begin")

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	return nil
}

// BeginWithTimeout starts a new transaction with a specific timeout
func (client *TransactionCurrentClient) BeginWithTimeout(seconds uint32) error {
	// Create a request to begin a transaction with timeout
	request := NewRequest(client.objRef, "BeginWithTimeout")

	// Add the timeout parameter
	request.AddParameter("seconds", seconds, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to begin transaction with timeout: %w", err)
	}

	return nil
}

// Commit commits the current transaction
func (client *TransactionCurrentClient) Commit(reportHeuristics bool) error {
	// Create a request to commit the transaction
	request := NewRequest(client.objRef, "Commit")

	// Add the reportHeuristics parameter
	request.AddParameter("reportHeuristics", reportHeuristics, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the current transaction
func (client *TransactionCurrentClient) Rollback() error {
	// Create a request to rollback the transaction
	request := NewRequest(client.objRef, "Rollback")

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// SetRollbackOnly marks the current transaction for rollback only
func (client *TransactionCurrentClient) SetRollbackOnly() error {
	// Create a request to set rollback only
	request := NewRequest(client.objRef, "SetRollbackOnly")

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to set rollback only: %w", err)
	}

	return nil
}

// GetStatus returns the status of the current transaction
func (client *TransactionCurrentClient) GetStatus() (int, error) {
	// Create a request to get the transaction status
	request := NewRequest(client.objRef, "GetStatus")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return StatusUnknown, fmt.Errorf("failed to get transaction status: %w", err)
	}

	// Extract the status value
	status, ok := response.(int)
	if !ok {
		return StatusUnknown, fmt.Errorf("invalid response type for transaction status")
	}

	return status, nil
}

// GetControl returns the Control for the current transaction
func (client *TransactionCurrentClient) GetControl() (Control, error) {
	// Create a request to get the transaction control
	request := NewRequest(client.objRef, "GetControl")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction control: %w", err)
	}

	// Extract the control object reference
	controlRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction control")
	}

	return NewControlClient(controlRef), nil
}

// Suspend suspends the current transaction
func (client *TransactionCurrentClient) Suspend() (Control, error) {
	// Create a request to suspend the transaction
	request := NewRequest(client.objRef, "Suspend")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to suspend transaction: %w", err)
	}

	// Extract the control object reference
	controlRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for transaction control")
	}

	return NewControlClient(controlRef), nil
}

// Resume resumes a suspended transaction
func (client *TransactionCurrentClient) Resume(control Control) error {
	// Create a request to resume the transaction
	request := NewRequest(client.objRef, "Resume")

	// We need to convert the Control to an ObjectRef
	controlClient, ok := control.(*ControlClient)
	if !ok {
		return fmt.Errorf("invalid control type for resume")
	}

	// Add the control parameter
	request.AddParameter("control", controlClient.objRef, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to resume transaction: %w", err)
	}

	return nil
}

// SetTimeout sets the transaction timeout
func (client *TransactionCurrentClient) SetTimeout(seconds uint32) error {
	// Create a request to set the timeout
	request := NewRequest(client.objRef, "SetTimeout")

	// Add the seconds parameter
	request.AddParameter("seconds", seconds, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to set timeout: %w", err)
	}

	return nil
}

// ControlClient implements a client for the Transaction Control
type ControlClient struct {
	objRef *ObjectRef
}

// NewControlClient creates a new Transaction Control client
func NewControlClient(objRef *ObjectRef) *ControlClient {
	return &ControlClient{
		objRef: objRef,
	}
}

// GetCoordinator returns the coordinator for this transaction
func (client *ControlClient) GetCoordinator() (Coordinator, error) {
	// Create a request to get the coordinator
	request := NewRequest(client.objRef, "GetCoordinator")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinator: %w", err)
	}

	// Extract the coordinator object reference
	coordRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for coordinator")
	}

	return NewCoordinatorClient(coordRef), nil
}

// GetTerminator returns the terminator for this transaction
func (client *ControlClient) GetTerminator() (Terminator, error) {
	// Create a request to get the terminator
	request := NewRequest(client.objRef, "GetTerminator")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to get terminator: %w", err)
	}

	// Extract the terminator object reference
	termRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for terminator")
	}

	return NewTerminatorClient(termRef), nil
}

// CoordinatorClient implements a client for the Transaction Coordinator
type CoordinatorClient struct {
	objRef *ObjectRef
}

// NewCoordinatorClient creates a new Transaction Coordinator client
func NewCoordinatorClient(objRef *ObjectRef) *CoordinatorClient {
	return &CoordinatorClient{
		objRef: objRef,
	}
}

// RegisterResource registers a resource as a participant in the transaction
func (client *CoordinatorClient) RegisterResource(resource Resource) (RecoveryCoordinator, error) {
	// Create a request to register a resource
	request := NewRequest(client.objRef, "RegisterResource")

	// The resource needs to be exported as an object reference
	// This would typically be done by the ORB, but for this implementation
	// we'll assume the resource implements ExportableResource
	exportableResource, ok := resource.(ExportableResource)
	if !ok {
		return nil, fmt.Errorf("resource does not implement ExportableResource")
	}

	resourceRef := exportableResource.ExportReference()

	// Add the resource parameter
	request.AddParameter("resource", resourceRef, FlagIn)

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to register resource: %w", err)
	}

	// Extract the recovery coordinator object reference
	recoveryRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for recovery coordinator")
	}

	return NewRecoveryCoordinatorClient(recoveryRef), nil
}

// RegisterSynchronization registers a synchronization callback
func (client *CoordinatorClient) RegisterSynchronization(sync Synchronization) error {
	// Create a request to register a synchronization
	request := NewRequest(client.objRef, "RegisterSynchronization")

	// The synchronization needs to be exported as an object reference
	// This would typically be done by the ORB, but for this implementation
	// we'll assume the synchronization implements ExportableSynchronization
	exportableSync, ok := sync.(ExportableSynchronization)
	if !ok {
		return fmt.Errorf("synchronization does not implement ExportableSynchronization")
	}

	syncRef := exportableSync.ExportReference()

	// Add the synchronization parameter
	request.AddParameter("sync", syncRef, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to register synchronization: %w", err)
	}

	return nil
}

// GetStatus returns the transaction status
func (client *CoordinatorClient) GetStatus() (int, error) {
	// Create a request to get the status
	request := NewRequest(client.objRef, "GetStatus")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return StatusUnknown, fmt.Errorf("failed to get status: %w", err)
	}

	// Extract the status value
	status, ok := response.(int)
	if !ok {
		return StatusUnknown, fmt.Errorf("invalid response type for status")
	}

	return status, nil
}

// GetTransactionName returns the name of the transaction
func (client *CoordinatorClient) GetTransactionName() string {
	// Create a request to get the transaction name
	request := NewRequest(client.objRef, "GetTransactionName")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return ""
	}

	// Extract the name value
	name, ok := response.(string)
	if !ok {
		return ""
	}

	return name
}

// IsRelatedTransaction checks if the transaction is related to the given transaction
func (client *CoordinatorClient) IsRelatedTransaction(xid TransactionID) bool {
	// Create a request to check related transaction
	request := NewRequest(client.objRef, "IsRelatedTransaction")

	// Add the xid parameter
	request.AddParameter("xid", string(xid), FlagIn)

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return false
	}

	// Extract the result value
	result, ok := response.(bool)
	if !ok {
		return false
	}

	return result
}

// IsSameTransaction checks if the transaction has the same id
func (client *CoordinatorClient) IsSameTransaction(xid TransactionID) bool {
	// Create a request to check same transaction
	request := NewRequest(client.objRef, "IsSameTransaction")

	// Add the xid parameter
	request.AddParameter("xid", string(xid), FlagIn)

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return false
	}

	// Extract the result value
	result, ok := response.(bool)
	if !ok {
		return false
	}

	return result
}

// GetTransactionID returns the transaction ID
func (client *CoordinatorClient) GetTransactionID() TransactionID {
	// Create a request to get the transaction ID
	request := NewRequest(client.objRef, "GetTransactionID")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return ""
	}

	// Extract the ID value
	id, ok := response.(string)
	if !ok {
		return ""
	}

	return TransactionID(id)
}

// CreateSubtransaction creates a nested transaction
func (client *CoordinatorClient) CreateSubtransaction() (Control, error) {
	// Create a request to create a subtransaction
	request := NewRequest(client.objRef, "CreateSubtransaction")

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return nil, fmt.Errorf("failed to create subtransaction: %w", err)
	}

	// Extract the control object reference
	controlRef, ok := response.(*ObjectRef)
	if !ok {
		return nil, fmt.Errorf("invalid response type for control")
	}

	return NewControlClient(controlRef), nil
}

// TerminatorClient implements a client for the Transaction Terminator
type TerminatorClient struct {
	objRef *ObjectRef
}

// NewTerminatorClient creates a new Transaction Terminator client
func NewTerminatorClient(objRef *ObjectRef) *TerminatorClient {
	return &TerminatorClient{
		objRef: objRef,
	}
}

// Commit commits the transaction
func (client *TerminatorClient) Commit(reportHeuristics bool) error {
	// Create a request to commit the transaction
	request := NewRequest(client.objRef, "Commit")

	// Add the reportHeuristics parameter
	request.AddParameter("reportHeuristics", reportHeuristics, FlagIn)

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction
func (client *TerminatorClient) Rollback() error {
	// Create a request to rollback the transaction
	request := NewRequest(client.objRef, "Rollback")

	// Invoke the request
	_, err := request.Invoke()
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	return nil
}

// RecoveryCoordinatorClient implements a client for the Recovery Coordinator
type RecoveryCoordinatorClient struct {
	objRef *ObjectRef
}

// NewRecoveryCoordinatorClient creates a new Recovery Coordinator client
func NewRecoveryCoordinatorClient(objRef *ObjectRef) *RecoveryCoordinatorClient {
	return &RecoveryCoordinatorClient{
		objRef: objRef,
	}
}

// ReplayCompletion replays the completion process for a resource
func (client *RecoveryCoordinatorClient) ReplayCompletion(resource Resource) (int, error) {
	// Create a request to replay completion
	request := NewRequest(client.objRef, "ReplayCompletion")

	// The resource needs to be exported as an object reference
	// This would typically be done by the ORB, but for this implementation
	// we'll assume the resource implements ExportableResource
	exportableResource, ok := resource.(ExportableResource)
	if !ok {
		return StatusUnknown, fmt.Errorf("resource does not implement ExportableResource")
	}

	resourceRef := exportableResource.ExportReference()

	// Add the resource parameter
	request.AddParameter("resource", resourceRef, FlagIn)

	// Invoke the request
	response, err := request.Invoke()
	if err != nil {
		return StatusUnknown, fmt.Errorf("failed to replay completion: %w", err)
	}

	// Extract the status value
	status, ok := response.(int)
	if !ok {
		return StatusUnknown, fmt.Errorf("invalid response type for status")
	}

	return status, nil
}

// Interface for resources that can be exported as object references
type ExportableResource interface {
	Resource
	ExportReference() *ObjectRef
}

// Interface for synchronizations that can be exported as object references
type ExportableSynchronization interface {
	Synchronization
	ExportReference() *ObjectRef
}

// Helper functions

// generateTransactionID generates a unique transaction ID
func generateTransactionID() (TransactionID, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return TransactionID(fmt.Sprintf("TXN:%s", hex.EncodeToString(b))), nil
}

// handleTransactionTimeout handles a transaction timeout
func handleTransactionTimeout(xid TransactionID) {
	transactionMutex.RLock()
	tx, exists := activeTransactions[xid]
	transactionMutex.RUnlock()

	if !exists {
		// Transaction no longer exists
		return
	}

	// Check if the transaction is still active
	status := tx.getStatus()
	if status == StatusActive || status == StatusMarkedRollback {
		// Mark the transaction for rollback
		tx.setStatus(StatusMarkedRollback)

		// Get the terminator
		terminator := tx.terminator

		// Roll back the transaction
		go terminator.Rollback()
	}
}
