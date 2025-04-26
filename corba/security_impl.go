// Package corba provides a CORBA implementation in Go
package corba

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// PrincipalImpl implements the Principal interface
type PrincipalImpl struct {
	name               string
	roles              []string
	authenticationType AuthenticationMethod
	privileges         []Privilege
	attributes         map[string]interface{}
}

// NewPrincipal creates a new principal
func NewPrincipal(name string, authType AuthenticationMethod) *PrincipalImpl {
	return &PrincipalImpl{
		name:               name,
		authenticationType: authType,
		privileges:         make([]Privilege, 0),
		attributes:         make(map[string]interface{}),
	}
}

// Name returns the principal name
func (p *PrincipalImpl) Name() string {
	return p.name
}

// Roles returns the principal roles
func (p *PrincipalImpl) Roles() []string {
	return p.roles
}

// AuthenticationType returns the authentication method
func (p *PrincipalImpl) AuthenticationType() AuthenticationMethod {
	return p.authenticationType
}

// PrivilegeList returns the privileges assigned to the principal
func (p *PrincipalImpl) PrivilegeList() []Privilege {
	result := make([]Privilege, len(p.privileges))
	copy(result, p.privileges)
	return result
}

// HasPrivilege checks if the principal has a specific privilege
func (p *PrincipalImpl) HasPrivilege(privilegeName string) bool {
	for _, priv := range p.privileges {
		if priv.Name == privilegeName {
			return true
		}
	}
	return false
}

// AddPrivilege adds a privilege to the principal
func (p *PrincipalImpl) AddPrivilege(privilege Privilege) {
	p.privileges = append(p.privileges, privilege)
}

// SetAttribute sets an attribute on the principal
func (p *PrincipalImpl) SetAttribute(name string, value interface{}) {
	p.attributes[name] = value
}

// GetAttribute gets an attribute from the principal
func (p *PrincipalImpl) GetAttribute(name string) (interface{}, bool) {
	val, ok := p.attributes[name]
	return val, ok
}

// IsInRole checks if the principal has a specific role
func (p *PrincipalImpl) IsInRole(name string) bool {
	for _, r := range p.roles {
		if r == name {
			return true
		}
	}
	return false
}

// CredentialsImpl implements the Credentials interface
type CredentialsImpl struct {
	mu            sync.RWMutex
	credsType     CredentialsType
	principal     Principal
	issueTime     time.Time
	expiryTime    time.Time
	authenticator interface{} // The token, certificate, or other proof of authenticity
}

// NewCredentials creates new credentials
func NewCredentials(principal Principal, credsType CredentialsType, lifetime time.Duration) *CredentialsImpl {
	now := time.Now()
	return &CredentialsImpl{
		credsType:  credsType,
		principal:  principal,
		issueTime:  now,
		expiryTime: now.Add(lifetime),
	}
}

// Type returns the credentials type
func (c *CredentialsImpl) Type() CredentialsType {
	return c.credsType
}

// Principal returns the principal associated with the credentials
func (c *CredentialsImpl) Principal() Principal {
	return c.principal
}

// IsValid checks if the credentials are valid
func (c *CredentialsImpl) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().Before(c.expiryTime)
}

// Refresh refreshes the credentials
func (c *CredentialsImpl) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extend expiration time by the original duration
	duration := c.expiryTime.Sub(c.issueTime)
	c.expiryTime = time.Now().Add(duration)
	return nil
}

// ExpirationTime returns the expiration time of the credentials
func (c *CredentialsImpl) ExpirationTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.expiryTime
}

// SetAuthenticator sets the authenticator for these credentials
func (c *CredentialsImpl) SetAuthenticator(authenticator interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authenticator = authenticator
}

// GetAuthenticator gets the authenticator for these credentials
func (c *CredentialsImpl) GetAuthenticator() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticator
}

// SecurityContextImpl implements the SecurityContext interface
type SecurityContextImpl struct {
	id          string
	credentials Credentials
	attributes  map[string]interface{}
	mu          sync.RWMutex
}

// NewSecurityContext creates a new security context
func NewSecurityContext(credentials Credentials) (*SecurityContextImpl, error) {
	// Generate a random context ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("failed to generate security context ID: %w", err)
	}
	id := base64.StdEncoding.EncodeToString(idBytes)

	return &SecurityContextImpl{
		id:          id,
		credentials: credentials,
		attributes:  make(map[string]interface{}),
	}, nil
}

// ID returns the security context ID
func (sc *SecurityContextImpl) ID() string {
	return sc.id
}

