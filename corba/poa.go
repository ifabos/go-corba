// Package corba provides CORBA functionality for Go
package corba

import (
	"fmt"
	"sync"
	"time"
)

// POAPolicyID represents a POA policy ID
type POAPolicyID int

// POA Policy IDs
const (
	ThreadPolicyID             POAPolicyID = 16
	LifespanPolicyID           POAPolicyID = 17
	IdUniquenessPolicyID       POAPolicyID = 18
	IdAssignmentPolicyID       POAPolicyID = 19
	ImplicitActivationPolicyID POAPolicyID = 20
	ServantRetentionPolicyID   POAPolicyID = 21
	RequestProcessingPolicyID  POAPolicyID = 22
)

// ThreadPolicy values
const (
	SingleThreadModel     = 0
	ORBControlledModel    = 1
	ThreadPoolModel       = 2 // Not in CORBA spec, but useful
	ThreadPerRequestModel = 3 // Not in CORBA spec, but useful
)

// LifespanPolicy values
const (
	TransientLifespan  = 0
	PersistentLifespan = 1
)

// IdUniquenessPolicy values
const (
	UniqueID   = 0
	MultipleID = 1
)

// IdAssignmentPolicy values
const (
	UserAssignedID   = 0
	SystemAssignedID = 1
)

// ImplicitActivationPolicy values
const (
	ImplicitActivationDisabled = 0
	ImplicitActivationEnabled  = 1
)

// ServantRetentionPolicy values
const (
	RetainServants    = 0
	NonRetainServants = 1
)

// RequestProcessingPolicy values
const (
	UseActiveObjectMapOnly = 0
	UseDefaultServant      = 1
	UseServantManager      = 2
)

// POAPolicy represents a policy for a POA
type POAPolicy interface {
	ID() POAPolicyID
	Value() interface{}
}

// Policy implementations
type policyImpl struct {
	policyID POAPolicyID
	value    interface{}
}

func (p *policyImpl) ID() POAPolicyID {
	return p.policyID
}

func (p *policyImpl) Value() interface{} {
	return p.value
}

// Policy factory functions
func NewThreadPolicy(value int) POAPolicy {
	return &policyImpl{policyID: ThreadPolicyID, value: value}
}

func NewLifespanPolicy(value int) POAPolicy {
	return &policyImpl{policyID: LifespanPolicyID, value: value}
}

func NewIdUniquenessPolicy(value int) POAPolicy {
	return &policyImpl{policyID: IdUniquenessPolicyID, value: value}
}

func NewIdAssignmentPolicy(value int) POAPolicy {
	return &policyImpl{policyID: IdAssignmentPolicyID, value: value}
}

func NewImplicitActivationPolicy(value int) POAPolicy {
	return &policyImpl{policyID: ImplicitActivationPolicyID, value: value}
}

func NewServantRetentionPolicy(value int) POAPolicy {
	return &policyImpl{policyID: ServantRetentionPolicyID, value: value}
}

func NewRequestProcessingPolicy(value int) POAPolicy {
	return &policyImpl{policyID: RequestProcessingPolicyID, value: value}
}

// POA interface errors
var (
	ErrAdapterAlreadyExists = fmt.Errorf("adapter already exists")
	ErrAdapterNonExistent   = fmt.Errorf("adapter non-existent")
	ErrInvalidPolicy        = fmt.Errorf("invalid policy")
	ErrNoServant            = fmt.Errorf("no servant")
	ErrObjectNotActive      = fmt.Errorf("object not active")
	ErrServantAlreadyActive = fmt.Errorf("servant already active")
	ErrWrongAdapter         = fmt.Errorf("wrong adapter")
	ErrWrongPolicy          = fmt.Errorf("wrong policy")
	ErrObjectAlreadyActive  = fmt.Errorf("object already active")
	ErrInvalidObjectID      = fmt.Errorf("invalid object id")
)

// ObjectID represents an object identifier in a POA
type ObjectID []byte

