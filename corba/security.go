// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
	"time"
)

// Security constants
const (
	// Security service context ID
	SecurityServiceContextID uint32 = 0x53454300 // "SEC"

	// Security service attribute names
	SecurityAttrPrincipalName   = "PrincipalName"
	SecurityAttrPrincipalAuth   = "PrincipalAuth"
	SecurityAttrPrivileges      = "Privileges"
	SecurityAttrAuthMethod      = "AuthMethod"
	SecurityAttrSessionID       = "SessionID"
	SecurityAttrCredentialsType = "CredentialsType"
	SecurityAttrAuthExpiry      = "AuthExpiry"
	SecurityAttrDelegationMode  = "DelegationMode"

	// Security service context names
	SecurityCtxAccessControl  = "AccessControl"
	SecurityCtxAudit          = "Audit"
	SecurityCtxAuthentication = "Authentication"
	SecurityCtxSecureComms    = "SecureComms"
	SecurityCtxNonRepudiation = "NonRepudiation"
)

// SecurityLevel represents the security compliance level
type SecurityLevel int

const (
	// SecurityLevelNone represents no security
	SecurityLevelNone SecurityLevel = iota
	// SecurityLevelBasic represents basic security
	SecurityLevelBasic
	// SecurityLevelIdentity represents identity-based security
	SecurityLevelIdentity
	// SecurityLevelPrivacy represents security with privacy features
	SecurityLevelPrivacy
	// SecurityLevelIntegrity represents security with integrity checks
	SecurityLevelIntegrity
	// SecurityLevelConfidentiality represents confidentiality security
	SecurityLevelConfidentiality
)

// AuthenticationMethod represents the method used to authenticate a principal
type AuthenticationMethod int

const (
	// AuthNone represents no authentication
	AuthNone AuthenticationMethod = iota
	// AuthPassword represents password authentication
	AuthPassword
	// AuthCertificate represents certificate-based authentication
	AuthCertificate
	// AuthToken represents token-based authentication
	AuthToken
	// AuthKerberos represents Kerberos-based authentication
	AuthKerberos
)

// CredentialsType represents the type of credentials
type CredentialsType int

const (
	// CredsInvocationCredentials represents credentials for invocation
	CredsInvocationCredentials CredentialsType = iota
	// CredsAcceptingCredentials represents credentials for accepting invocations
	CredsAcceptingCredentials
)

// DelegationMode represents the delegation mode
type DelegationMode int

const (
	// DelegationNone represents no delegation
	DelegationNone DelegationMode = iota
	// DelegationSimple represents simple delegation
	DelegationSimple
	// DelegationComposite represents composite delegation
	DelegationComposite
)

// SecurityException represents a security-related CORBA system exception
type SecurityException struct {
	*SystemException
	Reason string
}

// NewSecurityException creates a new security exception
func NewSecurityException(reason string, minor uint32, completed CompletionStatus) *SecurityException {
	return &SecurityException{
		SystemException: NewCORBASystemException("NO_PERMISSION", minor, completed),
		Reason:          reason,
	}
}

// Error implements the error interface for SecurityException
func (e *SecurityException) Error() string {
	return fmt.Sprintf("CORBA Security Exception: %s (reason: %s, minor code: %d, completion status: %v)",
		e.Name(), e.Reason, e.Minor(), e.Completed())
}

// Security related system exceptions defined by CORBA
var (
	// SecurityInvalidCredentials indicates invalid credentials
	SecurityInvalidCredentials = func(reason string) Exception {
		return NewSecurityException(reason, 1, CompletionStatusNo)
	}

	// SecurityInvalidToken indicates invalid security token
	SecurityInvalidToken = func(reason string) Exception {
		return NewSecurityException(reason, 2, CompletionStatusNo)
	}

	// SecurityAuthenticationFailed indicates authentication failure
	SecurityAuthenticationFailed = func(reason string) Exception {
		return NewSecurityException(reason, 3, CompletionStatusNo)
	}

	// SecurityAccessDenied indicates access denied
	SecurityAccessDenied = func(reason string, op string) Exception {
		return NewSecurityException(fmt.Sprintf("%s: operation %s", reason, op), 4, CompletionStatusNo)
	}

	// SecurityInvalidPolicy indicates invalid security policy
	SecurityInvalidPolicy = func(reason string) Exception {
		return NewSecurityException(reason, 5, CompletionStatusNo)
	}

	// SecurityContextExpired indicates security context expired
	SecurityContextExpired = func() Exception {
		return NewSecurityException("Security context expired", 6, CompletionStatusNo)
	}
)