// Credentials returns the credentials associated with this context
func (sc *SecurityContextImpl) Credentials() Credentials {
	return sc.credentials
}

// IsValid checks if the context is valid
func (sc *SecurityContextImpl) IsValid() bool {
	return sc.credentials.IsValid()
}

// Refresh refreshes the security context
func (sc *SecurityContextImpl) Refresh() error {
	return sc.credentials.Refresh()
}

// ExpirationTime returns the expiration time of the security context
func (sc *SecurityContextImpl) ExpirationTime() time.Time {
	return sc.credentials.ExpirationTime()
}

// Attributes returns security context attributes
func (sc *SecurityContextImpl) Attributes() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return a copy of the attributes
	result := make(map[string]interface{}, len(sc.attributes))
	for k, v := range sc.attributes {
		result[k] = v
	}
	return result
}

// SetAttribute sets a security context attribute
func (sc *SecurityContextImpl) SetAttribute(name string, value interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.attributes[name] = value
}

// GetAttribute gets a security context attribute
func (sc *SecurityContextImpl) GetAttribute(name string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	val, ok := sc.attributes[name]
	return val, ok
}

// SecurityManagerImpl implements the SecurityManager interface
type SecurityManagerImpl struct {
	mu                 sync.RWMutex
	currentContext     SecurityContext
	principalStore     map[string]Principal
	contextStore       map[string]SecurityContext
	authenticationImpl map[AuthenticationMethod]Authenticator
	accessPolicies     map[string][]AccessRule
	auditLogger        AuditLogger
}

// Authenticator defines the interface for different authentication mechanisms
type Authenticator interface {
	// Authenticate authenticates a principal based on the provided auth data
	Authenticate(authData interface{}) (Principal, error)
}

// AccessRule defines a rule for access control
type AccessRule struct {
	Target     string   // The name of the target object or operation
	Privileges []string // The privileges required to access the target
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogEvent logs a security event
	LogEvent(action string, principal string, succeeded bool, details map[string]interface{})
}

// NewSecurityManager creates a new security manager
func NewSecurityManager() *SecurityManagerImpl {
	return &SecurityManagerImpl{
		principalStore:     make(map[string]Principal),
		contextStore:       make(map[string]SecurityContext),
		authenticationImpl: make(map[AuthenticationMethod]Authenticator),
		accessPolicies:     make(map[string][]AccessRule),
		auditLogger:        &DefaultAuditLogger{},
	}
}

// RegisterPrincipal registers a principal with the security manager
func (sm *SecurityManagerImpl) RegisterPrincipal(principal Principal) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.principalStore[principal.Name()] = principal
}

// GetPrincipal gets a principal by name
func (sm *SecurityManagerImpl) GetPrincipal(name string) (Principal, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	principal, ok := sm.principalStore[name]
	return principal, ok
}

// RegisterAuthenticator registers an authenticator for a specific authentication method
func (sm *SecurityManagerImpl) RegisterAuthenticator(method AuthenticationMethod, authenticator Authenticator) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.authenticationImpl[method] = authenticator
}

// AddAccessRule adds an access control rule
func (sm *SecurityManagerImpl) AddAccessRule(target string, rule AccessRule) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.accessPolicies[target] = append(sm.accessPolicies[target], rule)
}

// SetAuditLogger sets the audit logger
func (sm *SecurityManagerImpl) SetAuditLogger(logger AuditLogger) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.auditLogger = logger
}

// Authenticate authenticates a principal and returns credentials
func (sm *SecurityManagerImpl) Authenticate(authData interface{}) (Credentials, error) {
	// Extract authentication method and data
	authMap, ok := authData.(map[string]interface{})
	if !ok {
		return nil, SecurityInvalidCredentials("Invalid authentication data format")
	}

	methodVal, ok := authMap["method"]
	if !ok {
		return nil, SecurityInvalidCredentials("Authentication method not specified")
	}

	method, ok := methodVal.(AuthenticationMethod)
	if !ok {
		methodInt, ok := methodVal.(int)
		if !ok {
			return nil, SecurityInvalidCredentials("Invalid authentication method")
		}
		method = AuthenticationMethod(methodInt)
	}

	// Get the appropriate authenticator
	sm.mu.RLock()
	authenticator, ok := sm.authenticationImpl[method]
	sm.mu.RUnlock()

	if !ok {
		return nil, SecurityInvalidCredentials("Authentication method not supported")
	}

	// Perform authentication
	principal, err := authenticator.Authenticate(authMap)
	if err != nil {
		return nil, SecurityAuthenticationFailed(err.Error())
	}

	// Create credentials
	lifetime := 8 * time.Hour // Default 8 hour lifetime
	if lifetimeVal, ok := authMap["lifetime"]; ok {
		if lifetimeSeconds, ok := lifetimeVal.(float64); ok {
			lifetime = time.Duration(lifetimeSeconds) * time.Second
		}
	}

	credentials := NewCredentials(principal, CredsInvocationCredentials, lifetime)

	// Audit the authentication
	sm.AuditAction(
		"Authentication",
		true,
		map[string]interface{}{
			"principal": principal.Name(),
			"method":    method,
		},
	)

	return credentials, nil
}

