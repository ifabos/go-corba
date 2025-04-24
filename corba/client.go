package corba

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ifabos/go-corba/giop"
)

// Client represents a CORBA client
type Client struct {
	orb              *ORB
	connections      map[string]net.Conn
	requestIDCounter uint32
	mu               sync.RWMutex
}

// Connect establishes a connection to a CORBA server
func (c *Client) Connect(host string, port int) error {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to CORBA server at %s: %w", address, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connections == nil {
		c.connections = make(map[string]net.Conn)
	}
	c.connections[address] = conn
	return nil
}

// Disconnect closes a connection to a CORBA server
func (c *Client) Disconnect(host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)

	c.mu.Lock()
	defer c.mu.Unlock()

	conn, exists := c.connections[address]
	if !exists {
		return fmt.Errorf("no connection exists to %s", address)
	}

	// Send a CloseConnection message before closing
	closeMsg := &giop.Message{
		Header: giop.NewMessageHeader(giop.MsgCloseConn, 0),
		Body:   nil,
	}

	data, err := giop.MarshalGIOPMessage(closeMsg)
	if err == nil {
		conn.Write(data) // Best effort, ignore errors
	}

	if err := conn.Close(); err != nil {
		return fmt.Errorf("error closing connection to %s: %w", address, err)
	}

	delete(c.connections, address)
	return nil
}

// NextRequestID generates a new unique request ID
func (c *Client) NextRequestID() uint32 {
	return atomic.AddUint32(&c.requestIDCounter, 1)
}

