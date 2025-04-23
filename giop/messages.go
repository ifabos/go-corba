// Package giop provides implementation of the General Inter-ORB Protocol (GIOP)
// as defined in the CORBA specification.
package giop

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// GIOP message types
const (
	MsgRequest       = 0
	MsgReply         = 1
	MsgCancelRequest = 2
	MsgLocateRequest = 3
	MsgLocateReply   = 4
	MsgCloseConn     = 5
	MsgMessageError  = 6
	MsgFragment      = 7
)

// Reply status values
const (
	ReplyStatusNoException         = 0
	ReplyStatusUserException       = 1
	ReplyStatusSystemException     = 2
	ReplyStatusLocationForward     = 3
	ReplyStatusLocationForwardPerm = 4
	ReplyStatusNeedsAddressingMode = 5
)

// Locate reply status values
const (
	LocateStatusUnknownObject             = 0
	LocateStatusObjectHere                = 1
	LocateStatusObjectForward             = 2
	LocateStatusObjectForwardPerm         = 3
	LocateStatusLOC_SYSTEM_EXCEPTION      = 4
	LocateStatusLOC_NEEDS_ADDRESSING_MODE = 5
)

// GIOP versions
var (
	GIOP_1_0 = [2]byte{1, 0}
	GIOP_1_1 = [2]byte{1, 1}
	GIOP_1_2 = [2]byte{1, 2}
	GIOP_1_3 = [2]byte{1, 3}
)

// MessageHeader is the common header for all GIOP messages
type MessageHeader struct {
	Magic   [4]byte // "GIOP"
	Version [2]byte // Major, Minor
	Flags   byte    // Flags (e.g. endianness, fragments)
	MsgType byte    // Message type
	MsgSize uint32  // Size of the message body
}

// ServiceContext contains information that may affect the processing of a request
type ServiceContext struct {
	ID   uint32
	Data []byte
}

// ServiceContextList is a sequence of service contexts
type ServiceContextList []ServiceContext

// RequestHeader contains fields specific to a request message
type RequestHeader struct {
	ServiceContexts  ServiceContextList
	RequestID        uint32
	ResponseExpected bool
	ObjectKey        []byte
	Operation        string
	Principal        []byte // Deprecated in GIOP 1.2+
}

// ReplyHeader contains fields specific to a reply message
type ReplyHeader struct {
	ServiceContexts ServiceContextList
	RequestID       uint32
	ReplyStatus     uint32
}

// CancelRequestHeader contains fields specific to a cancel request message
type CancelRequestHeader struct {
	RequestID uint32
}

// LocateRequestHeader contains fields specific to a locate request message
type LocateRequestHeader struct {
	RequestID uint32
	ObjectKey []byte
}

// LocateReplyHeader contains fields specific to a locate reply message
type LocateReplyHeader struct {
	RequestID uint32
	Status    uint32
}

// Message represents a complete GIOP message with header and body
type Message struct {
	Header MessageHeader
	Body   interface{}
}

// NewMessageHeader creates a new GIOP message header
func NewMessageHeader(msgType byte, msgSize uint32) MessageHeader {
	return MessageHeader{
		Magic:   [4]byte{'G', 'I', 'O', 'P'},
		Version: GIOP_1_2,
		Flags:   0, // Default to big endian
		MsgType: msgType,
		MsgSize: msgSize,
	}
}

// NewRequestMessage creates a new GIOP request message
func NewRequestMessage(requestID uint32, objectKey []byte, operation string, responseExpected bool) *Message {
	requestHeader := &RequestHeader{
		ServiceContexts:  make(ServiceContextList, 0),
		RequestID:        requestID,
		ResponseExpected: responseExpected,
		ObjectKey:        objectKey,
		Operation:        operation,
		Principal:        []byte{}, // Empty for GIOP 1.2+
	}

	// Add a timestamp service context for correlation
	timestamp := time.Now().UnixNano()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, timestamp)

	requestHeader.ServiceContexts = append(requestHeader.ServiceContexts, ServiceContext{
		ID:   0x54534400, // "TSD" (Timestamp Data)
		Data: buf.Bytes(),
	})

	// Create the message
	return &Message{
		Header: NewMessageHeader(MsgRequest, 0), // Size will be set during marshalling
		Body:   requestHeader,
	}
}

// NewReplyMessage creates a new GIOP reply message
func NewReplyMessage(requestID uint32, status uint32) *Message {
	replyHeader := &ReplyHeader{
		ServiceContexts: make(ServiceContextList, 0),
		RequestID:       requestID,
		ReplyStatus:     status,
	}

	return &Message{
		Header: NewMessageHeader(MsgReply, 0), // Size will be set during marshalling
		Body:   replyHeader,
	}
}

// IsLittleEndian returns whether the message is encoded in little endian
func (h *MessageHeader) IsLittleEndian() bool {
	return (h.Flags & 0x01) == 1
}

// HasMoreFragments returns whether more fragments follow
func (h *MessageHeader) HasMoreFragments() bool {
	return (h.Flags & 0x02) == 2
}

// Validate checks if the message header is valid
func (h *MessageHeader) Validate() error {
	if h.Magic != [4]byte{'G', 'I', 'O', 'P'} {
		return fmt.Errorf("invalid GIOP magic: %v", h.Magic)
	}

	if h.MsgType > MsgFragment {
		return fmt.Errorf("invalid message type: %d", h.MsgType)
	}

	return nil
}