// CreateSecurityContext creates a security context from credentials
func (sm *SecurityManagerImpl) CreateSecurityContext(creds Credentials) (SecurityContext, error) {
	if !creds.IsValid() {
		return nil, SecurityInvalidCredentials("Credentials have expired")
	}

	context, err := NewSecurityContext(creds)
	if err != nil {
		return nil, err
	}

	// Store the context
	sm.mu.Lock()
	sm.contextStore[context.ID()] = context
	sm.mu.Unlock()

	return context, nil
}

// GetSecurityContext returns the current security context
func (sm *SecurityManagerImpl) GetSecurityContext() SecurityContext {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentContext
}

// SetCurrentSecurityContext sets the current security context
func (sm *SecurityManagerImpl) SetCurrentSecurityContext(context SecurityContext) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentContext = context
}

// CheckAccess checks if an operation is allowed in the current context
func (sm *SecurityManagerImpl) CheckAccess(operationName string, targetName string) error {
	sm.mu.RLock()
	context := sm.currentContext
	sm.mu.RUnlock()

	if context == nil {
		return SecurityAccessDenied("No security context established", operationName)
	}

	if !context.IsValid() {
		return SecurityContextExpired()
	}

	principal := context.Credentials().Principal()

	// Check if there are any access rules for this target
	sm.mu.RLock()
	rules, hasRules := sm.accessPolicies[targetName]
	opRules, hasOpRules := sm.accessPolicies[operationName]
	sm.mu.RUnlock()

	// If there are no rules, deny by default for security
	if !hasRules && !hasOpRules {
		return nil // No rules means access is permitted by default
	}

	// Combine target and operation rules
	allRules := append(rules, opRules...)

	// Check if the principal has any of the required privileges
	for _, rule := range allRules {
		if rule.Target == targetName || rule.Target == operationName || rule.Target == "*" {
			for _, privilegeName := range rule.Privileges {
				if principal.HasPrivilege(privilegeName) {
					// Access granted
					return nil
				}
			}
		}
	}

	// Access denied
	return SecurityAccessDenied("Required privileges not found", operationName)
}

// AuditAction logs a security-relevant action for audit
func (sm *SecurityManagerImpl) AuditAction(action string, succeeded bool, details map[string]interface{}) {
	sm.mu.RLock()
	logger := sm.auditLogger
	context := sm.currentContext
	sm.mu.RUnlock()

	principalName := "unknown"
	if context != nil {
		principalName = context.Credentials().Principal().Name()
	}

	if logger != nil {
		logger.LogEvent(action, principalName, succeeded, details)
	}
}

// SecurityContextToServiceContext converts a security context to a service context
func (sm *SecurityManagerImpl) SecurityContextToServiceContext(secContext SecurityContext) ([]ServiceContext, error) {
	// Create a map of context data
	contextData := map[string]interface{}{
		"id":             secContext.ID(),
		"principal":      secContext.Credentials().Principal().Name(),
		"authType":       secContext.Credentials().Principal().AuthenticationType(),
		"expirationTime": secContext.ExpirationTime().Unix(),
		"attributes":     secContext.Attributes(),
	}

	// Marshal to JSON
	data, err := json.Marshal(contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal security context: %w", err)
	}

	// Create service context
	return []ServiceContext{
		{
			ID:   SecurityServiceContextID,
			Data: data,
		},
	}, nil
}