// InvokeMethod invokes a method on a remote object using GIOP/IIOP
func (c *Client) InvokeMethod(objectName string, methodName string, serverHost string, serverPort int, args ...interface{}) (interface{}, error) {
	// Get the connection or create one if it doesn't exist
	address := fmt.Sprintf("%s:%d", serverHost, serverPort)

	c.mu.RLock()
	conn, exists := c.connections[address]
	c.mu.RUnlock()

	if !exists {
		if err := c.Connect(serverHost, serverPort); err != nil {
			return nil, err
		}
		c.mu.RLock()
		conn = c.connections[address]
		c.mu.RUnlock()
	}

	// Generate a unique request ID
	requestID := c.NextRequestID()

	// Create object key from the object name
	objectKey := []byte(objectName)

	// Create a GIOP request message
	requestMsg := giop.NewRequestMessage(requestID, objectKey, methodName, true)

	// Create request info for interceptors
	reqInfo := &RequestInfo{
		Operation:        methodName,
		ObjectKey:        objectName,
		Arguments:        args,
		RequestID:        requestID,
		ResponseExpected: true,
		ServiceContexts:  []ServiceContext{},
	}

	// Call client request interceptors - SendRequest
	interceptors := c.orb.GetInterceptorRegistry().GetClientRequestInterceptors()
	for _, interceptor := range interceptors {
		if err := interceptor.SendRequest(reqInfo); err != nil {
			return nil, err
		}
	}

	// Update service contexts from interceptors
	for _, ctx := range reqInfo.ServiceContexts {
		requestHeader, ok := requestMsg.Body.(*giop.RequestHeader)
		if !ok {
			return nil, fmt.Errorf("invalid request message format")
		}
		requestHeader.ServiceContexts = append(
			requestHeader.ServiceContexts,
			giop.ServiceContext{
				ID:   ctx.ID,
				Data: ctx.Data,
			},
		)
	}

	// Marshal the arguments using CDR
	if len(args) > 0 {
		// For now, simply store the arguments as a placeholder
		// In a real implementation, we'd use CDR to marshal them properly based on IDL definitions
		// This would be done after the request header
	}

	// Marshal the complete message
	data, err := giop.MarshalGIOPMessage(requestMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send the request
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive the reply
	headerBuf := make([]byte, 12) // Size of the GIOP header
	if _, err := io.ReadFull(conn, headerBuf); err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	// Unmarshal the header
	unmarshaller := giop.NewCDRUnmarshaller(headerBuf, binary.BigEndian)
	header, err := unmarshaller.ReadMessageHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response header: %w", err)
	}

	// Read the message body
	bodyBuf := make([]byte, header.MsgSize)
	if _, err := io.ReadFull(conn, bodyBuf); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal the entire message
	data = append(headerBuf, bodyBuf...)
	msg, err := giop.UnmarshalGIOPMessage(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Process the response
	if msg.Header.MsgType != giop.MsgReply {
		return nil, fmt.Errorf("expected reply message, got message type %d", msg.Header.MsgType)
	}

	replyHeader, ok := msg.Body.(*giop.ReplyHeader)
	if !ok {
		return nil, fmt.Errorf("invalid reply message format")
	}

	// Verify the request ID matches
	if replyHeader.RequestID != requestID {
		return nil, fmt.Errorf("mismatched request ID: expected %d, got %d", requestID, replyHeader.RequestID)
	}

	// Convert service contexts to our format for interceptors
	for _, ctx := range replyHeader.ServiceContexts {
		reqInfo.ServiceContexts = append(reqInfo.ServiceContexts, ServiceContext{
			ID:   ctx.ID,
			Data: ctx.Data,
		})
	}

	var result interface{} = "placeholder result"
	var exception Exception

	// Check the reply status
	if replyHeader.ReplyStatus != giop.ReplyStatusNoException {
		switch replyHeader.ReplyStatus {
		case giop.ReplyStatusUserException, giop.ReplyStatusSystemException:
			exception, err = c.handleExceptionReply(replyHeader)
			if err != nil {
				return nil, err
			}

			// Call client request interceptors - ReceiveException
			for _, interceptor := range interceptors {
				if err := interceptor.ReceiveException(reqInfo, exception); err != nil {
					return nil, err
				}
			}

			return nil, exception

		case giop.ReplyStatusLocationForward:
			// Call client request interceptors - ReceiveOther
			for _, interceptor := range interceptors {
				if err := interceptor.ReceiveOther(reqInfo); err != nil {
					return nil, err
				}
			}

			return nil, fmt.Errorf("location forward")

		default:
			// Call client request interceptors - ReceiveOther for unknown status
			for _, interceptor := range interceptors {
				if err := interceptor.ReceiveOther(reqInfo); err != nil {
					return nil, err
				}
			}

			return nil, fmt.Errorf("unknown reply status: %d", replyHeader.ReplyStatus)
		}
	}

	// In a real implementation, we'd unmarshal the return value from the reply body
	// For the placeholder, we'll set the result in the request info
	reqInfo.Result = result

	// Call client request interceptors - ReceiveReply
	for _, interceptor := range interceptors {
		if err := interceptor.ReceiveReply(reqInfo); err != nil {
			return nil, err
		}
	}

	// Return potentially modified result from interceptors
	return reqInfo.Result, nil
}

// handleExceptionReply processes a GIOP exception reply
func (c *Client) handleExceptionReply(reply *giop.ReplyHeader) (Exception, error) {
	// Look for the exception service context
	var exceptionData []byte
	for _, ctx := range reply.ServiceContexts {
		if ctx.ID == 0x45584350 { // "EXCP"
			exceptionData = ctx.Data
			break
		}
	}

	if len(exceptionData) == 0 {
		// No exception data found, create a generic system exception
		return UNKNOWN(0, CompletionStatusNo), nil
	}

	// In a real implementation, we would also have the TypeCode information
	// For now, we'll try to parse the simple string format used in MarshalException
	ex, err := UnmarshalException(exceptionData, nil)
	if err != nil {
		return MARSHAL(1, CompletionStatusNo), fmt.Errorf("failed to unmarshal exception: %w", err)
	}

	return ex, nil
}

// GetObject retrieves a reference to a remote object
func (c *Client) GetObject(name string, serverHost string, serverPort int) (*ObjectRef, error) {
	// Connect to server if not already connected
	address := fmt.Sprintf("%s:%d", serverHost, serverPort)

	c.mu.RLock()
	_, exists := c.connections[address]
	c.mu.RUnlock()

	if !exists {
		if err := c.Connect(serverHost, serverPort); err != nil {
			return nil, err
		}
	}

	// Create an object reference
	ref := &ObjectRef{
		Name:       name,
		ServerHost: serverHost,
		ServerPort: serverPort,
		client:     c,
	}

	return ref, nil
}
