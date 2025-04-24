// Package examples contains various examples of using the CORBA implementation
package examples

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// SecureCalculator is a secure CORBA calculator implementation
type SecureCalculator struct{}

// Dispatch implements the servant interface
func (c *SecureCalculator) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "add":
		if len(args) != 2 {
			return nil, fmt.Errorf("add requires two arguments")
		}
		a, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("first argument must be a number")
		}
		b, ok := args[1].(float64)
		if !ok {
			return nil, fmt.Errorf("second argument must be a number")
		}
		return a + b, nil

	case "subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("subtract requires two arguments")
		}
		a, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("first argument must be a number")
		}
		b, ok := args[1].(float64)
		if !ok {
			return nil, fmt.Errorf("second argument must be a number")
		}
		return a - b, nil

	case "multiply":
		// This is a protected operation requiring admin rights
		if len(args) != 2 {
			return nil, fmt.Errorf("multiply requires two arguments")
		}
		a, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("first argument must be a number")
		}
		b, ok := args[1].(float64)
		if !ok {
			return nil, fmt.Errorf("second argument must be a number")
		}
		return a * b, nil

	case "divide":
		// This is a protected operation requiring admin rights
		if len(args) != 2 {
			return nil, fmt.Errorf("divide requires two arguments")
		}
		a, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("first argument must be a number")
		}
		b, ok := args[1].(float64)
		if !ok {
			return nil, fmt.Errorf("second argument must be a number")
		}
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}

// setupCertificates generates or loads SSL/TLS certificates for secure communication
func setupCertificates() (certFile, keyFile string, err error) {
	// Create a temp directory for certificates if it doesn't exist
	tempDir := filepath.Join(os.TempDir(), "gocorba-certs")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create certificates directory: %w", err)
	}

	certFile = filepath.Join(tempDir, "server.crt")
	keyFile = filepath.Join(tempDir, "server.key")

	// Check if certificates already exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			// Both files exist, reuse them
			return certFile, keyFile, nil
		}
	}

	// Generate new self-signed certificates
	certPEM, keyPEM, err := corba.GenerateSelfSignedCert("GoCorba Security Example")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Write certificate and key to files
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write certificate file: %w", err)
	}

	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write key file: %w", err)
	}

	return certFile, keyFile, nil
}

// RunSecureServer demonstrates running a server with the Security Service
func RunSecureServer() {
	// Initialize the ORB
	orb := corba.Init()

	// Setup SSL/TLS certificates
	certFile, keyFile, err := setupCertificates()
	if err != nil {
		fmt.Printf("Failed to setup certificates: %v\n", err)
		os.Exit(1)
	}

	// Create a secure server
	server, err := orb.NewSecureServer("localhost", 8099, certFile, keyFile)
	if err != nil {
		fmt.Printf("Failed to create secure server: %v\n", err)
		os.Exit(1)
	}

	// Create the security manager
	securityManager := corba.NewSecurityManager()

	// Setup password-based authentication
	passwordAuth := corba.NewPasswordAuthenticator()

	// Register users
	adminUser := passwordAuth.RegisterUser("admin", "admin123")
	guestUser := passwordAuth.RegisterUser("guest", "guest123")

	// Add privileges to users
	adminUser.AddPrivilege(corba.Privilege{Name: "admin", Rights: []string{"read", "write", "execute"}})
	adminUser.AddPrivilege(corba.Privilege{Name: "calculator", Rights: []string{"add", "subtract", "multiply", "divide"}})
	guestUser.AddPrivilege(corba.Privilege{Name: "calculator", Rights: []string{"add", "subtract"}})

	// Register principals with security manager
	securityManager.RegisterPrincipal(adminUser)
	securityManager.RegisterPrincipal(guestUser)

	// Register authenticator with security manager
	securityManager.RegisterAuthenticator(corba.AuthPassword, passwordAuth)

	// Create security service interceptor
	secInterceptor := corba.NewSecurityServiceInterceptor(securityManager, true)

	// Define access control rules for calculator methods
	secInterceptor.RequirePrivilege("multiply", "admin")
	secInterceptor.RequirePrivilege("divide", "admin")

	// Register authentication interceptor (processes username/password)
	authInterceptor := corba.NewAuthenticationInterceptor(securityManager)
	orb.RegisterServerRequestInterceptor(authInterceptor)

	// Register security service interceptor (enforces access control)
	orb.RegisterServerRequestInterceptor(secInterceptor)

	// Register audit interceptor
	auditInterceptor := corba.NewSecurityAuditInterceptor(nil) // Use default logger
	orb.RegisterServerRequestInterceptor(auditInterceptor)

	// Register calculator servant
	calculator := &SecureCalculator{}
	if err := server.RegisterServant("SecureCalculator", calculator); err != nil {
		fmt.Printf("Failed to register calculator servant: %v\n", err)
		os.Exit(1)
	}

	// Start the server
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Secure CORBA server running with Security Service enabled")
	fmt.Println("Press Ctrl+C to stop")

	// Keep the server running
	select {}
}

