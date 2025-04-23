// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"reflect"
	"strings"
)

// CompletionStatus indicates the status of an operation that raised an exception
type CompletionStatus int32

const (
	// CompletionStatusYes indicates the operation was completed
	CompletionStatusYes CompletionStatus = 0
	// CompletionStatusNo indicates the operation was not completed
	CompletionStatusNo CompletionStatus = 1
	// CompletionStatusMaybe indicates the operation completion status is unknown
	CompletionStatusMaybe CompletionStatus = 2
)

// Exception is the base interface for all CORBA exceptions
type Exception interface {
	error
	ID() string                  // Repository ID of this exception
	Name() string                // Name of this exception
	Minor() uint32               // Minor code for the exception
	Completed() CompletionStatus // Completion status of the operation
}

// SystemException represents a CORBA system exception
type SystemException struct {
	exceptionName  string
	minorCode      uint32
	completedValue CompletionStatus
}

// UserException represents a CORBA user-defined exception
type UserException struct {
	exceptionName string
	exceptionID   string
	members       map[string]interface{}
}

// NewCORBASystemException creates a new CORBA system exception
func NewCORBASystemException(name string, minor uint32, completed CompletionStatus) *SystemException {
	return &SystemException{
		exceptionName:  name,
		minorCode:      minor,
		completedValue: completed,
	}
}

// NewCORBAUserException creates a new CORBA user-defined exception
func NewCORBAUserException(name string, id string) *UserException {
	return &UserException{
		exceptionName: name,
		exceptionID:   id,
		members:       make(map[string]interface{}),
	}
}

// Error implements the error interface for SystemException
func (e *SystemException) Error() string {
	return fmt.Sprintf("CORBA System Exception: %s (minor code: %d, completion status: %v)",
		e.exceptionName, e.minorCode, e.completedValue)
}

// ID returns the repository ID of this system exception
func (e *SystemException) ID() string {
	return fmt.Sprintf("IDL:omg.org/CORBA/%s:1.0", e.exceptionName)
}

// Name returns the name of this system exception
func (e *SystemException) Name() string {
	return e.exceptionName
}

// Minor returns the minor code of this system exception
func (e *SystemException) Minor() uint32 {
	return e.minorCode
}

// Completed returns the completion status of the operation that raised this exception
func (e *SystemException) Completed() CompletionStatus {
	return e.completedValue
}

// Error implements the error interface for UserException
func (e *UserException) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CORBA User Exception: %s (ID: %s)", e.exceptionName, e.exceptionID))

	if len(e.members) > 0 {
		sb.WriteString(", members: [")
		first := true
		for name, value := range e.members {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", name, value))
			first = false
		}
		sb.WriteString("]")
	}

	return sb.String()
}

// ID returns the repository ID of this user exception
func (e *UserException) ID() string {
	return e.exceptionID
}

// Name returns the name of this user exception
func (e *UserException) Name() string {
	return e.exceptionName
}

// Minor returns the minor code of this user exception (always 0)
func (e *UserException) Minor() uint32 {
	return 0
}

// Completed returns the completion status of the operation that raised this exception (always No)
func (e *UserException) Completed() CompletionStatus {
	return CompletionStatusNo
}

// SetMember sets a member value for this user exception
func (e *UserException) SetMember(name string, value interface{}) {
	e.members[name] = value
}

// GetMember retrieves a member value from this user exception
func (e *UserException) GetMember(name string) (interface{}, bool) {
	value, exists := e.members[name]
	return value, exists
}

// Members returns all members of this user exception
func (e *UserException) Members() map[string]interface{} {
	// Return a copy to prevent modification of internal state
	result := make(map[string]interface{}, len(e.members))
	for k, v := range e.members {
		result[k] = v
	}
	return result
}

