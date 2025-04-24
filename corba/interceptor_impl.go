// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LoggingClientInterceptor provides a client interceptor that logs requests and responses
type LoggingClientInterceptor struct {
	logLevel int // 0=none, 1=basic, 2=detailed
}

// NewLoggingClientInterceptor creates a new logging client interceptor
func NewLoggingClientInterceptor(logLevel int) *LoggingClientInterceptor {
	return &LoggingClientInterceptor{
		logLevel: logLevel,
	}
}

// Name returns the name of the interceptor
func (i *LoggingClientInterceptor) Name() string {
	return "LoggingClientInterceptor"
}

// SendRequest is called before the request is sent to the server
func (i *LoggingClientInterceptor) SendRequest(info *RequestInfo) error {
	if i.logLevel > 0 {
		log.Printf("[CLIENT-REQUEST] Sending %s request to %s", info.Operation, info.ObjectKey)

		if i.logLevel > 1 {
			log.Printf("[CLIENT-REQUEST] Arguments: %v", info.Arguments)
			log.Printf("[CLIENT-REQUEST] Service Contexts: %d", len(info.ServiceContexts))
		}
	}
	return nil
}

// ReceiveReply is called after a normal reply is received
func (i *LoggingClientInterceptor) ReceiveReply(info *RequestInfo) error {
	if i.logLevel > 0 {
		log.Printf("[CLIENT-REPLY] Received reply for %s from %s", info.Operation, info.ObjectKey)

		if i.logLevel > 1 {
			log.Printf("[CLIENT-REPLY] Result: %v", info.Result)
			log.Printf("[CLIENT-REPLY] Service Contexts: %d", len(info.ServiceContexts))
		}
	}
	return nil
}

// ReceiveException is called if an exception is received
func (i *LoggingClientInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	log.Printf("[CLIENT-EXCEPTION] Received exception for %s: %s", info.Operation, ex.Error())
	return nil
}

// ReceiveOther is called for other outcomes (timeout, etc.)
func (i *LoggingClientInterceptor) ReceiveOther(info *RequestInfo) error {
	log.Printf("[CLIENT-OTHER] Received other response for %s", info.Operation)
	return nil
}

// LoggingServerInterceptor provides a server interceptor that logs requests and responses
type LoggingServerInterceptor struct {
	logLevel int // 0=none, 1=basic, 2=detailed
}

// NewLoggingServerInterceptor creates a new logging server interceptor
func NewLoggingServerInterceptor(logLevel int) *LoggingServerInterceptor {
	return &LoggingServerInterceptor{
		logLevel: logLevel,
	}
}

// Name returns the name of the interceptor
func (i *LoggingServerInterceptor) Name() string {
	return "LoggingServerInterceptor"
}

// ReceiveRequest is called before the servant operation is invoked
func (i *LoggingServerInterceptor) ReceiveRequest(info *RequestInfo) error {
	if i.logLevel > 0 {
		log.Printf("[SERVER-REQUEST] Received %s request for %s", info.Operation, info.ObjectKey)

		if i.logLevel > 1 {
			log.Printf("[SERVER-REQUEST] Arguments: %v", info.Arguments)
			log.Printf("[SERVER-REQUEST] Service Contexts: %d", len(info.ServiceContexts))
		}
	}
	return nil
}

// SendReply is called after the servant operation returns
func (i *LoggingServerInterceptor) SendReply(info *RequestInfo) error {
	if i.logLevel > 0 {
		log.Printf("[SERVER-REPLY] Sending reply for %s operation", info.Operation)

		if i.logLevel > 1 {
			log.Printf("[SERVER-REPLY] Result: %v", info.Result)
		}
	}
	return nil
}

// SendException is called if the operation raises an exception
func (i *LoggingServerInterceptor) SendException(info *RequestInfo, ex Exception) error {
	log.Printf("[SERVER-EXCEPTION] Exception for %s: %s", info.Operation, ex.Error())
	return nil
}

// TimingInterceptor times operation execution
type TimingInterceptor struct {
	clientTimings map[uint32]time.Time
	serverTimings map[uint32]time.Time
	mu            sync.RWMutex
}

// NewTimingInterceptor creates a new timing interceptor
func NewTimingInterceptor() *TimingInterceptor {
	return &TimingInterceptor{
		clientTimings: make(map[uint32]time.Time),
		serverTimings: make(map[uint32]time.Time),
	}
}

// Name returns the name of the interceptor
func (i *TimingInterceptor) Name() string {
	return "TimingInterceptor"
}

// Client interceptor methods
func (i *TimingInterceptor) SendRequest(info *RequestInfo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.clientTimings[info.RequestID] = time.Now()
	return nil
}

func (i *TimingInterceptor) ReceiveReply(info *RequestInfo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if start, ok := i.clientTimings[info.RequestID]; ok {
		duration := time.Since(start)
		log.Printf("[TIMING] Operation %s on %s took %v", info.Operation, info.ObjectKey, duration)
		delete(i.clientTimings, info.RequestID)
	}
	return nil
}

