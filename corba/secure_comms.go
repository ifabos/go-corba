// Package corba provides a CORBA implementation in Go
package corba

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// SecureIIOP represents a secure IIOP connection
type SecureIIOP struct {
	tlsConfig *tls.Config
	host      string
	port      int
}

// NewSecureIIOP creates a new secure IIOP connection
func NewSecureIIOP(host string, port int, tlsConfig *tls.Config) *SecureIIOP {
	return &SecureIIOP{
		tlsConfig: tlsConfig,
		host:      host,
		port:      port,
	}
}

// Connect establishes a secure connection to a CORBA server
func (s *SecureIIOP) Connect() (net.Conn, error) {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	return tls.Dial("tcp", address, s.tlsConfig)
}

// Listen starts a secure IIOP server listener
func (s *SecureIIOP) Listen() (net.Listener, error) {
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	return tls.Listen("tcp", address, s.tlsConfig)
}

// SecureServer represents a CORBA server with security features
type SecureServer struct {
	*Server
	secureIIOP *SecureIIOP
}

// NewSecureServer creates a new secure server
func (orb *ORB) NewSecureServer(host string, port int, certFile, keyFile string) (*SecureServer, error) {
	server, err := orb.CreateServer(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate and key: %w", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	secureIIOP := NewSecureIIOP(host, port, tlsConfig)

	return &SecureServer{
		Server:     server,
		secureIIOP: secureIIOP,
	}, nil
}

// NewSecureServerWithConfig creates a new secure server with a custom TLS configuration
func (orb *ORB) NewSecureServerWithConfig(host string, port int, tlsConfig *tls.Config) (*SecureServer, error) {
	server, err := orb.CreateServer(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	secureIIOP := NewSecureIIOP(host, port, tlsConfig)

	return &SecureServer{
		Server:     server,
		secureIIOP: secureIIOP,
	}, nil
}

// Start starts the secure server
func (s *SecureServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// Start secure IIOP listener
	listener, err := s.secureIIOP.Listen()
	if err != nil {
		return fmt.Errorf("failed to start secure IIOP listener: %w", err)
	}

	s.listener = listener
	s.running = true

	fmt.Printf("Secure CORBA server listening on %s (TLS/SSL enabled)\n", listener.Addr())

	// Handle incoming connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if server was stopped
				s.mu.RLock()
				running := s.running
				s.mu.RUnlock()
				if !running {
					return
				}
				fmt.Printf("Error accepting secure connection: %v\n", err)
				continue
			}

			// Handle connection in a new goroutine
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// SecureClient represents a CORBA client with security features
type SecureClient struct {
	*Client
	secureIIOP      *SecureIIOP
	securityManager *SecurityManagerImpl
	securityContext SecurityContext
}

// NewSecureClient creates a new secure client
func (orb *ORB) NewSecureClient(trustFile string) (*SecureClient, error) {
	client := orb.CreateClient()

	// Load trusted CA certificate
	certPool := x509.NewCertPool()
	var tlsConfig *tls.Config

	if trustFile != "" {
		// Load CA certificate from file
		if !certPool.AppendCertsFromPEM([]byte(trustFile)) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}

		tlsConfig = &tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		}
	} else {
		// Use system default CA certificates
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	// Create security manager
	securityManager := NewSecurityManager()

	return &SecureClient{
		Client:          client,
		secureIIOP:      NewSecureIIOP("", 0, tlsConfig),
		securityManager: securityManager,
	}, nil
}

// NewSecureClientWithConfig creates a new secure client with a custom TLS configuration
func (orb *ORB) NewSecureClientWithConfig(tlsConfig *tls.Config) (*SecureClient, error) {
	client := orb.CreateClient()
	securityManager := NewSecurityManager()

	return &SecureClient{
		Client:          client,
		secureIIOP:      NewSecureIIOP("", 0, tlsConfig),
		securityManager: securityManager,
		securityContext: nil,
	}, nil
}

// Connect securely connects to a CORBA server
func (c *SecureClient) Connect(host string, port int) error {
	// Update SecureIIOP with host and port
	c.secureIIOP.host = host
	c.secureIIOP.port = port

	// Establish secure connection
	conn, err := c.secureIIOP.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect securely: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connections == nil {
		c.connections = make(map[string]net.Conn)
	}

	address := fmt.Sprintf("%s:%d", host, port)
	c.connections[address] = conn
	return nil
}

// Authenticate authenticates the client
func (c *SecureClient) Authenticate(authData interface{}) error {
	// Authenticate using security manager
	credentials, err := c.securityManager.Authenticate(authData)
	if err != nil {
		return err
	}

	// Create security context
	context, err := c.securityManager.CreateSecurityContext(credentials)
	if err != nil {
		return err
	}

	// Set current security context
	c.securityContext = context
	c.securityManager.SetCurrentSecurityContext(context)

	return nil
}

// InvokeSecureMethod invokes a method with the current security context
func (c *SecureClient) InvokeSecureMethod(objectName string, methodName string, serverHost string, serverPort int, args ...interface{}) (interface{}, error) {
	if c.securityContext == nil {
		return nil, SecurityInvalidCredentials("Not authenticated")
	}

	// Convert security context to service contexts
	serviceContexts, err := c.securityManager.SecurityContextToServiceContext(c.securityContext)
	if err != nil {
		return nil, err
	}

	// Add service contexts to RequestInfo
	reqInfo := &RequestInfo{
		Operation:       methodName,
		ObjectKey:       objectName,
		Arguments:       args,
		ServiceContexts: serviceContexts,
	}

	// Call client request interceptors - SendRequest
	interceptors := c.orb.GetInterceptorRegistry().GetClientRequestInterceptors()
	for _, interceptor := range interceptors {
		if err := interceptor.SendRequest(reqInfo); err != nil {
			return nil, err
		}
	}

	// Update args with any modifications from interceptors
	args = reqInfo.Arguments

	// Invoke method
	return c.InvokeMethod(objectName, methodName, serverHost, serverPort, args...)
}

// GenerateSelfSignedCert generates a self-signed certificate for testing
func GenerateSelfSignedCert(organization string) (certPEM, keyPEM []byte, err error) {
	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year validity

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
			CommonName:   "CORBA SSL Certificate",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return certPEM, keyPEM, nil
}

// LoadTLSConfig loads a TLS configuration from certificate and key files
func LoadTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate and key: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Load CA certificate if provided
	if caFile != "" {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(caFile)) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsConfig.RootCAs = certPool
	}

	return tlsConfig, nil
}
