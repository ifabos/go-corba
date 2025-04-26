// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"reflect"
	"sync"
)

// irObjectBase implements the common attributes and methods for all IR objects
type irObjectBase struct {
	id             string
	name           string
	container      IRContainer
	definitionKind DefinitionKind
}

func (obj *irObjectBase) Id() string {
	return obj.id
}

func (obj *irObjectBase) Name() string {
	return obj.name
}

func (obj *irObjectBase) Container() IRContainer {
	return obj.container
}

func (obj *irObjectBase) DefKind() DefinitionKind {
	return obj.definitionKind
}

func (obj *irObjectBase) Describe() string {
	return fmt.Sprintf("%s %s (ID: %s)", obj.definitionKind, obj.name, obj.id)
}

// containerBase implements the Container interface
type containerBase struct {
	irObjectBase
	mu       sync.RWMutex
	contents map[string]IRObject
}

func newContainerBase(id, name string, kind DefinitionKind, container IRContainer) *containerBase {
	return &containerBase{
		irObjectBase: irObjectBase{
			id:             id,
			name:           name,
			container:      container,
			definitionKind: kind,
		},
		contents: make(map[string]IRObject),
	}
}

func (c *containerBase) Contents(limit DefinitionKind) []IRObject {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := []IRObject{}
	for _, obj := range c.contents {
		if limit == DK_ALL || obj.DefKind() == limit {
			results = append(results, obj)
		}
	}
	return results
}

func (c *containerBase) Lookup(name string) (IRObject, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if obj, ok := c.contents[name]; ok {
		return obj, nil
	}
	return nil, ErrInterfaceNotFound
}

func (c *containerBase) LookupName(search_name string, levels int, limit DefinitionKind) []IRObject {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := []IRObject{}

	// Search in this container
	if obj, ok := c.contents[search_name]; ok {
		if limit == DK_ALL || obj.DefKind() == limit {
			results = append(results, obj)
		}
	}

	// Search in nested containers if levels > 0
	if levels != 0 {
		nextLevel := levels
		if levels > 0 {
			nextLevel--
		}

		for _, obj := range c.contents {
			if container, ok := obj.(IRContainer); ok {
				nestedResults := container.LookupName(search_name, nextLevel, limit)
				results = append(results, nestedResults...)
			}
		}
	}

	return results
}

func (c *containerBase) Add(obj IRObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.contents[obj.Name()]; exists {
		return ErrDuplicateDefinition
	}

	// If it's also a Contained object, set its container
	if contained, ok := obj.(Contained); ok {
		if contained, ok := contained.(*containedBase); ok {
			contained.container = c
		}
	}

	c.contents[obj.Name()] = obj
	return nil
}

// containedBase implements the Contained interface
type containedBase struct {
	irObjectBase
}

func newContainedBase(id, name string, kind DefinitionKind, container IRContainer) *containedBase {
	return &containedBase{
		irObjectBase: irObjectBase{
			id:             id,
			name:           name,
			container:      container,
			definitionKind: kind,
		},
	}
}

func (c *containedBase) Move(new_container IRContainer, new_name string) error {
	if new_container == nil {
		return fmt.Errorf("cannot move to nil container")
	}

	oldName := c.name
	oldContainer := c.container

	// Remove from old container if it exists
	if oldContainer != nil {
		if containerBase, ok := oldContainer.(*containerBase); ok {
			containerBase.mu.Lock()
			delete(containerBase.contents, c.name)
			containerBase.mu.Unlock()
		} else {
			// If it's not a containerBase, try a generic remove approach
			// This might not be thread-safe depending on the container implementation
			if containerObj, ok := oldContainer.(interface{ Remove(string) error }); ok {
				if err := containerObj.Remove(c.name); err != nil {
					return fmt.Errorf("failed to remove from old container: %w", err)
				}
			}
		}
	}

	// Update the object state
	c.container = new_container
	c.name = new_name

	// Add to new container
	err := new_container.Add(c)
	if err != nil {
		// Restore old state on error
		c.name = oldName
		c.container = oldContainer

		// Try to add back to old container if it existed
		if oldContainer != nil {
			if containerBase, ok := oldContainer.(*containerBase); ok {
				containerBase.mu.Lock()
				containerBase.contents[oldName] = c
				containerBase.mu.Unlock()
			}
		}

		return err
	}

	return nil
}