// ServantManager interface for POA servant management
type ServantManager interface {
	// ServantLocator or ServantActivator functionality
}

// ServantActivator interface for incarnating/etherealizing servants
type ServantActivator interface {
	ServantManager
	Incarnate(objectID ObjectID, adapter *POA) (interface{}, error)
	Etherealize(objectID ObjectID, adapter *POA, servant interface{}, cleanup bool) error
}

// ServantLocator interface for finding servants per-request
type ServantLocator interface {
	ServantManager
	Preinvoke(objectID ObjectID, adapter *POA, operation string) (interface{}, interface{}, error)
	Postinvoke(objectID ObjectID, adapter *POA, operation string, servant interface{}, cookieVal interface{}) error
}

// POA represents a Portable Object Adapter
type POA struct {
	name            string
	parent          *POA
	orb             *ORB
	children        map[string]*POA
	policies        map[POAPolicyID]POAPolicy
	defaultServant  interface{}
	servantManager  ServantManager
	objectMap       map[string]interface{}   // Maps object ID (string) to servant
	oidToServantMap map[string]interface{}   // For efficient lookup
	servantToOidMap map[interface{}][]string // For efficient lookup (multiple OIDs per servant if MultipleID policy)
	mutex           sync.RWMutex
	isActive        bool

	// Cached policy values for quick access
	threadModel        int
	lifespan           int
	uniqueID           int
	idAssignment       int
	implicitActivation int
	servantRetention   int
	requestProcessing  int
}

// NewRootPOA creates a new root POA with default policies
func (o *ORB) NewRootPOA() *POA {
	policies := map[POAPolicyID]POAPolicy{
		ThreadPolicyID:             NewThreadPolicy(ORBControlledModel),
		LifespanPolicyID:           NewLifespanPolicy(TransientLifespan),
		IdUniquenessPolicyID:       NewIdUniquenessPolicy(UniqueID),
		IdAssignmentPolicyID:       NewIdAssignmentPolicy(SystemAssignedID),
		ImplicitActivationPolicyID: NewImplicitActivationPolicy(ImplicitActivationEnabled),
		ServantRetentionPolicyID:   NewServantRetentionPolicy(RetainServants),
		RequestProcessingPolicyID:  NewRequestProcessingPolicy(UseActiveObjectMapOnly),
	}

	poa := &POA{
		name:            "RootPOA",
		parent:          nil,
		orb:             o,
		children:        make(map[string]*POA),
		policies:        policies,
		objectMap:       make(map[string]interface{}),
		oidToServantMap: make(map[string]interface{}),
		servantToOidMap: make(map[interface{}][]string),
		isActive:        true,

		// Cache policy values
		threadModel:        ORBControlledModel,
		lifespan:           TransientLifespan,
		uniqueID:           UniqueID,
		idAssignment:       SystemAssignedID,
		implicitActivation: ImplicitActivationEnabled,
		servantRetention:   RetainServants,
		requestProcessing:  UseActiveObjectMapOnly,
	}

	// Register the POA manager
	o.poaManagers = append(o.poaManagers, &POAManager{
		state: POAManagerActive,
		poas:  []*POA{poa},
	})

	return poa
}

// GetPolicy returns a policy for the POA
func (p *POA) GetPolicy(policyID POAPolicyID) (POAPolicy, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if policy, ok := p.policies[policyID]; ok {
		return policy, nil
	}
	return nil, ErrInvalidPolicy
}