func (i *TimingInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if start, ok := i.clientTimings[info.RequestID]; ok {
		duration := time.Since(start)
		log.Printf("[TIMING] Failed operation %s on %s took %v", info.Operation, info.ObjectKey, duration)
		delete(i.clientTimings, info.RequestID)
	}
	return nil
}

func (i *TimingInterceptor) ReceiveOther(info *RequestInfo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if start, ok := i.clientTimings[info.RequestID]; ok {
		duration := time.Since(start)
		log.Printf("[TIMING] Other response for %s on %s took %v", info.Operation, info.ObjectKey, duration)
		delete(i.clientTimings, info.RequestID)
	}
	return nil
}

// Server interceptor methods
func (i *TimingInterceptor) ReceiveRequest(info *RequestInfo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.serverTimings[info.RequestID] = time.Now()
	return nil
}

func (i *TimingInterceptor) SendReply(info *RequestInfo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if start, ok := i.serverTimings[info.RequestID]; ok {
		duration := time.Since(start)
		log.Printf("[TIMING] Server handled %s on %s in %v", info.Operation, info.ObjectKey, duration)
		delete(i.serverTimings, info.RequestID)
	}
	return nil
}

func (i *TimingInterceptor) SendException(info *RequestInfo, ex Exception) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if start, ok := i.serverTimings[info.RequestID]; ok {
		duration := time.Since(start)
		log.Printf("[TIMING] Server exception for %s on %s after %v: %s",
			info.Operation, info.ObjectKey, duration, ex.Error())
		delete(i.serverTimings, info.RequestID)
	}
	return nil
}

// SecurityInterceptor implements basic security checks
type SecurityInterceptor struct {
	authToken     string
	requiredRoles map[string][]string // Method -> required roles
}

// NewSecurityInterceptor creates a new security interceptor
func NewSecurityInterceptor(authToken string) *SecurityInterceptor {
	return &SecurityInterceptor{
		authToken:     authToken,
		requiredRoles: make(map[string][]string),
	}
}

// Name returns the name of the interceptor
func (i *SecurityInterceptor) Name() string {
	return "SecurityInterceptor"
}

// RequireRole specifies that a method requires a specific role
func (i *SecurityInterceptor) RequireRole(method string, role string) {
	if roles, ok := i.requiredRoles[method]; ok {
		i.requiredRoles[method] = append(roles, role)
	} else {
		i.requiredRoles[method] = []string{role}
	}
}

// Server interceptor methods
func (i *SecurityInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Check for auth token in service context
	var token string
	for _, ctx := range info.ServiceContexts {
		if ctx.ID == 0x41555448 { // "AUTH"
			token = string(ctx.Data)
			break
		}
	}

	// Validate token
	if token != i.authToken {
		return OBJECT_NOT_EXIST(1, CompletionStatusNo)
	}

	// Check if method has role requirements
	if roles, ok := i.requiredRoles[info.Operation]; ok {
		// In a real implementation, we would check if the user has the required roles
		log.Printf("[SECURITY] Method %s requires roles: %v", info.Operation, roles)
		// For now, we'll just log the requirement
	}

	return nil
}

func (i *SecurityInterceptor) SendReply(info *RequestInfo) error {
	return nil
}

func (i *SecurityInterceptor) SendException(info *RequestInfo, ex Exception) error {
	return nil
}

// Client security interceptor
type ClientSecurityInterceptor struct {
	authToken string
}

// NewClientSecurityInterceptor creates a new client security interceptor
func NewClientSecurityInterceptor(authToken string) *ClientSecurityInterceptor {
	return &ClientSecurityInterceptor{
		authToken: authToken,
	}
}

// Name returns the name of the interceptor
func (i *ClientSecurityInterceptor) Name() string {
	return "ClientSecurityInterceptor"
}

func (i *ClientSecurityInterceptor) SendRequest(info *RequestInfo) error {
	// Add auth token to service context
	info.ServiceContexts = append(info.ServiceContexts, ServiceContext{
		ID:   0x41555448, // "AUTH"
		Data: []byte(i.authToken),
	})
	return nil
}

func (i *ClientSecurityInterceptor) ReceiveReply(info *RequestInfo) error {
	return nil
}

func (i *ClientSecurityInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	return nil
}

func (i *ClientSecurityInterceptor) ReceiveOther(info *RequestInfo) error {
	return nil
}

// TransactionInterceptor manages transaction contexts
type TransactionInterceptor struct {
	txID uint32
}

// NewTransactionInterceptor creates a new transaction interceptor
func NewTransactionInterceptor() *TransactionInterceptor {
	return &TransactionInterceptor{
		txID: 1, // Start with transaction ID 1
	}
}

// Name returns the name of the interceptor
func (i *TransactionInterceptor) Name() string {
	return "TransactionInterceptor"
}

