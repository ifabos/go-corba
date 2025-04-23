package examples

import (
	"fmt"
	"log"
	"sync"

	"github.com/ifabos/go-corba/corba"
)

// AdvancedClientExample demonstrates working with multiple servers
// and handling error conditions gracefully
func AdvancedClientExample() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a CORBA client
	client := orb.CreateClient()

	// Define multiple servers to connect to
	servers := []struct {
		host string
		port int
		name string
	}{
		{"primary.example.com", 8099, "Primary"},
		{"secondary.example.com", 8099, "Secondary"},
		{"backup.example.com", 8099, "Backup"},
	}

	// Try to connect to each server with timeout and fallback handling
	var connectedServer struct {
		host string
		port int
		name string
	}

	connected := false
	for _, server := range servers {
		fmt.Printf("Trying to connect to %s server (%s:%d)...\n",
			server.name, server.host, server.port)

		// In a production environment, you would implement a timeout here
		err := client.Connect(server.host, server.port)
		if err == nil {
			connected = true
			connectedServer = server
			fmt.Printf("Successfully connected to %s server\n", server.name)
			break
		}

		fmt.Printf("Failed to connect to %s server: %v\n", server.name, err)
		// In a real application, you might want to add a delay before retrying
		// time.Sleep(1 * time.Second)
	}

	if !connected {
		log.Fatal("Failed to connect to any CORBA server")
	}

	// Demonstrate parallel method invocations
	var wg sync.WaitGroup
	results := make(chan string, 3)
	errors := make(chan error, 3)

	// Get object reference
	stockService, err := client.GetObject("StockQuoteService",
		connectedServer.host, connectedServer.port)
	if err != nil {
		log.Fatalf("Failed to get StockQuoteService reference: %v", err)
	}

	// Invoke multiple methods in parallel
	stocks := []string{"AAPL", "MSFT", "GOOG"}
	for _, symbol := range stocks {
		wg.Add(1)
		go func(stockSymbol string) {
			defer wg.Done()

			// Add a timeout context (conceptual - not fully implemented in current SDK)
			ctx := corba.NewContext()
			ctx.Set("timeout_ms", 5000)

			fmt.Printf("Requesting quote for %s\n", stockSymbol)
			result, err := stockService.Invoke("getQuote", stockSymbol)
			if err != nil {
				errors <- fmt.Errorf("error getting quote for %s: %w", stockSymbol, err)
				return
			}

			results <- fmt.Sprintf("%s: %v", stockSymbol, result)
		}(symbol)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)
	close(errors)

	// Process results and errors
	for result := range results {
		fmt.Println("Quote received:", result)
	}

	for err := range errors {
		fmt.Println("Error:", err)
	}

	// Demonstrate graceful shutdown with cleanup
	fmt.Println("Performing graceful shutdown...")

	// Disconnect from server
	if err := client.Disconnect(connectedServer.host, connectedServer.port); err != nil {
		fmt.Printf("Warning: Error during disconnect: %v\n", err)
	}

	// Shutdown the ORB with waiting for pending operations
	orb.Shutdown(true)
	fmt.Println("CORBA client shutdown complete")
}