// SecurityPrincipal represents a security principal (user or system entity)
type Principal interface {
	// Name returns the principal name
	Name() string

	// AuthenticationType returns the authentication method
	AuthenticationType() AuthenticationMethod

	// PrivilegeList returns the privileges assigned to the principal
	PrivilegeList() []Privilege

	// HasPrivilege checks if the principal has a specific privilege
	HasPrivilege(privilegeName string) bool

	Roles() []string

	IsInRole(string) bool
}

// Privilege represents a security privilege
type Privilege struct {
	Name   string
	Rights []string
}

// Credentials represents security credentials
type Credentials interface {
	// Type returns the credentials type
	Type() CredentialsType

	// Principal returns the principal associated with the credentials
	Principal() Principal

	// IsValid checks if the credentials are valid
	IsValid() bool

	// Refresh refreshes the credentials
	Refresh() error

	// ExpirationTime returns the expiration time of the credentials
	ExpirationTime() time.Time
}

// SecurityContext represents a security context
type SecurityContext interface {
	// ID returns the security context ID
	ID() string

	// Credentials returns the credentials associated with this context
	Credentials() Credentials

	// IsValid checks if the context is valid
	IsValid() bool

	// Refresh refreshes the security context
	Refresh() error

	// ExpirationTime returns the expiration time of the security context
	ExpirationTime() time.Time

	// Attributes returns security context attributes
	Attributes() map[string]interface{}
}

// SecurityManager is the central point for security management
type SecurityManager interface {
	// Authenticate authenticates a principal and returns credentials
	Authenticate(authData interface{}) (Credentials, error)

	// CreateSecurityContext creates a security context from credentials
	CreateSecurityContext(creds Credentials) (SecurityContext, error)

	// GetSecurityContext returns the current security context
	GetSecurityContext() SecurityContext

	// CheckAccess checks if an operation is allowed in the current context
	CheckAccess(operationName string, targetName string) error

	// AuditAction logs a security-relevant action for audit
	AuditAction(action string, succeeded bool, details map[string]interface{})

	// SecurityContextToServiceContext converts a security context to a service context
	SecurityContextToServiceContext(secContext SecurityContext) ([]ServiceContext, error)

	// ServiceContextToSecurityContext creates a security context from service contexts
	ServiceContextToSecurityContext(serviceContexts []ServiceContext) (SecurityContext, error)
}

// SecurityPolicy defines security policies
type SecurityPolicy interface {
	Policy

	// SecurityFeatures returns the security features enabled by this policy
	SecurityFeatures() map[string]interface{}

	// Evaluate evaluates if an action is allowed under this policy
	Evaluate(principal Principal, action string, target string) bool
}

// Initialize standard system exceptions for security
func init() {
	// Register security exceptions with the exception system
	// This will allow them to be properly identified when received over the wire
	RegisterException("IDL:omg.org/Security/InvalidCredentials:1.0", &SecurityException{})
}

// BasicPrincipal provides a simple implementation of Principal
type BasicPrincipal struct {
	PrincipalName        string
	PrincipalRoles       []string
	AuthenticationMethod AuthenticationMethod
	Privileges           []Privilege
}

// IsInRole checks if this principal has a specific role
func (p *BasicPrincipal) IsInRole(role string) bool {
	for _, r := range p.PrincipalRoles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPrivilege checks if the principal has a specific privilege
func (p *BasicPrincipal) HasPrivilege(privilegeName string) bool {
	for _, p := range p.Privileges {
		for _, r := range p.Rights {
			if r == privilegeName {
				return true
			}
		}
	}
	return false
}

// Name returns the principal name
func (p *BasicPrincipal) Name() string {
	return p.PrincipalName
}

// AuthenticationType returns the authentication method
func (p *BasicPrincipal) AuthenticationType() AuthenticationMethod {
	return p.AuthenticationMethod
}

// PrivilegeList returns the privileges assigned to the principal
func (p *BasicPrincipal) PrivilegeList() []Privilege {
	return append([]Privilege{}, p.Privileges...)
}

// Roles returns the roles assigned to the principal
func (p *BasicPrincipal) Roles() []string {
	return p.PrincipalRoles
}