// RunSecureClient demonstrates using a client with the Security Service
func RunSecureClient() {
	// Initialize the ORB
	orb := corba.Init()

	// Setup SSL/TLS certificates
	certFile, _, err := setupCertificates()
	if err != nil {
		fmt.Printf("Failed to setup certificates: %v\n", err)
		os.Exit(1)
	}

	// Create a secure client with the certificate
	client, err := orb.NewSecureClient(certFile)
	if err != nil {
		fmt.Printf("Failed to create secure client: %v\n", err)
		os.Exit(1)
	}

	// Connect to the server
	err = client.Connect("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to secure server")

	// First try basic operations as a guest user
	fmt.Println("\n--- Guest User Session ---")
	err = client.Authenticate(map[string]interface{}{
		"method":   corba.AuthPassword,
		"username": "guest",
		"password": "guest123",
	})

	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Authenticated as guest")

	// Get a reference to the calculator
	_, err = client.GetObject("SecureCalculator", "localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to get calculator reference: %v\n", err)
		os.Exit(1)
	}

	// Try addition (allowed for guest)
	result, err := client.InvokeSecureMethod("SecureCalculator", "add", "localhost", 8099, 10.5, 20.7)
	if err != nil {
		fmt.Printf("Add failed: %v\n", err)
	} else {
		fmt.Printf("10.5 + 20.7 = %v\n", result)
	}

	// Try multiplication (should be denied for guest)
	result, err = client.InvokeSecureMethod("SecureCalculator", "multiply", "localhost", 8099, 10.5, 20.7)
	if err != nil {
		fmt.Printf("Multiply failed as expected: %v\n", err)
	} else {
		fmt.Printf("10.5 * 20.7 = %v (Unexpectedly succeeded!)\n", result)
	}

	// Now authenticate as admin
	fmt.Println("\n--- Admin User Session ---")
	err = client.Authenticate(map[string]interface{}{
		"method":   corba.AuthPassword,
		"username": "admin",
		"password": "admin123",
	})

	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Authenticated as admin")

	// Try addition (allowed for admin)
	result, err = client.InvokeSecureMethod("SecureCalculator", "add", "localhost", 8099, 10.5, 20.7)
	if err != nil {
		fmt.Printf("Add failed: %v\n", err)
	} else {
		fmt.Printf("10.5 + 20.7 = %v\n", result)
	}

	// Try multiplication (should be allowed for admin)
	result, err = client.InvokeSecureMethod("SecureCalculator", "multiply", "localhost", 8099, 10.5, 20.7)
	if err != nil {
		fmt.Printf("Multiply failed: %v\n", err)
	} else {
		fmt.Printf("10.5 * 20.7 = %v\n", result)
	}

	// Try division (should be allowed for admin)
	result, err = client.InvokeSecureMethod("SecureCalculator", "divide", "localhost", 8099, 100.0, 5.0)
	if err != nil {
		fmt.Printf("Divide failed: %v\n", err)
	} else {
		fmt.Printf("100.0 / 5.0 = %v\n", result)
	}

	// Disconnect from server
	if err := client.Disconnect("localhost", 8099); err != nil {
		fmt.Printf("Warning: Error during disconnect: %v\n", err)
	}

	fmt.Println("Secure CORBA client session completed")
}

// RunSecurityDemo runs both the secure server and client
func RunSecurityDemo() {
	// Start server in a separate goroutine
	go func() {
		RunSecureServer()
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Run client
	RunSecureClient()
}