// CreatePOA creates a new child POA with the given name and policies
func (p *POA) CreatePOA(name string, manager *POAManager, policies []POAPolicy) (*POA, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if a child with the same name already exists
	if _, exists := p.children[name]; exists {
		return nil, ErrAdapterAlreadyExists
	}

	// Start with default policies from parent
	childPolicies := make(map[POAPolicyID]POAPolicy)
	for id, policy := range p.policies {
		childPolicies[id] = policy
	}

	// Override with provided policies
	for _, policy := range policies {
		childPolicies[policy.ID()] = policy
	}

	// Create the child POA
	child := &POA{
		name:            name,
		parent:          p,
		orb:             p.orb,
		children:        make(map[string]*POA),
		policies:        childPolicies,
		objectMap:       make(map[string]interface{}),
		oidToServantMap: make(map[string]interface{}),
		servantToOidMap: make(map[interface{}][]string),
		isActive:        true,
	}

	// Cache policy values
	child.setCachedPolicyValues()

	// Add to parent's children list
	p.children[name] = child

	// Add to POA manager
	if manager != nil {
		manager.addPOA(child)
	} else {
		// Use parent's manager by default
		for _, mgr := range p.orb.poaManagers {
			if containsPOA(mgr.poas, p) {
				mgr.addPOA(child)
				break
			}
		}
	}

	return child, nil
}

// setCachedPolicyValues caches policy values for quick access
func (p *POA) setCachedPolicyValues() {
	if policy, ok := p.policies[ThreadPolicyID]; ok {
		p.threadModel = policy.Value().(int)
	}
	if policy, ok := p.policies[LifespanPolicyID]; ok {
		p.lifespan = policy.Value().(int)
	}
	if policy, ok := p.policies[IdUniquenessPolicyID]; ok {
		p.uniqueID = policy.Value().(int)
	}
	if policy, ok := p.policies[IdAssignmentPolicyID]; ok {
		p.idAssignment = policy.Value().(int)
	}
	if policy, ok := p.policies[ImplicitActivationPolicyID]; ok {
		p.implicitActivation = policy.Value().(int)
	}
	if policy, ok := p.policies[ServantRetentionPolicyID]; ok {
		p.servantRetention = policy.Value().(int)
	}
	if policy, ok := p.policies[RequestProcessingPolicyID]; ok {
		p.requestProcessing = policy.Value().(int)
	}
}

// Helper function to check if a POA slice contains a specific POA
func containsPOA(poas []*POA, poa *POA) bool {
	for _, p := range poas {
		if p == poa {
			return true
		}
	}
	return false
}

// FindPOA finds a child POA by name
func (p *POA) FindPOA(name string, activate bool) (*POA, error) {
	p.mutex.RLock()
	child, exists := p.children[name]
	p.mutex.RUnlock()

	if !exists {
		return nil, ErrAdapterNonExistent
	}

	if activate && !child.isActive {
		child.Activate()
	}

	return child, nil
}