// repositoryImpl implements the Repository interface
type repositoryImpl struct {
	containerBase
}

// NewRepository creates a new Interface Repository
func NewRepository() Repository {
	repo := &repositoryImpl{
		containerBase: *newContainerBase("IDL:omg.org/CORBA/Repository:1.0", "InterfaceRepository", DK_REPOSITORY, nil),
	}

	// Initialize with primitive types
	repo.initializePrimitiveTypes()

	return repo
}

func (r *repositoryImpl) initializePrimitiveTypes() {
	// Add primitive types to the repository
	primitives := []struct {
		id   string
		name string
	}{
		{"IDL:omg.org/CORBA/Short:1.0", "short"},
		{"IDL:omg.org/CORBA/Long:1.0", "long"},
		{"IDL:omg.org/CORBA/UShort:1.0", "unsigned short"},
		{"IDL:omg.org/CORBA/ULong:1.0", "unsigned long"},
		{"IDL:omg.org/CORBA/Float:1.0", "float"},
		{"IDL:omg.org/CORBA/Double:1.0", "double"},
		{"IDL:omg.org/CORBA/Boolean:1.0", "boolean"},
		{"IDL:omg.org/CORBA/Char:1.0", "char"},
		{"IDL:omg.org/CORBA/Octet:1.0", "octet"},
		{"IDL:omg.org/CORBA/Any:1.0", "any"},
		{"IDL:omg.org/CORBA/TypeCode:1.0", "TypeCode"},
		{"IDL:omg.org/CORBA/Principal:1.0", "Principal"},
		{"IDL:omg.org/CORBA/Object:1.0", "Object"},
		{"IDL:omg.org/CORBA/String:1.0", "string"},
		{"IDL:omg.org/CORBA/WString:1.0", "wstring"},
		{"IDL:omg.org/CORBA/LongLong:1.0", "long long"},
		{"IDL:omg.org/CORBA/ULongLong:1.0", "unsigned long long"},
		{"IDL:omg.org/CORBA/LongDouble:1.0", "long double"},
	}

	for _, p := range primitives {
		tc := &primitiveTypeCode{
			typeCodeBase: typeCodeBase{
				id:   p.id,
				name: p.name,
				kind: DK_PRIMITIVE,
			},
		}
		r.Add(tc)
	}
}

func (r *repositoryImpl) LookupId(id string) (IRObject, error) {
	// Use a stack-based approach for more efficient traversal
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a lookup helper function that avoids redundant traversals
	var lookupInContainer func(IRContainer) IRObject
	visited := make(map[IRContainer]bool)

	lookupInContainer = func(container IRContainer) IRObject {
		// Avoid cycles
		if visited[container] {
			return nil
		}
		visited[container] = true

		// Check direct children
		for _, obj := range container.Contents(DK_ALL) {
			if obj.Id() == id {
				return obj
			}

			// Check nested containers
			if nestedContainer, ok := obj.(IRContainer); ok {
				if found := lookupInContainer(nestedContainer); found != nil {
					return found
				}
			}
		}

		return nil
	}

	// Start the search from repository
	if obj := lookupInContainer(r); obj != nil {
		return obj, nil
	}

	return nil, ErrInterfaceNotFound
}

func (r *repositoryImpl) CreateModule(id string, name string) (ModuleDef, error) {
	module := &moduleDefImpl{
		containerBase: *newContainerBase(id, name, DK_MODULE, r),
		containedBase: *newContainedBase(id, name, DK_MODULE, r),
	}

	if err := r.Add(module); err != nil {
		return nil, err
	}

	return module, nil
}

func (r *repositoryImpl) CreateInterface(id string, name string) (InterfaceDef, error) {
	iface := &interfaceDefImpl{
		containerBase: *newContainerBase(id, name, DK_INTERFACE, r),
		containedBase: *newContainedBase(id, name, DK_INTERFACE, r),
		bases:         []InterfaceDef{},
	}

	if err := r.Add(iface); err != nil {
		return nil, err
	}

	return iface, nil
}

func (r *repositoryImpl) CreateStruct(id string, name string) (StructDef, error) {
	str := &structDefImpl{
		containedBase: *newContainedBase(id, name, DK_STRUCT, r),
		typeCodeBase:  typeCodeBase{id: id, name: name, kind: DK_STRUCT},
		members:       []StructMember{},
	}

	if err := r.Add(str); err != nil {
		return nil, err
	}

	return str, nil
}

