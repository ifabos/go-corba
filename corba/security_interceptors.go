// Package corba provides a CORBA implementation in Go
package corba

import (
	"encoding/json"
	"log"
	"time"
)

// SecurityServiceInterceptor implements a server interceptor that integrates with the security service
type SecurityServiceInterceptor struct {
	secManager     *SecurityManagerImpl
	requiresAuth   bool
	accessPolicies map[string][]string // Method -> required privileges
}

// NewSecurityServiceInterceptor creates a new security service interceptor
func NewSecurityServiceInterceptor(secManager *SecurityManagerImpl, requiresAuth bool) *SecurityServiceInterceptor {
	return &SecurityServiceInterceptor{
		secManager:     secManager,
		requiresAuth:   requiresAuth,
		accessPolicies: make(map[string][]string),
	}
}

// Name returns the name of the interceptor
func (i *SecurityServiceInterceptor) Name() string {
	return "SecurityServiceInterceptor"
}

// RequirePrivilege specifies that a method requires a specific privilege
func (i *SecurityServiceInterceptor) RequirePrivilege(method string, privilege string) {
	if privileges, ok := i.accessPolicies[method]; ok {
		i.accessPolicies[method] = append(privileges, privilege)
	} else {
		i.accessPolicies[method] = []string{privilege}
	}

	// Also register this with the security manager's access control
	i.secManager.AddAccessRule(method, AccessRule{
		Target:     method,
		Privileges: []string{privilege},
	})
}

// ReceiveRequest is called when a request is received
func (i *SecurityServiceInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Check if authentication is required
	if !i.requiresAuth {
		return nil
	}

	// Find security service context
	var secCtx *ServiceContext
	for idx, ctx := range info.ServiceContexts {
		if ctx.ID == SecurityServiceContextID {
			secCtx = &info.ServiceContexts[idx]
			break
		}
	}

	if secCtx == nil {
		i.secManager.AuditAction("Authentication", false, map[string]interface{}{
			"operation": info.Operation,
			"reason":    "No security context provided",
		})
		return SecurityInvalidCredentials("No security context provided")
	}

	// Extract security context from service context
	context, err := i.secManager.ServiceContextToSecurityContext(info.ServiceContexts)
	if err != nil {
		i.secManager.AuditAction("Authentication", false, map[string]interface{}{
			"operation": info.Operation,
			"reason":    err.Error(),
		})
		return err
	}

	// Set as current security context
	i.secManager.SetCurrentSecurityContext(context)

	// Check access control
	if err := i.secManager.CheckAccess(info.Operation, info.ObjectKey); err != nil {
		i.secManager.AuditAction("AccessControl", false, map[string]interface{}{
			"operation": info.Operation,
			"object":    info.ObjectKey,
			"principal": context.Credentials().Principal().Name(),
			"reason":    err.Error(),
		})
		return err
	}

	// Successful access
	i.secManager.AuditAction("AccessControl", true, map[string]interface{}{
		"operation": info.Operation,
		"object":    info.ObjectKey,
		"principal": context.Credentials().Principal().Name(),
	})

	return nil
}

// SendReply is called before sending a reply
func (i *SecurityServiceInterceptor) SendReply(info *RequestInfo) error {
	// Get current security context
	context := i.secManager.GetSecurityContext()
	if context == nil {
		return nil
	}

	// Add security context to reply
	serviceContexts, err := i.secManager.SecurityContextToServiceContext(context)
	if err != nil {
		return err
	}

	// Add to service contexts
	info.ServiceContexts = append(info.ServiceContexts, serviceContexts...)

	return nil
}

// SendException is called when an exception is sent
func (i *SecurityServiceInterceptor) SendException(info *RequestInfo, ex Exception) error {
	// Audit security exceptions
	if _, ok := ex.(*SecurityException); ok {
		context := i.secManager.GetSecurityContext()
		principalName := "unknown"
		if context != nil {
			principalName = context.Credentials().Principal().Name()
		}

		i.secManager.AuditAction("SecurityException", false, map[string]interface{}{
			"operation": info.Operation,
			"object":    info.ObjectKey,
			"principal": principalName,
			"exception": ex.Error(),
		})
	}

	return nil
}