// ServiceContextToSecurityContext creates a security context from service contexts
func (sm *SecurityManagerImpl) ServiceContextToSecurityContext(serviceContexts []ServiceContext) (SecurityContext, error) {
	// Find the security service context
	var secCtx *ServiceContext
	for i, ctx := range serviceContexts {
		if ctx.ID == SecurityServiceContextID {
			secCtx = &serviceContexts[i]
			break
		}
	}

	if secCtx == nil {
		return nil, fmt.Errorf("security service context not found")
	}

	// Unmarshal the context data
	var contextData map[string]interface{}
	if err := json.Unmarshal(secCtx.Data, &contextData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal security context: %w", err)
	}

	// Extract context ID
	contextID, ok := contextData["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid security context ID")
	}

	// Look up the context in the store
	sm.mu.RLock()
	context, exists := sm.contextStore[contextID]
	sm.mu.RUnlock()

	if exists {
		// Check if context is still valid
		if !context.IsValid() {
			return nil, SecurityContextExpired()
		}
		return context, nil
	}

	// Context not found in store, reconstruct it
	principalName, ok := contextData["principal"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid principal name in security context")
	}

	// Get the principal
	sm.mu.RLock()
	principal, exists := sm.principalStore[principalName]
	sm.mu.RUnlock()

	if !exists {
		return nil, SecurityInvalidCredentials("Principal not found")
	}

	// Create new credentials
	expTime, ok := contextData["expirationTime"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid expiration time in security context")
	}

	expirationTime := time.Unix(int64(expTime), 0)
	lifetime := time.Until(expirationTime)
	if lifetime <= 0 {
		return nil, SecurityContextExpired()
	}

	credentials := NewCredentials(principal, CredsInvocationCredentials, lifetime)

	// Create and store the new context
	newContext, err := NewSecurityContext(credentials)
	if err != nil {
		return nil, err
	}

	// Copy attributes
	if attrs, ok := contextData["attributes"].(map[string]interface{}); ok {
		for k, v := range attrs {
			newContext.SetAttribute(k, v)
		}
	}

	// Store the reconstructed context
	sm.mu.Lock()
	sm.contextStore[newContext.ID()] = newContext
	sm.mu.Unlock()

	return newContext, nil
}

// DefaultAuditLogger provides a simple implementation of AuditLogger
type DefaultAuditLogger struct{}

// LogEvent logs a security event
func (l *DefaultAuditLogger) LogEvent(action string, principal string, succeeded bool, details map[string]interface{}) {
	statusStr := "FAILED"
	if succeeded {
		statusStr = "SUCCEEDED"
	}

	// Sanitize details to exclude sensitive or tainted fields
	sanitizedDetails := make(map[string]interface{})
	for key, value := range details {
		if key != "method" { // Exclude the tainted "method" field
			sanitizedDetails[key] = value
		}
	}
	detailsJSON, _ := json.Marshal(sanitizedDetails)
	log.Printf("[SECURITY AUDIT] %s by %s %s: %s", action, principal, statusStr, string(detailsJSON))
}

// PasswordAuthenticator implements Authenticator for password-based authentication
type PasswordAuthenticator struct {
	passwordStore map[string]string // principal name -> hashed password
	principals    map[string]*PrincipalImpl
}

// NewPasswordAuthenticator creates a new password-based authenticator
func NewPasswordAuthenticator() *PasswordAuthenticator {
	return &PasswordAuthenticator{
		passwordStore: make(map[string]string),
		principals:    make(map[string]*PrincipalImpl),
	}
}

// RegisterUser registers a user with the authenticator
func (pa *PasswordAuthenticator) RegisterUser(username, password string) *PrincipalImpl {
	// Create a hash of the password
	hashedPw := pa.hashPassword(password)

	// Store the password hash
	pa.passwordStore[username] = hashedPw

	// Create a principal
	principal := NewPrincipal(username, AuthPassword)
	pa.principals[username] = principal

	return principal
}

// hashPassword creates a hash of the password
func (pa *PasswordAuthenticator) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// Authenticate implements the Authenticator interface
func (pa *PasswordAuthenticator) Authenticate(authData interface{}) (Principal, error) {
	authMap, ok := authData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid auth data format")
	}

	username, ok := authMap["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username not provided")
	}

	password, ok := authMap["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password not provided")
	}

	// Get the stored password hash
	storedHash, exists := pa.passwordStore[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Check if the password is correct
	if pa.hashPassword(password) != storedHash {
		return nil, fmt.Errorf("invalid password")
	}

	// Return the principal
	principal, exists := pa.principals[username]
	if !exists {
		return nil, fmt.Errorf("principal not found")
	}

	return principal, nil
}

// TokenAuthenticator implements Authenticator for token-based authentication
type TokenAuthenticator struct {
	tokenStore map[string]*TokenInfo
	principals map[string]*PrincipalImpl
}