func (r *repositoryImpl) CreateException(id string, name string) (ExceptionDef, error) {
	except := &exceptionDefImpl{
		containedBase: *newContainedBase(id, name, DK_EXCEPTION, r),
		typeCodeBase:  typeCodeBase{id: id, name: name, kind: DK_EXCEPTION},
		members:       []StructMember{},
	}

	if err := r.Add(except); err != nil {
		return nil, err
	}

	return except, nil
}

func (r *repositoryImpl) CreateEnum(id string, name string) (EnumDef, error) {
	enum := &enumDefImpl{
		containedBase: *newContainedBase(id, name, DK_ENUM, r),
		typeCodeBase:  typeCodeBase{id: id, name: name, kind: DK_ENUM},
		members:       []string{},
	}

	if err := r.Add(enum); err != nil {
		return nil, err
	}

	return enum, nil
}

func (r *repositoryImpl) CreateUnion(id string, name string) (UnionDef, error) {
	union := &unionDefImpl{
		containedBase: *newContainedBase(id, name, DK_UNION, r),
		typeCodeBase:  typeCodeBase{id: id, name: name, kind: DK_UNION},
		members:       []UnionMember{},
		discriminator: nil, // Must be set later
	}

	if err := r.Add(union); err != nil {
		return nil, err
	}

	return union, nil
}

func (r *repositoryImpl) CreateAlias(id string, name string, original TypeCode) (AliasDef, error) {
	alias := &aliasDefImpl{
		containedBase: *newContainedBase(id, name, DK_ALIAS, r),
		typeCodeBase:  typeCodeBase{id: id, name: name, kind: DK_ALIAS},
		original:      original,
	}

	if err := r.Add(alias); err != nil {
		return nil, err
	}

	return alias, nil
}

func (r *repositoryImpl) Destroy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear all contents
	r.contents = make(map[string]IRObject)

	return nil
}

// moduleDefImpl implements the ModuleDef interface
type moduleDefImpl struct {
	containerBase
	containedBase
}

// Resolve ambiguity by explicitly defining the Container method
func (m *moduleDefImpl) Container() IRContainer {
	return m.containedBase.Container()
}

// Resolve ambiguity by explicitly defining the DefKind method
func (m *moduleDefImpl) DefKind() DefinitionKind {
	return m.containedBase.DefKind()
}

// Resolve ambiguity by explicitly defining the Describe method
func (m *moduleDefImpl) Describe() string {
	return m.containedBase.Describe()
}

// Resolve ambiguity by explicitly defining the Id method
func (m *moduleDefImpl) Id() string {
	return m.containedBase.Id()
}

// Resolve ambiguity by explicitly defining the Name method
func (m *moduleDefImpl) Name() string {
	return m.containedBase.Name()
}

// Ensure moduleDefImpl implements ModuleDef
var _ ModuleDef = &moduleDefImpl{}

// interfaceDefImpl implements the InterfaceDef interface
type interfaceDefImpl struct {
	containerBase
	containedBase
	bases []InterfaceDef
}

// Resolve ambiguity by explicitly defining the Container method
func (i *interfaceDefImpl) Container() IRContainer {
	return i.containedBase.Container()
}

// Resolve ambiguity by explicitly defining the DefKind method
func (i *interfaceDefImpl) DefKind() DefinitionKind {
	return i.containedBase.DefKind()
}

// Resolve ambiguity by explicitly defining the Describe method
func (i *interfaceDefImpl) Describe() string {
	return i.containedBase.Describe()
}

// Resolve ambiguity by explicitly defining the Id method
func (i *interfaceDefImpl) Id() string {
	return i.containedBase.Id()
}

// Resolve ambiguity by explicitly defining the Name method
func (i *interfaceDefImpl) Name() string {
	return i.containedBase.Name()
}

// Ensure interfaceDefImpl implements InterfaceDef
var _ InterfaceDef = &interfaceDefImpl{}

func (i *interfaceDefImpl) BaseInterfaces() []InterfaceDef {
	return i.bases
}