// Destroy destroys this POA and all its children
func (p *POA) Destroy(etherializeObjects bool, waitForCompletion bool) error {
	p.mutex.Lock()

	// First deactivate the POA
	p.isActive = false

	// Destroy all children first
	for _, child := range p.children {
		child.Destroy(etherializeObjects, waitForCompletion)
	}

	// Clear children map
	p.children = make(map[string]*POA)

	if etherializeObjects {
		// Etherealize all active objects
		if activator, ok := p.servantManager.(ServantActivator); ok && p.servantRetention == NonRetainServants {
			for objectID, servant := range p.objectMap {
				// Convert string objectID back to []byte for ServantActivator
				oid := []byte(objectID)
				activator.Etherealize(oid, p, servant, true)
			}
		}
	}

	// Clear maps
	p.objectMap = make(map[string]interface{})
	p.oidToServantMap = make(map[string]interface{})
	p.servantToOidMap = make(map[interface{}][]string)

	// Remove from manager
	for _, mgr := range p.orb.poaManagers {
		mgr.removePOA(p)
	}

	// If parent exists, remove from parent's children
	if p.parent != nil {
		delete(p.parent.children, p.name)
	}

	p.mutex.Unlock()

	if waitForCompletion {
		// In a real implementation, we would wait for all outstanding requests to complete
		// For now, we'll just simulate a small delay
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// Activate activates the POA
func (p *POA) Activate() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.isActive = true
}

// Deactivate deactivates the POA
func (p *POA) Deactivate(etherializeObjects bool, waitForCompletion bool) error {
	p.mutex.Lock()
	p.isActive = false
	p.mutex.Unlock()

	if etherializeObjects {
		// Etherealize all active objects
		if activator, ok := p.servantManager.(ServantActivator); ok && p.servantRetention == NonRetainServants {
			p.mutex.RLock()
			for objectID, servant := range p.objectMap {
				oid := []byte(objectID)
				activator.Etherealize(oid, p, servant, true)
			}
			p.mutex.RUnlock()
		}
	}

	if waitForCompletion {
		// In a real implementation, we would wait for all outstanding requests to complete
		// For now, we'll just simulate a small delay
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// ServantToID gets the ObjectID associated with a servant
func (p *POA) ServantToID(servant interface{}) (ObjectID, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Check if servant is registered
	oids, exists := p.servantToOidMap[servant]
	if !exists || len(oids) == 0 {
		// Check if implicit activation is enabled
		if p.implicitActivation == ImplicitActivationEnabled {
			// Generate a new ObjectID
			return p.activateObject(servant)
		}
		return nil, ErrNoServant
	}

	// Return first OID (most common case)
	return []byte(oids[0]), nil
}

// IDToServant gets the servant associated with an ObjectID
func (p *POA) IDToServant(id ObjectID) (interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Convert ObjectID to string for map lookup
	oidStr := string(id)

	// Check active object map first
	if p.servantRetention == RetainServants {
		if servant, exists := p.oidToServantMap[oidStr]; exists {
			return servant, nil
		}
	}

	// If we have a default servant and the policy allows it
	if p.requestProcessing == UseDefaultServant && p.defaultServant != nil {
		return p.defaultServant, nil
	}

	// If we have a servant manager and policy allows it
	if p.requestProcessing == UseServantManager && p.servantManager != nil {
		// ServantActivator case
		if p.servantRetention == RetainServants {
			if activator, ok := p.servantManager.(ServantActivator); ok {
				servant, err := activator.Incarnate(id, p)
				if err != nil {
					return nil, err
				}

				// Add to object map
				p.mutex.RUnlock()
				p.mutex.Lock()
				p.objectMap[oidStr] = servant
				p.oidToServantMap[oidStr] = servant

				// Add to servant-to-OID map based on uniqueness policy
				if p.uniqueID == UniqueID {
					p.servantToOidMap[servant] = []string{oidStr}
				} else {
					p.servantToOidMap[servant] = append(p.servantToOidMap[servant], oidStr)
				}
				p.mutex.Unlock()
				p.mutex.RLock()

				return servant, nil
			}
		}
	}

	return nil, ErrObjectNotActive
}

// ActivateObject activates an object with a system-generated ID
func (p *POA) ActivateObject(servant interface{}) (ObjectID, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.activateObject(servant)
}

// internal helper to activate an object
func (p *POA) activateObject(servant interface{}) (ObjectID, error) {
	// Check servant uniqueness policy
	if p.uniqueID == UniqueID {
		// Check if servant is already active
		if _, exists := p.servantToOidMap[servant]; exists {
			return nil, ErrServantAlreadyActive
		}
	}

	// Generate a new ObjectID
	var objectID ObjectID
	if p.idAssignment == SystemAssignedID {
		// Generate a system ID (UUID-like)
		objectID = generateObjectID()
	} else {
		return nil, ErrWrongPolicy
	}

	oidStr := string(objectID)

	// Check if this ObjectID is already in use
	if _, exists := p.oidToServantMap[oidStr]; exists {
		// Very unlikely with a proper UUID generation, but let's handle it
		return nil, ErrObjectAlreadyActive
	}

	// Store in object maps
	p.objectMap[oidStr] = servant
	p.oidToServantMap[oidStr] = servant

	// Add to servant-to-OID map based on uniqueness policy
	if p.uniqueID == UniqueID {
		p.servantToOidMap[servant] = []string{oidStr}
	} else {
		p.servantToOidMap[servant] = append(p.servantToOidMap[servant], oidStr)
	}

	return objectID, nil
}

// ActivateObjectWithID activates an object with a user-provided ID
func (p *POA) ActivateObjectWithID(id ObjectID, servant interface{}) error {
	if len(id) == 0 {
		return ErrInvalidObjectID
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if this ObjectID is already in use
	oidStr := string(id)
	if _, exists := p.oidToServantMap[oidStr]; exists {
		return ErrObjectAlreadyActive
	}

	// Check servant uniqueness policy
	if p.uniqueID == UniqueID {
		// Check if servant is already active
		if _, exists := p.servantToOidMap[servant]; exists {
			return ErrServantAlreadyActive
		}
	}

	// Store in object maps
	p.objectMap[oidStr] = servant
	p.oidToServantMap[oidStr] = servant

	// Add to servant-to-OID map based on uniqueness policy
	if p.uniqueID == UniqueID {
		p.servantToOidMap[servant] = []string{oidStr}
	} else {
		p.servantToOidMap[servant] = append(p.servantToOidMap[servant], oidStr)
	}

	return nil
}

// DeactivateObject deactivates an object with the given ObjectID
func (p *POA) DeactivateObject(id ObjectID) error {
	if len(id) == 0 {
		return ErrInvalidObjectID
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	oidStr := string(id)

	// Check if the object is active
	servant, exists := p.oidToServantMap[oidStr]
	if !exists {
		return ErrObjectNotActive
	}

	// Call etherealize if a servant activator is registered
	if p.servantManager != nil {
		if activator, ok := p.servantManager.(ServantActivator); ok {
			activator.Etherealize(id, p, servant, true)
		}
	}

	// Remove from object maps
	delete(p.objectMap, oidStr)
	delete(p.oidToServantMap, oidStr)

	// Update servant-to-OID map
	oids := p.servantToOidMap[servant]
	if len(oids) == 1 {
		delete(p.servantToOidMap, servant)
	} else {
		// Remove the specific OID
		newOids := make([]string, 0, len(oids)-1)
		for _, oid := range oids {
			if oid != oidStr {
				newOids = append(newOids, oid)
			}
		}
		p.servantToOidMap[servant] = newOids
	}

	return nil
}

// SetServantManager sets the servant manager for this POA
func (p *POA) SetServantManager(manager ServantManager) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.servantRetention == RetainServants && !isActivator(manager) {
		return ErrWrongPolicy
	} else if p.servantRetention == NonRetainServants && !isLocator(manager) {
		return ErrWrongPolicy
	}

	p.servantManager = manager
	return nil
}

// Helper to check if a ServantManager is a ServantActivator
func isActivator(manager ServantManager) bool {
	_, ok := manager.(ServantActivator)
	return ok
}

// Helper to check if a ServantManager is a ServantLocator
func isLocator(manager ServantManager) bool {
	_, ok := manager.(ServantLocator)
	return ok
}

// GetServantManager gets the servant manager for this POA
func (p *POA) GetServantManager() (ServantManager, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.servantManager == nil {
		return nil, ErrNoServant
	}

	return p.servantManager, nil
}

// SetDefaultServant sets the default servant for this POA
func (p *POA) SetDefaultServant(servant interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.requestProcessing != UseDefaultServant {
		return ErrWrongPolicy
	}

	p.defaultServant = servant
	return nil
}

// GetDefaultServant gets the default servant for this POA
func (p *POA) GetDefaultServant() (interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.defaultServant == nil {
		return nil, ErrNoServant
	}

	return p.defaultServant, nil
}

// ObjectIDToString converts an ObjectID to a string
func (p *POA) ObjectIDToString(id ObjectID) string {
	return string(id)
}

// StringToObjectID converts a string to an ObjectID
func (p *POA) StringToObjectID(s string) ObjectID {
	return ObjectID(s)
}

// generateObjectID generates a new ObjectID
func generateObjectID() ObjectID {
	// In a real implementation, this would be more sophisticated (UUID)
	// For now, just use the current time as a string
	return ObjectID(fmt.Sprintf("OBJ_%d", time.Now().UnixNano()))
}

// POAManager states
const (
	POAManagerHolding    = 0
	POAManagerActive     = 1
	POAManagerDiscarding = 2
	POAManagerInactive   = 3
)

// POAManager manages the state of one or more POAs
type POAManager struct {
	state int
	poas  []*POA
	mutex sync.RWMutex
}

// NewPOAManager creates a new POA manager
func (o *ORB) NewPOAManager() *POAManager {
	manager := &POAManager{
		state: POAManagerHolding,
		poas:  make([]*POA, 0),
	}

	o.poaManagers = append(o.poaManagers, manager)
	return manager
}

// addPOA adds a POA to this manager
func (m *POAManager) addPOA(poa *POA) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.poas = append(m.poas, poa)
}

// removePOA removes a POA from this manager
func (m *POAManager) removePOA(poa *POA) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var newPoas []*POA
	for _, p := range m.poas {
		if p != poa {
			newPoas = append(newPoas, p)
		}
	}
	m.poas = newPoas
}

// Activate activates all POAs managed by this manager
func (m *POAManager) Activate() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state = POAManagerActive
	for _, poa := range m.poas {
		poa.Activate()
	}
}

// Hold puts all POAs managed by this manager in holding state
func (m *POAManager) Hold() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state = POAManagerHolding
}