// ClientSecurityServiceInterceptor implements a client interceptor that integrates with the security service
type ClientSecurityServiceInterceptor struct {
	secManager      *SecurityManagerImpl
	securityContext SecurityContext
}

// NewClientSecurityServiceInterceptor creates a new client security service interceptor
func NewClientSecurityServiceInterceptor(secManager *SecurityManagerImpl) *ClientSecurityServiceInterceptor {
	return &ClientSecurityServiceInterceptor{
		secManager: secManager,
	}
}

// Name returns the name of the interceptor
func (i *ClientSecurityServiceInterceptor) Name() string {
	return "ClientSecurityServiceInterceptor"
}

// SetSecurityContext sets the security context to use for outgoing requests
func (i *ClientSecurityServiceInterceptor) SetSecurityContext(context SecurityContext) {
	i.securityContext = context
}

// SendRequest is called before sending a request
func (i *ClientSecurityServiceInterceptor) SendRequest(info *RequestInfo) error {
	if i.securityContext == nil {
		return nil
	}

	// Convert security context to service contexts
	serviceContexts, err := i.secManager.SecurityContextToServiceContext(i.securityContext)
	if err != nil {
		return err
	}

	// Add to outgoing request
	info.ServiceContexts = append(info.ServiceContexts, serviceContexts...)

	return nil
}

// ReceiveReply is called when a reply is received
func (i *ClientSecurityServiceInterceptor) ReceiveReply(info *RequestInfo) error {
	// Find security service context
	var secCtx *ServiceContext
	for idx, ctx := range info.ServiceContexts {
		if ctx.ID == SecurityServiceContextID {
			secCtx = &info.ServiceContexts[idx]
			break
		}
	}

	if secCtx == nil {
		return nil
	}

	// Extract security context from service context
	context, err := i.secManager.ServiceContextToSecurityContext(info.ServiceContexts)
	if err != nil {
		log.Printf("[CLIENT-SECURITY] Failed to extract security context: %v", err)
		return nil
	}

	// Update current security context
	i.securityContext = context
	i.secManager.SetCurrentSecurityContext(context)

	return nil
}

// ReceiveException is called when an exception is received
func (i *ClientSecurityServiceInterceptor) ReceiveException(info *RequestInfo, ex Exception) error {
	// Handle security exceptions
	if secEx, ok := ex.(*SecurityException); ok {
		log.Printf("[CLIENT-SECURITY] Received security exception: %s", secEx.Error())
	}

	return nil
}

// ReceiveOther is called for other outcomes
func (i *ClientSecurityServiceInterceptor) ReceiveOther(info *RequestInfo) error {
	return nil
}

// SecurityAuditInterceptor implements an interceptor that logs security-relevant events
type SecurityAuditInterceptor struct {
	auditLogger AuditLogger
}

// NewSecurityAuditInterceptor creates a new security audit interceptor
func NewSecurityAuditInterceptor(auditLogger AuditLogger) *SecurityAuditInterceptor {
	if auditLogger == nil {
		auditLogger = &DefaultAuditLogger{}
	}

	return &SecurityAuditInterceptor{
		auditLogger: auditLogger,
	}
}

// Name returns the name of the interceptor
func (i *SecurityAuditInterceptor) Name() string {
	return "SecurityAuditInterceptor"
}

// ReceiveRequest is called when a request is received
func (i *SecurityAuditInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Extract principal from security context if present
	principalName := "anonymous"
	for _, ctx := range info.ServiceContexts {
		if ctx.ID == SecurityServiceContextID {
			var contextData map[string]interface{}
			if err := json.Unmarshal(ctx.Data, &contextData); err == nil {
				if p, ok := contextData["principal"].(string); ok {
					principalName = p
				}
			}
			break
		}
	}

	// Log the request
	i.auditLogger.LogEvent(
		"Request",
		principalName,
		true,
		map[string]interface{}{
			"operation":  info.Operation,
			"object":     info.ObjectKey,
			"timestamp":  time.Now().Format(time.RFC3339),
			"request_id": info.RequestID,
		},
	)

	return nil
}