// Standard CORBA system exceptions as defined in the CORBA specification
var (
	// UNKNOWN - The unknown exception
	UNKNOWN = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("UNKNOWN", minor, completed)
	}

	// BAD_PARAM - An invalid parameter was passed
	BAD_PARAM = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_PARAM", minor, completed)
	}

	// NO_MEMORY - Dynamic memory allocation failure
	NO_MEMORY = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("NO_MEMORY", minor, completed)
	}

	// IMP_LIMIT - Violated implementation limit
	IMP_LIMIT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("IMP_LIMIT", minor, completed)
	}

	// COMM_FAILURE - Communication failure
	COMM_FAILURE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("COMM_FAILURE", minor, completed)
	}

	// INV_OBJREF - Invalid object reference
	INV_OBJREF = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INV_OBJREF", minor, completed)
	}

	// NO_PERMISSION - No permission for attempted operation
	NO_PERMISSION = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("NO_PERMISSION", minor, completed)
	}

	// INTERNAL - ORB internal error
	INTERNAL = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INTERNAL", minor, completed)
	}

	// MARSHAL - Error marshalling parameter or result
	MARSHAL = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("MARSHAL", minor, completed)
	}

	// INITIALIZE - ORB initialization failure
	INITIALIZE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INITIALIZE", minor, completed)
	}

	// NO_IMPLEMENT - Operation implementation unavailable
	NO_IMPLEMENT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("NO_IMPLEMENT", minor, completed)
	}

	// BAD_TYPECODE - Bad typecode
	BAD_TYPECODE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_TYPECODE", minor, completed)
	}

	// BAD_OPERATION - Invalid operation
	BAD_OPERATION = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_OPERATION", minor, completed)
	}

	// NO_RESOURCES - Insufficient resources for request
	NO_RESOURCES = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("NO_RESOURCES", minor, completed)
	}

	// NO_RESPONSE - Response to request not yet available
	NO_RESPONSE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("NO_RESPONSE", minor, completed)
	}

	// PERSIST_STORE - Persistent storage failure
	PERSIST_STORE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("PERSIST_STORE", minor, completed)
	}

	// BAD_INV_ORDER - Routine invocations out of order
	BAD_INV_ORDER = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_INV_ORDER", minor, completed)
	}

	// TRANSIENT - Transient failure, reissue request
	TRANSIENT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TRANSIENT", minor, completed)
	}

	// FREE_MEM - Cannot free memory
	FREE_MEM = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("FREE_MEM", minor, completed)
	}

	// INV_IDENT - Invalid identifier
	INV_IDENT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INV_IDENT", minor, completed)
	}

	// INV_FLAG - Invalid flag was specified
	INV_FLAG = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INV_FLAG", minor, completed)
	}

	// INTF_REPOS - Error accessing interface repository
	INTF_REPOS = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INTF_REPOS", minor, completed)
	}

	// BAD_CONTEXT - Error processing context object
	BAD_CONTEXT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_CONTEXT", minor, completed)
	}

	// OBJ_ADAPTER - Failure detected by object adapter
	OBJ_ADAPTER = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("OBJ_ADAPTER", minor, completed)
	}

	// DATA_CONVERSION - Data conversion error
	DATA_CONVERSION = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("DATA_CONVERSION", minor, completed)
	}

	// OBJECT_NOT_EXIST - Non-existent object, delete reference
	OBJECT_NOT_EXIST = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("OBJECT_NOT_EXIST", minor, completed)
	}

	// TRANSACTION_REQUIRED - Transaction required
	TRANSACTION_REQUIRED = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TRANSACTION_REQUIRED", minor, completed)
	}

	// TRANSACTION_ROLLEDBACK - Transaction rolled back
	TRANSACTION_ROLLEDBACK = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TRANSACTION_ROLLEDBACK", minor, completed)
	}

	// INVALID_TRANSACTION - Invalid transaction
	INVALID_TRANSACTION = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INVALID_TRANSACTION", minor, completed)
	}

	// INV_POLICY - Invalid policy
	INV_POLICY = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("INV_POLICY", minor, completed)
	}

	// CODESET_INCOMPATIBLE - Incompatible code set
	CODESET_INCOMPATIBLE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("CODESET_INCOMPATIBLE", minor, completed)
	}

	// REBIND - Rebind needed
	REBIND = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("REBIND", minor, completed)
	}

	// TIMEOUT - Operation timed out
	TIMEOUT = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TIMEOUT", minor, completed)
	}

	// TRANSACTION_UNAVAILABLE - Transaction unavailable
	TRANSACTION_UNAVAILABLE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TRANSACTION_UNAVAILABLE", minor, completed)
	}

	// TRANSACTION_MODE - Invalid transaction mode
	TRANSACTION_MODE = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("TRANSACTION_MODE", minor, completed)
	}

	// BAD_QOS - Bad quality of service
	BAD_QOS = func(minor uint32, completed CompletionStatus) *SystemException {
		return NewCORBASystemException("BAD_QOS", minor, completed)
	}
)