func (i *TransactionInterceptor) nextTxID() uint32 {
	i.txID++
	return i.txID
}

// Client transaction methods
func (i *TransactionInterceptor) SendRequest(info *RequestInfo) error {
	// Create transaction ID and add to service context
	txID := i.nextTxID()
	info.ServiceContexts = append(info.ServiceContexts, ServiceContext{
		ID:   0x54524158, // "TRAX"
		Data: []byte(fmt.Sprintf("%d", txID)),
	})
	return nil
}

func (i *TransactionInterceptor) ReceiveReply(info *RequestInfo) error {
	return nil
}

func (i *TransactionInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	return nil
}

func (i *TransactionInterceptor) ReceiveOther(info *RequestInfo) error {
	return nil
}

// Server transaction methods
func (i *TransactionInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Extract transaction ID from service context
	var txID string
	for _, ctx := range info.ServiceContexts {
		if ctx.ID == 0x54524158 { // "TRAX"
			txID = string(ctx.Data)
			break
		}
	}

	if txID != "" {
		log.Printf("[TRANSACTION] Request %s is part of transaction %s",
			info.Operation, txID)
	}
	return nil
}

func (i *TransactionInterceptor) SendReply(info *RequestInfo) error {
	return nil
}

func (i *TransactionInterceptor) SendException(info *RequestInfo, ex Exception) error {
	return nil
}

// ServiceContextInterceptor adds and processes service contexts
type ServiceContextInterceptor struct {
	contexts map[uint32][]byte
}

// NewServiceContextInterceptor creates a new service context interceptor
func NewServiceContextInterceptor() *ServiceContextInterceptor {
	return &ServiceContextInterceptor{
		contexts: make(map[uint32][]byte),
	}
}

// Name returns the name of the interceptor
func (i *ServiceContextInterceptor) Name() string {
	return "ServiceContextInterceptor"
}

// AddContext adds a service context
func (i *ServiceContextInterceptor) AddContext(id uint32, data []byte) {
	i.contexts[id] = data
}

// Client methods
func (i *ServiceContextInterceptor) SendRequest(info *RequestInfo) error {
	// Add all contexts to the request
	for id, data := range i.contexts {
		info.ServiceContexts = append(info.ServiceContexts, ServiceContext{
			ID:   id,
			Data: data,
		})
	}
	return nil
}

func (i *ServiceContextInterceptor) ReceiveReply(info *RequestInfo) error {
	// Process contexts in the reply
	for _, ctx := range info.ServiceContexts {
		log.Printf("[SERVICE-CONTEXT] Received context ID: 0x%X, data length: %d",
			ctx.ID, len(ctx.Data))
	}
	return nil
}

func (i *ServiceContextInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	return nil
}

func (i *ServiceContextInterceptor) ReceiveOther(info *RequestInfo) error {
	return nil
}

// Server methods
func (i *ServiceContextInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Process contexts in the request
	for _, ctx := range info.ServiceContexts {
		log.Printf("[SERVICE-CONTEXT] Received context ID: 0x%X, data length: %d",
			ctx.ID, len(ctx.Data))
	}
	return nil
}

func (i *ServiceContextInterceptor) SendReply(info *RequestInfo) error {
	// Add contexts to the reply
	for id, data := range i.contexts {
		info.ServiceContexts = append(info.ServiceContexts, ServiceContext{
			ID:   id,
			Data: data,
		})
	}
	return nil
}

func (i *ServiceContextInterceptor) SendException(info *RequestInfo, ex Exception) error {
	return nil
}

// ParameterValidationInterceptor validates parameters on the client side
type ParameterValidationInterceptor struct {
	validators map[string]func([]interface{}) error
}

// NewParameterValidationInterceptor creates a new parameter validation interceptor
func NewParameterValidationInterceptor() *ParameterValidationInterceptor {
	return &ParameterValidationInterceptor{
		validators: make(map[string]func([]interface{}) error),
	}
}

// Name returns the name of the interceptor
func (i *ParameterValidationInterceptor) Name() string {
	return "ParameterValidationInterceptor"
}

// AddValidator adds a validator for a specific operation
func (i *ParameterValidationInterceptor) AddValidator(operation string, validator func([]interface{}) error) {
	i.validators[operation] = validator
}

// SendRequest validates parameters before sending the request
func (i *ParameterValidationInterceptor) SendRequest(info *RequestInfo) error {
	if validator, ok := i.validators[info.Operation]; ok {
		if err := validator(info.Arguments); err != nil {
			return BAD_PARAM(1, CompletionStatusNo)
		}
	}
	return nil
}

func (i *ParameterValidationInterceptor) ReceiveReply(info *RequestInfo) error {
	return nil
}

func (i *ParameterValidationInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	return nil
}

func (i *ParameterValidationInterceptor) ReceiveOther(info *RequestInfo) error {
	return nil
}