// SendReply is called before sending a reply
func (i *SecurityAuditInterceptor) SendReply(info *RequestInfo) error {
	// Extract principal from security context if present
	principalName := "anonymous"
	for _, ctx := range info.ServiceContexts {
		if ctx.ID == SecurityServiceContextID {
			var contextData map[string]interface{}
			if err := json.Unmarshal(ctx.Data, &contextData); err == nil {
				if p, ok := contextData["principal"].(string); ok {
					principalName = p
				}
			}
			break
		}
	}

	// Log the successful operation
	i.auditLogger.LogEvent(
		"OperationSuccess",
		principalName,
		true,
		map[string]interface{}{
			"operation":  info.Operation,
			"object":     info.ObjectKey,
			"timestamp":  time.Now().Format(time.RFC3339),
			"request_id": info.RequestID,
		},
	)

	return nil
}

// SendException is called when an exception is sent
func (i *SecurityAuditInterceptor) SendException(info *RequestInfo, ex Exception) error {
	// Extract principal from security context if present
	principalName := "anonymous"
	for _, ctx := range info.ServiceContexts {
		if ctx.ID == SecurityServiceContextID {
			var contextData map[string]interface{}
			if err := json.Unmarshal(ctx.Data, &contextData); err == nil {
				if p, ok := contextData["principal"].(string); ok {
					principalName = p
				}
			}
			break
		}
	}

	// Log the exception
	i.auditLogger.LogEvent(
		"OperationException",
		principalName,
		false,
		map[string]interface{}{
			"operation":  info.Operation,
			"object":     info.ObjectKey,
			"timestamp":  time.Now().Format(time.RFC3339),
			"request_id": info.RequestID,
			"exception":  ex.Error(),
		},
	)

	return nil
}

// AuthenticationInterceptor implements an interceptor for simple authentication
type AuthenticationInterceptor struct {
	secManager *SecurityManagerImpl
}

// NewAuthenticationInterceptor creates a new authentication interceptor
func NewAuthenticationInterceptor(secManager *SecurityManagerImpl) *AuthenticationInterceptor {
	return &AuthenticationInterceptor{
		secManager: secManager,
	}
}

// Name returns the name of the interceptor
func (i *AuthenticationInterceptor) Name() string {
	return "AuthenticationInterceptor"
}

// ReceiveRequest is called when a request is received
func (i *AuthenticationInterceptor) ReceiveRequest(info *RequestInfo) error {
	// Look for basic authentication in service contexts
	var username, password string
	var found bool

	for _, ctx := range info.ServiceContexts {
		if ctx.ID == 0x42415349 { // "BASI" for basic auth
			authData := string(ctx.Data)
			parts := make([]string, 0)

			// Simple parsing of username:password
			for _, part := range authData {
				if part == ':' {
					parts = append(parts, username)
					username = ""
					continue
				}
				username += string(part)
			}

			if len(parts) > 0 {
				username = parts[0]
				password = authData[len(username)+1:]
				found = true
				break
			}
		}
	}

	if !found {
		return nil // No authentication data found, continue
	}

	// Authenticate
	// Pass only non-sensitive data to Authenticate
	creds, err := i.secManager.Authenticate(map[string]interface{}{
		"method":   AuthPassword,
		"username": username,
	})

	if err != nil {
		return err
	}

	// Create security context
	context, err := i.secManager.CreateSecurityContext(creds)
	if err != nil {
		return err
	}

	// Set as current security context
	i.secManager.SetCurrentSecurityContext(context)

	// Add context to service contexts for downstream interceptors
	serviceContexts, err := i.secManager.SecurityContextToServiceContext(context)
	if err != nil {
		return err
	}

	// Add to service contexts
	info.ServiceContexts = append(info.ServiceContexts, serviceContexts...)

	return nil
}

// SendReply is called before sending a reply
func (i *AuthenticationInterceptor) SendReply(info *RequestInfo) error {
	return nil
}

// SendException is called when an exception is sent
func (i *AuthenticationInterceptor) SendException(info *RequestInfo, ex Exception) error {
	return nil
}