// ExceptionHolder is used to transport exceptions across CORBA boundaries
type ExceptionHolder struct {
	Exception Exception
	TypeCode  TypeCode
}

// NewExceptionHolder creates a new exception holder
func NewExceptionHolder(ex Exception) (*ExceptionHolder, error) {
	var tc TypeCode
	var err error

	if sysEx, ok := ex.(*SystemException); ok {
		// Create a TypeCode for a system exception
		tc, err = CreateSystemExceptionTypeCode(sysEx.Name())
	} else if userEx, ok := ex.(*UserException); ok {
		// Lookup TypeCode for this user exception from IR
		tc, err = GetTypeCode(userEx.ID())
		if err != nil {
			// If not found, create a minimal TypeCode
			tc = &typeCodeBase{
				id:   userEx.ID(),
				name: userEx.Name(),
				kind: DK_EXCEPTION,
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported exception type: %T", ex)
	}

	return &ExceptionHolder{
		Exception: ex,
		TypeCode:  tc,
	}, err
}

// CreateSystemExceptionTypeCode creates a TypeCode for a system exception
func CreateSystemExceptionTypeCode(name string) (TypeCode, error) {
	id := fmt.Sprintf("IDL:omg.org/CORBA/%s:1.0", name)

	// Check if it already exists in the registry
	tc, err := GetTypeCode(id)
	if err == nil {
		return tc, nil
	}

	// Create a new struct-like TypeCode for system exceptions
	sysExTc, err := CreateStructTypeCode(id, name)
	if err != nil {
		return nil, err
	}

	// Add the standard members of system exceptions
	minorTc, _ := GetBasicTypeCode(TC_ULONG)
	completedTc, _ := GetBasicTypeCode(TC_LONG)

	// Add members
	sysExTc.AddMember("minor", minorTc)
	sysExTc.AddMember("completed", completedTc)

	return sysExTc, nil
}

// IsSystemException checks if an error is a CORBA system exception
func IsSystemException(err error) bool {
	_, ok := err.(*SystemException)
	return ok
}

// IsUserException checks if an error is a CORBA user exception
func IsUserException(err error) bool {
	_, ok := err.(*UserException)
	return ok
}

// IsException checks if an error is a CORBA exception (system or user)
func IsException(err error) bool {
	return IsSystemException(err) || IsUserException(err)
}

// GetExceptionTypeCode returns the TypeCode for an exception
func GetExceptionTypeCode(ex Exception) (TypeCode, error) {
	if ex == nil {
		return nil, fmt.Errorf("cannot get TypeCode for nil exception")
	}

	holder, err := NewExceptionHolder(ex)
	if err != nil {
		return nil, err
	}

	return holder.TypeCode, nil
}

// MarshalException serializes an exception for transmission
func MarshalException(ex Exception) ([]byte, error) {
	// This is a placeholder for actual marshalling code
	// In a real implementation, this would use CDR marshalling to encode the exception

	// For now, we'll simulate it with a simple string encoding
	var payload string

	if sysEx, ok := ex.(*SystemException); ok {
		payload = fmt.Sprintf("SYSTEM:%s:%d:%d", sysEx.Name(), sysEx.Minor(), sysEx.Completed())
	} else if userEx, ok := ex.(*UserException); ok {
		payload = fmt.Sprintf("USER:%s:%s", userEx.ID(), userEx.Name())
		// In a real implementation, we would also marshal the member values
	} else {
		return nil, fmt.Errorf("unsupported exception type: %T", ex)
	}

	return []byte(payload), nil
}

// UnmarshalException deserializes an exception from transmission
func UnmarshalException(data []byte, tc TypeCode) (Exception, error) {
	// This is a placeholder for actual unmarshalling code
	// In a real implementation, this would use CDR unmarshalling to decode the exception

	// For now, we'll simulate it with a simple string decoding
	payload := string(data)
	parts := strings.Split(payload, ":")

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid exception data format")
	}

	switch parts[0] {
	case "SYSTEM":
		if len(parts) < 4 {
			return nil, fmt.Errorf("invalid system exception data format")
		}

		name := parts[1]
		minor, err := parseUint32(parts[2])
		if err != nil {
			return nil, err
		}

		completed, err := parseInt32(parts[3])
		if err != nil {
			return nil, err
		}

		return NewCORBASystemException(name, minor, CompletionStatus(completed)), nil

	case "USER":
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid user exception data format")
		}

		id := parts[1]
		name := parts[2]

		return NewCORBAUserException(name, id), nil

	default:
		return nil, fmt.Errorf("unknown exception type: %s", parts[0])
	}
}