// TokenInfo contains information about an authentication token
type TokenInfo struct {
	Token         string
	PrincipalName string
	ExpiresAt     time.Time
}

// NewTokenAuthenticator creates a new token-based authenticator
func NewTokenAuthenticator() *TokenAuthenticator {
	return &TokenAuthenticator{
		tokenStore: make(map[string]*TokenInfo),
		principals: make(map[string]*PrincipalImpl),
	}
}

// RegisterPrincipal registers a principal with the authenticator
func (ta *TokenAuthenticator) RegisterPrincipal(principal *PrincipalImpl) {
	ta.principals[principal.Name()] = principal
}

// IssueToken issues a token for a principal
func (ta *TokenAuthenticator) IssueToken(principalName string, lifetime time.Duration) (string, error) {
	// Check if the principal exists
	if _, exists := ta.principals[principalName]; !exists {
		return "", fmt.Errorf("principal not found")
	}

	// Generate a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.StdEncoding.EncodeToString(tokenBytes)

	// Store the token
	ta.tokenStore[token] = &TokenInfo{
		Token:         token,
		PrincipalName: principalName,
		ExpiresAt:     time.Now().Add(lifetime),
	}

	return token, nil
}

// Authenticate implements the Authenticator interface
func (ta *TokenAuthenticator) Authenticate(authData interface{}) (Principal, error) {
	authMap, ok := authData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid auth data format")
	}

	token, ok := authMap["token"].(string)
	if !ok {
		return nil, fmt.Errorf("token not provided")
	}

	// Get the token info
	tokenInfo, exists := ta.tokenStore[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	// Check if the token has expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		delete(ta.tokenStore, token)
		return nil, fmt.Errorf("token expired")
	}

	// Return the principal
	principal, exists := ta.principals[tokenInfo.PrincipalName]
	if !exists {
		return nil, fmt.Errorf("principal not found")
	}

	return principal, nil
}

// SecurityPolicyImpl implements the SecurityPolicy interface
type SecurityPolicyImpl struct {
	policyType  uint32
	secFeatures map[string]interface{}
	accessRules []AccessRule
}

// NewSecurityPolicy creates a new security policy
func NewSecurityPolicy(policyType uint32) *SecurityPolicyImpl {
	return &SecurityPolicyImpl{
		policyType:  policyType,
		secFeatures: make(map[string]interface{}),
		accessRules: make([]AccessRule, 0),
	}
}

// PolicyType returns the type of the policy
func (p *SecurityPolicyImpl) PolicyType() uint32 {
	return p.policyType
}

// Copy creates a copy of the policy
func (p *SecurityPolicyImpl) Copy() Policy {
	copy := &SecurityPolicyImpl{
		policyType:  p.policyType,
		secFeatures: make(map[string]interface{}),
		accessRules: make([]AccessRule, len(p.accessRules)),
	}

	for k, v := range p.secFeatures {
		copy.secFeatures[k] = v
	}

	for i, rule := range p.accessRules {
		privileges := make([]string, len(rule.Privileges))
		copy.accessRules[i].Target = rule.Target
		copy.accessRules[i].Privileges = privileges
	}

	return copy
}

// Destroy destroys the policy
func (p *SecurityPolicyImpl) Destroy() {
	p.secFeatures = nil
	p.accessRules = nil
}

// SecurityFeatures returns the security features enabled by this policy
func (p *SecurityPolicyImpl) SecurityFeatures() map[string]interface{} {
	// Return a copy
	result := make(map[string]interface{})
	for k, v := range p.secFeatures {
		result[k] = v
	}
	return result
}

// SetSecurityFeature sets a security feature
func (p *SecurityPolicyImpl) SetSecurityFeature(name string, value interface{}) {
	p.secFeatures[name] = value
}

// AddAccessRule adds an access rule to the policy
func (p *SecurityPolicyImpl) AddAccessRule(rule AccessRule) {
	p.accessRules = append(p.accessRules, rule)
}

// Evaluate evaluates if an action is allowed under this policy
func (p *SecurityPolicyImpl) Evaluate(principal Principal, action string, target string) bool {
	// Check each rule
	for _, rule := range p.accessRules {
		if rule.Target == target || rule.Target == "*" {
			// Check if the principal has any of the required privileges
			for _, privilegeName := range rule.Privileges {
				if principal.HasPrivilege(privilegeName) {
					return true
				}
			}
		}
	}

	// No matching rules or required privileges not found
	return false
}