func (i *interfaceDefImpl) CreateAttribute(id string, name string, type_code TypeCode, mode int) (AttributeDef, error) {
	attr := &attributeDefImpl{
		containedBase: *newContainedBase(id, name, DK_ATTRIBUTE, i),
		typeCode:      type_code,
		mode:          mode,
	}

	if err := i.Add(attr); err != nil {
		return nil, err
	}

	return attr, nil
}

func (i *interfaceDefImpl) CreateOperation(id string, name string, result TypeCode, mode int) (OperationDef, error) {
	op := &operationDefImpl{
		containedBase: *newContainedBase(id, name, DK_OPERATION, i),
		resultType:    result,
		params:        []ParameterDescription{},
		exceptions:    []ExceptionDef{},
	}

	if err := i.Add(op); err != nil {
		return nil, err
	}

	return op, nil
}

// attributeDefImpl implements the AttributeDef interface
type attributeDefImpl struct {
	containedBase
	typeCode TypeCode
	mode     int
}

// Ensure attributeDefImpl implements AttributeDef
var _ AttributeDef = &attributeDefImpl{}

func (a *attributeDefImpl) Type() TypeCode {
	return a.typeCode
}

func (a *attributeDefImpl) Mode() int {
	return a.mode
}

// operationDefImpl implements the OperationDef interface
type operationDefImpl struct {
	containedBase
	resultType TypeCode
	params     []ParameterDescription
	exceptions []ExceptionDef
}

// Ensure operationDefImpl implements OperationDef
var _ OperationDef = &operationDefImpl{}

func (o *operationDefImpl) Result() TypeCode {
	return o.resultType
}

func (o *operationDefImpl) Params() []ParameterDescription {
	return o.params
}

func (o *operationDefImpl) Exceptions() []ExceptionDef {
	return o.exceptions
}

func (o *operationDefImpl) AddParameter(name string, type_code TypeCode, mode ParameterMode) error {
	o.params = append(o.params, ParameterDescription{
		Name: name,
		Type: type_code,
		Mode: mode,
	})

	return nil
}

func (o *operationDefImpl) AddException(except ExceptionDef) error {
	o.exceptions = append(o.exceptions, except)
	return nil
}

// typeCodeBase implements the TypeCode interface
type typeCodeBase struct {
	id   string
	name string
	kind DefinitionKind
}

func (tc *typeCodeBase) Kind() DefinitionKind {
	return tc.kind
}

func (tc *typeCodeBase) Id() string {
	return tc.id
}

func (tc *typeCodeBase) Name() string {
	return tc.name
}

func (tc *typeCodeBase) Equal(other TypeCode) bool {
	if tc.kind != other.Kind() {
		return false
	}

	return tc.id == other.Id() && tc.name == other.Name()
}

func (tc *typeCodeBase) String() string {
	return fmt.Sprintf("%s %s (ID: %s)", tc.kind, tc.name, tc.id)
}

// primitiveTypeCode implements primitive types in the repository
type primitiveTypeCode struct {
	typeCodeBase
}

// Implement IRObject interface for primitiveTypeCode
func (p *primitiveTypeCode) DefKind() DefinitionKind {
	return p.kind
}

func (p *primitiveTypeCode) Container() IRContainer {
	return nil
}

func (p *primitiveTypeCode) Describe() string {
	return fmt.Sprintf("Primitive TypeCode: %s (ID: %s)", p.name, p.id)
}

// structDefImpl implements the StructDef interface
type structDefImpl struct {
	containedBase
	typeCodeBase
	members []StructMember
	mu      sync.RWMutex
}

// Ensure structDefImpl implements StructDef
var _ StructDef = &structDefImpl{}

func (s *structDefImpl) Members() []StructMember {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.members
}

func (s *structDefImpl) AddMember(name string, type_code TypeCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate members
	for _, member := range s.members {
		if member.Name == name {
			return ErrDuplicateDefinition
		}
	}

	s.members = append(s.members, StructMember{
		Name: name,
		Type: type_code,
	})

	return nil
}

// Fix method ambiguity in structDefImpl
func (s *structDefImpl) Container() IRContainer {
	return s.containedBase.Container()
}

func (s *structDefImpl) Id() string {
	return s.containedBase.Id()
}

func (s *structDefImpl) Name() string {
	return s.containedBase.Name()
}

func (s *structDefImpl) DefKind() DefinitionKind {
	return s.containedBase.DefKind()
}