// Helper functions for parsing
func parseUint32(s string) (uint32, error) {
	var result uint32
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func parseInt32(s string) (int32, error) {
	var result int32
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// ExceptionRegistry maintains a registry of user-defined exceptions
type ExceptionRegistry struct {
	exceptions map[string]reflect.Type
}

// Global exception registry
var globalExceptionRegistry = NewExceptionRegistry()

// NewExceptionRegistry creates a new exception registry
func NewExceptionRegistry() *ExceptionRegistry {
	return &ExceptionRegistry{
		exceptions: make(map[string]reflect.Type),
	}
}

// Register registers a user-defined exception type with the registry
func (r *ExceptionRegistry) Register(id string, exType reflect.Type) {
	r.exceptions[id] = exType
}

// Lookup looks up a user-defined exception type in the registry
func (r *ExceptionRegistry) Lookup(id string) (reflect.Type, bool) {
	t, ok := r.exceptions[id]
	return t, ok
}

// RegisterException registers a user-defined exception type with the global registry
func RegisterException(id string, ex interface{}) {
	globalExceptionRegistry.Register(id, reflect.TypeOf(ex))
}

// CreateExceptionFromTypeCode creates a new exception instance from its TypeCode
func CreateExceptionFromTypeCode(tc TypeCode) (Exception, error) {
	if tc == nil {
		return nil, fmt.Errorf("cannot create exception from nil TypeCode")
	}

	// Check if it's a system exception
	if strings.HasPrefix(tc.Id(), "IDL:omg.org/CORBA/") {
		// Extract the name from the ID
		name := strings.TrimPrefix(tc.Id(), "IDL:omg.org/CORBA/")
		name = strings.TrimSuffix(name, ":1.0")

		return NewCORBASystemException(name, 0, CompletionStatusNo), nil
	}

	// Otherwise, it's a user exception
	return NewCORBAUserException(tc.Name(), tc.Id()), nil
}

// ThrowableToException converts a Go error or panic to a CORBA exception
func ThrowableToException(err interface{}) Exception {
	switch e := err.(type) {
	case nil:
		// No error
		return nil

	case Exception:
		// Already a CORBA exception
		return e

	case error:
		// Convert Go error to UNKNOWN system exception
		return UNKNOWN(0, CompletionStatusNo)

	default:
		// Convert other panics to UNKNOWN system exception
		return UNKNOWN(0, CompletionStatusNo)
	}
}

// RecoverException tries to recover from a panic and convert it to a CORBA exception
func RecoverException() Exception {
	r := recover()
	if r == nil {
		return nil
	}

	return ThrowableToException(r)
}

// SafeInvoke safely invokes a function and converts any panics to exceptions
func SafeInvoke(fn func() (interface{}, error)) (interface{}, Exception) {
	var ex Exception
	defer func() {
		if r := recover(); r != nil {
			ex = ThrowableToException(r)
		}
	}()

	result, err := fn()
	if err != nil {
		return nil, ThrowableToException(err)
	}

	return result, ex
}

// UpdateExceptionHandling updates the exception handling in the CORBA server
// to properly handle system and user exceptions
func UpdateExceptionHandling(server *Server) {
	// This function would modify the server to use proper exception handling
	// However, as this would require deeper integration with the server's message
	// processing logic, we'll leave this as a placeholder for now.
}

// GetExceptionFromError extracts a CORBA exception from an error
func GetExceptionFromError(err error) (Exception, bool) {
	if ex, ok := err.(Exception); ok {
		return ex, true
	}
	return nil, false
}