// Discard puts all POAs managed by this manager in discarding state
func (m *POAManager) Discard() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state = POAManagerDiscarding
}

// Deactivate deactivates all POAs managed by this manager
func (m *POAManager) Deactivate(etherializeObjects bool, waitForCompletion bool) {
	m.mutex.Lock()
	m.state = POAManagerInactive
	poas := m.poas // Make a copy to avoid holding the lock during deactivation
	m.mutex.Unlock()

	for _, poa := range poas {
		poa.Deactivate(etherializeObjects, waitForCompletion)
	}
}

// GetState returns the current state of the POA manager
func (m *POAManager) GetState() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.state
}

// Implementation of a basic ServantActivator
type BasicServantActivator struct {
	IncarnateFunc   func(ObjectID, *POA) (interface{}, error)
	EtherealizeFUnc func(ObjectID, *POA, interface{}, bool) error
}

func (a *BasicServantActivator) Incarnate(objectID ObjectID, adapter *POA) (interface{}, error) {
	if a.IncarnateFunc != nil {
		return a.IncarnateFunc(objectID, adapter)
	}
	return nil, ErrNoServant
}

func (a *BasicServantActivator) Etherealize(objectID ObjectID, adapter *POA, servant interface{}, cleanup bool) error {
	if a.EtherealizeFUnc != nil {
		return a.EtherealizeFUnc(objectID, adapter, servant, cleanup)
	}
	return nil
}

// Implementation of a basic ServantLocator
type BasicServantLocator struct {
	PreinvokeFunc  func(ObjectID, *POA, string) (interface{}, interface{}, error)
	PostinvokeFunc func(ObjectID, *POA, string, interface{}, interface{}) error
}

func (l *BasicServantLocator) Preinvoke(objectID ObjectID, adapter *POA, operation string) (interface{}, interface{}, error) {
	if l.PreinvokeFunc != nil {
		return l.PreinvokeFunc(objectID, adapter, operation)
	}
	return nil, nil, ErrNoServant
}

func (l *BasicServantLocator) Postinvoke(objectID ObjectID, adapter *POA, operation string, servant interface{}, cookieVal interface{}) error {
	if l.PostinvokeFunc != nil {
		return l.PostinvokeFunc(objectID, adapter, operation, servant, cookieVal)
	}
	return nil
}
