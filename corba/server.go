package corba

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ifabos/go-corba/giop"
)

// ServerBinding represents a binding between an object and a service name
type ServerBinding struct {
	ObjectName string
	Object     interface{}
	ServiceID  string
}

// Server represents a CORBA server
type Server struct {
	orb      *ORB
	bindings []ServerBinding
	running  bool
	mu       sync.RWMutex
	listener net.Listener
	host     string
	port     int
}

// CreateServer creates a new server at the specified host and port
func (o *ORB) CreateServer(host string, port int) (*Server, error) {
	return &Server{
		orb:      o,
		bindings: make([]ServerBinding, 0),
		running:  false,
		host:     host,
		port:     port,
	}, nil
}

// RegisterServant registers a servant object with a name
func (s *Server) RegisterServant(objectName string, servant interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the servant implements the necessary interface
	invoker, ok := servant.(interface {
		Dispatch(methodName string, args []interface{}) (interface{}, error)
	})
	if !ok {
		return fmt.Errorf("servant does not implement Dispatch method")
	}

	// Register with the ORB
	if err := s.orb.RegisterObject(objectName, invoker); err != nil {
		return err
	}

	// Create a server binding
	binding := ServerBinding{
		ObjectName: objectName,
		Object:     invoker,
		ServiceID:  generateServiceID(objectName),
	}

	s.bindings = append(s.bindings, binding)
	return nil
}

// Bind registers an object with a name for the server (alias for RegisterServant)
func (s *Server) Bind(objectName string, obj interface{}) error {
	return s.RegisterServant(objectName, obj)
}

// Run starts the server
func (s *Server) Run() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	// Start the IIOP listener
	return s.startIIOPListener()
}

// Shutdown stops the server
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %w", err)
		}
	}

	s.running = false
	return nil
}

// Stop is an alias for Shutdown
func (s *Server) Stop() error {
	return s.Shutdown()
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// startIIOPListener starts the IIOP listener
func (s *Server) startIIOPListener() error {
	var err error
	address := fmt.Sprintf("%s:%d", s.host, s.port)
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	fmt.Printf("CORBA server listening on %s\n", s.listener.Addr())

	// Handle incoming connections
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				// Check if server was stopped
				s.mu.RLock()
				running := s.running
				s.mu.RUnlock()
				if !running {
					return
				}
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}

			// Handle connection in a new goroutine
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// handleConnection processes incoming IIOP requests
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// Set a read deadline to avoid hanging forever
		conn.SetReadDeadline(time.Now().Add(1 * time.Hour))

		// Read GIOP header (12 bytes)
		headerBuf := make([]byte, 12)
		if _, err := io.ReadFull(conn, headerBuf); err != nil {
			if err == io.EOF {
				// Client disconnected
				return
			}
			fmt.Printf("Error reading GIOP header: %v\n", err)
			return
		}

		// Parse the header
		unmarshaller := giop.NewCDRUnmarshaller(headerBuf, binary.BigEndian)
		header, err := unmarshaller.ReadMessageHeader()
		if err != nil {
			fmt.Printf("Error parsing GIOP header: %v\n", err)
			return
		}

		// Read the message body
		bodyBuf := make([]byte, header.MsgSize)
		if _, err := io.ReadFull(conn, bodyBuf); err != nil {
			fmt.Printf("Error reading GIOP message body: %v\n", err)
			return
		}

		// Process the message
		data := append(headerBuf, bodyBuf...)
		msg, err := giop.UnmarshalGIOPMessage(data)
		if err != nil {
			fmt.Printf("Error unmarshalling GIOP message: %v\n", err)
			continue
		}

		// Handle different message types
		switch msg.Header.MsgType {
		case giop.MsgRequest:
			requestHeader, ok := msg.Body.(*giop.RequestHeader)
			if !ok {
				fmt.Println("Invalid request message format")
				continue
			}
			// Process the request
			s.handleGIOPRequest(conn, requestHeader)

		case giop.MsgLocateRequest:
			locateHeader, ok := msg.Body.(*giop.LocateRequestHeader)
			if !ok {
				fmt.Println("Invalid locate request message format")
				continue
			}
			// Process the locate request
			s.handleGIOPLocateRequest(conn, locateHeader)

		case giop.MsgCancelRequest:
			// Currently we don't support cancellation, so we just acknowledge
			fmt.Println("Received cancel request - not implemented")

		case giop.MsgCloseConn:
			// Client wants to close the connection
			return

		default:
			fmt.Printf("Unsupported message type: %d\n", msg.Header.MsgType)
		}
	}
}