func (s *structDefImpl) Describe() string {
	return s.containedBase.Describe()
}

// exceptionDefImpl implements the ExceptionDef interface
type exceptionDefImpl struct {
	containedBase
	typeCodeBase
	members []StructMember
	mu      sync.RWMutex
}

// Ensure exceptionDefImpl implements ExceptionDef
var _ ExceptionDef = &exceptionDefImpl{}

func (e *exceptionDefImpl) Members() []StructMember {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.members
}

func (e *exceptionDefImpl) AddMember(name string, type_code TypeCode) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check for duplicate members
	for _, member := range e.members {
		if member.Name == name {
			return ErrDuplicateDefinition
		}
	}

	e.members = append(e.members, StructMember{
		Name: name,
		Type: type_code,
	})

	return nil
}

// Fix method ambiguity in exceptionDefImpl
func (e *exceptionDefImpl) Container() IRContainer {
	return e.containedBase.Container()
}

func (e *exceptionDefImpl) Id() string {
	return e.containedBase.Id()
}

func (e *exceptionDefImpl) Name() string {
	return e.containedBase.Name()
}

func (e *exceptionDefImpl) DefKind() DefinitionKind {
	return e.containedBase.DefKind()
}

func (e *exceptionDefImpl) Describe() string {
	return e.containedBase.Describe()
}

// unionDefImpl implements the UnionDef interface
type unionDefImpl struct {
	containedBase
	typeCodeBase
	discriminator TypeCode
	members       []UnionMember
	mu            sync.RWMutex
}

// Ensure unionDefImpl implements UnionDef
var _ UnionDef = &unionDefImpl{}

func (u *unionDefImpl) Discriminator() TypeCode {
	return u.discriminator
}

func (u *unionDefImpl) Members() []UnionMember {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.members
}

func (u *unionDefImpl) AddMember(name string, label interface{}, type_code TypeCode) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Check for duplicate members
	for _, member := range u.members {
		if member.Name == name {
			return ErrDuplicateDefinition
		}
	}

	u.members = append(u.members, UnionMember{
		Name:  name,
		Label: label,
		Type:  type_code,
	})

	return nil
}

// Fix method ambiguity in unionDefImpl
func (u *unionDefImpl) Container() IRContainer {
	return u.containedBase.Container()
}

func (u *unionDefImpl) Id() string {
	return u.containedBase.Id()
}

func (u *unionDefImpl) Name() string {
	return u.containedBase.Name()
}

func (u *unionDefImpl) DefKind() DefinitionKind {
	return u.containedBase.DefKind()
}

func (u *unionDefImpl) Describe() string {
	return u.containedBase.Describe()
}

// enumDefImpl implements the EnumDef interface
type enumDefImpl struct {
	containedBase
	typeCodeBase
	members []string
	mu      sync.RWMutex
}

// Ensure enumDefImpl implements EnumDef
var _ EnumDef = &enumDefImpl{}

func (e *enumDefImpl) Members() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.members
}

func (e *enumDefImpl) AddMember(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check for duplicate members
	for _, member := range e.members {
		if member == name {
			return ErrDuplicateDefinition
		}
	}

	e.members = append(e.members, name)

	return nil
}

// Fix method ambiguity in enumDefImpl
func (e *enumDefImpl) Container() IRContainer {
	return e.containedBase.Container()
}

func (e *enumDefImpl) Id() string {
	return e.containedBase.Id()
}

func (e *enumDefImpl) Name() string {
	return e.containedBase.Name()
}

func (e *enumDefImpl) DefKind() DefinitionKind {
	return e.containedBase.DefKind()
}

func (e *enumDefImpl) Describe() string {
	return e.containedBase.Describe()
}

// aliasDefImpl implements the AliasDef interface
type aliasDefImpl struct {
	containedBase
	typeCodeBase
	original TypeCode
}

// Ensure aliasDefImpl implements AliasDef
var _ AliasDef = &aliasDefImpl{}

func (a *aliasDefImpl) OriginalType() TypeCode {
	return a.original
}

// Fix method ambiguity in aliasDefImpl
func (a *aliasDefImpl) Container() IRContainer {
	return a.containedBase.Container()
}

func (a *aliasDefImpl) Id() string {
	return a.containedBase.Id()
}