// handleGIOPRequest processes a GIOP request message
func (s *Server) handleGIOPRequest(conn net.Conn, request *giop.RequestHeader) {
	// Convert object key to string
	objectName := string(request.ObjectKey)

	// Find the object in the ORB
	obj, err := s.orb.ResolveObject(objectName)
	if err != nil {
		// Object not found, send a OBJECT_NOT_EXIST system exception
		s.sendExceptionReply(conn, request.RequestID,
			OBJECT_NOT_EXIST(1, CompletionStatusNo))
		return
	}

	// Check if the object implements the Invoke method
	invoker, ok := obj.(interface {
		Dispatch(methodName string, args []interface{}) (interface{}, error)
	})
	if !ok {
		// Object doesn't implement the Invoke method
		s.sendExceptionReply(conn, request.RequestID,
			OBJ_ADAPTER(1, CompletionStatusNo))
		return
	}

	// Safely invoke the method and convert any errors to exceptions
	result, ex := SafeInvoke(func() (interface{}, error) {
		// In a real implementation, we would extract arguments from the request body
		// For now, we just call the method without arguments
		return invoker.Dispatch(request.Operation, []interface{}{})
	})

	if ex != nil {
		// Method invocation failed with an exception
		s.sendExceptionReply(conn, request.RequestID, ex)
		return
	}

	// Send a successful reply
	s.sendSuccessReply(conn, request.RequestID, result)
}

// handleGIOPLocateRequest processes a GIOP locate request message
func (s *Server) handleGIOPLocateRequest(conn net.Conn, request *giop.LocateRequestHeader) {
	// Convert object key to string
	objectName := string(request.ObjectKey)

	// Check if the object exists in the ORB
	_, err := s.orb.ResolveObject(objectName)
	if err != nil {
		// Object not found
		s.sendLocateReply(conn, request.RequestID, giop.LocateStatusUnknownObject)
		return
	}

	// Object exists
	s.sendLocateReply(conn, request.RequestID, giop.LocateStatusObjectHere)
}

// sendSuccessReply sends a successful reply with a result
func (s *Server) sendSuccessReply(conn net.Conn, requestID uint32, _ interface{}) {
	// Create reply header
	replyHeader := &giop.ReplyHeader{
		ServiceContexts: make(giop.ServiceContextList, 0),
		RequestID:       requestID,
		ReplyStatus:     giop.ReplyStatusNoException,
	}

	// Create a reply message
	replyMsg := &giop.Message{
		Header: giop.NewMessageHeader(giop.MsgReply, 0), // Size will be set during marshalling
		Body:   replyHeader,
	}

	// In a real implementation, we would add the result value to the body
	// For now, we just send the reply header

	// Marshal the message
	data, err := giop.MarshalGIOPMessage(replyMsg)
	if err != nil {
		fmt.Printf("Error marshalling success reply: %v\n", err)
		return
	}

	// Send the reply
	if _, err := conn.Write(data); err != nil {
		fmt.Printf("Error sending success reply: %v\n", err)
	}
}

// sendLocateReply sends a locate reply
func (s *Server) sendLocateReply(conn net.Conn, requestID uint32, status uint32) {
	// Create locate reply header
	locateHeader := &giop.LocateReplyHeader{
		RequestID: requestID,
		Status:    status,
	}

	// Create a locate reply message
	locateMsg := &giop.Message{
		Header: giop.NewMessageHeader(giop.MsgLocateReply, 0), // Size will be set during marshalling
		Body:   locateHeader,
	}

	// Marshal the message
	data, err := giop.MarshalGIOPMessage(locateMsg)
	if err != nil {
		fmt.Printf("Error marshalling locate reply: %v\n", err)
		return
	}

	// Send the reply
	if _, err := conn.Write(data); err != nil {
		fmt.Printf("Error sending locate reply: %v\n", err)
	}
}

// sendExceptionReply sends an exception reply
func (s *Server) sendExceptionReply(conn net.Conn, requestID uint32, ex Exception) {
	// Create reply header with appropriate reply status
	var replyStatus uint32
	if IsSystemException(ex) {
		replyStatus = giop.ReplyStatusSystemException
	} else if IsUserException(ex) {
		replyStatus = giop.ReplyStatusUserException
	} else {
		// Default to system exception if it's an unknown exception type
		replyStatus = giop.ReplyStatusSystemException
	}

	replyHeader := &giop.ReplyHeader{
		ServiceContexts: make(giop.ServiceContextList, 0),
		RequestID:       requestID,
		ReplyStatus:     replyStatus,
	}

	// Marshal the exception
	exData, err := MarshalException(ex)
	if err != nil {
		fmt.Printf("Error marshalling exception: %v\n", err)
		// Fall back to a simple error message
		exData = []byte(ex.Error())
	}

	// Add exception info to service context
	replyHeader.ServiceContexts = append(replyHeader.ServiceContexts, giop.ServiceContext{
		ID:   0x45584350, // "EXCP"
		Data: exData,
	})

	// Create a reply message
	replyMsg := &giop.Message{
		Header: giop.NewMessageHeader(giop.MsgReply, 0), // Size will be set during marshalling
		Body:   replyHeader,
	}

	// Marshal the message
	data, err := giop.MarshalGIOPMessage(replyMsg)
	if err != nil {
		fmt.Printf("Error marshalling exception reply: %v\n", err)
		return
	}

	// Send the reply
	if _, err := conn.Write(data); err != nil {
		fmt.Printf("Error sending exception reply: %v\n", err)
	}
}

// generateServiceID creates a unique service ID for a binding
func generateServiceID(name string) string {
	return fmt.Sprintf("IDL:%s:1.0", name)
}