func (a *aliasDefImpl) Name() string {
	return a.containedBase.Name()
}

func (a *aliasDefImpl) DefKind() DefinitionKind {
	return a.containedBase.DefKind()
}

func (a *aliasDefImpl) Describe() string {
	return a.containedBase.Describe()
}

// interfaceRepositoryImpl implements the InterfaceRepository interface
type interfaceRepositoryImpl struct {
	repository Repository
	mu         sync.RWMutex
	registry   map[string]string // Maps object impl -> repository ID
}

// NewInterfaceRepository creates a new Interface Repository service
func NewInterfaceRepository() InterfaceRepository {
	return &interfaceRepositoryImpl{
		repository: NewRepository(),
		registry:   make(map[string]string),
	}
}

func (ir *interfaceRepositoryImpl) GetRepository() Repository {
	return ir.repository
}

func (ir *interfaceRepositoryImpl) RegisterServant(servant interface{}, id string) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	// Register the object with its repository ID
	key := fmt.Sprintf("%v", reflect.ValueOf(servant).Pointer())
	ir.registry[key] = id
	return nil
}

func (ir *interfaceRepositoryImpl) LookupInterface(id string) (InterfaceDef, error) {
	obj, err := ir.repository.LookupId(id)
	if err != nil {
		return nil, err
	}

	if iface, ok := obj.(InterfaceDef); ok {
		return iface, nil
	}

	return nil, fmt.Errorf("object with ID %s is not an interface", id)
}

func (ir *interfaceRepositoryImpl) GetRepositoryID(obj interface{}) (string, error) {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	key := fmt.Sprintf("%v", reflect.ValueOf(obj).Pointer())
	id, ok := ir.registry[key]
	if !ok {
		return "", fmt.Errorf("object not registered with Interface Repository")
	}

	return id, nil
}

func (ir *interfaceRepositoryImpl) IsA(obj interface{}, interfaceID string) (bool, error) {
	// First, get the object's repository ID
	objID, err := ir.GetRepositoryID(obj)
	if err != nil {
		return false, err
	}

	// If the IDs match directly, return true
	if objID == interfaceID {
		return true, nil
	}

	// Use a recursive approach with cycle detection to check inheritance
	visited := make(map[string]bool)

	var checkInterface func(id string) (bool, error)
	checkInterface = func(id string) (bool, error) {
		// Prevent infinite recursion due to cycles in the interface hierarchy
		if visited[id] {
			return false, nil
		}
		visited[id] = true

		// Check if this is the interface we're looking for
		if id == interfaceID {
			return true, nil
		}

		// Get the interface definition
		iface, err := ir.LookupInterface(id)
		if err != nil {
			return false, err
		}

		// Check all base interfaces
		for _, base := range iface.BaseInterfaces() {
			if isA, err := checkInterface(base.Id()); err != nil {
				return false, err
			} else if isA {
				return true, nil
			}
		}

		return false, nil
	}

	// Start the recursive check
	return checkInterface(objID)
}

func (ir *interfaceRepositoryImpl) GetInterfaces(obj interface{}) ([]string, error) {
	// First, get the object's repository ID
	objID, err := ir.GetRepositoryID(obj)
	if err != nil {
		return nil, err
	}

	// Use a depth-first search to collect all interfaces
	interfaces := make(map[string]bool)
	interfaces[objID] = true // Start with the object's own interface

	// Use a recursive approach with cycle detection
	visited := make(map[string]bool)

	var collectInterfaces func(id string) error
	collectInterfaces = func(id string) error {
		// Prevent infinite recursion due to cycles in the interface hierarchy
		if visited[id] {
			return nil
		}
		visited[id] = true

		// Get the interface definition
		iface, err := ir.LookupInterface(id)
		if err != nil {
			return err
		}

		// Add all base interfaces
		for _, base := range iface.BaseInterfaces() {
			baseID := base.Id()
			interfaces[baseID] = true

			// Recursively collect base interfaces
			if err := collectInterfaces(baseID); err != nil {
				return err
			}
		}

		return nil
	}

	// Start the collection from the object's interface
	if err := collectInterfaces(objID); err != nil {
		return nil, err
	}

	// Convert map keys to slice
	result := make([]string, 0, len(interfaces))
	for id := range interfaces {
		result = append(result, id)
	}

	return result, nil
}
