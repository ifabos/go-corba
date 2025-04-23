package giop

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
)

// CDR alignment sizes
const (
	Align1 = 1 // 8-bit types: octet, boolean, char
	Align2 = 2 // 16-bit types: short, unsigned short
	Align4 = 4 // 32-bit types: long, unsigned long, float
	Align8 = 8 // 64-bit types: long long, unsigned long long, double
)

// CDRMarshaller marshals data into CDR format
type CDRMarshaller struct {
	buffer    *bytes.Buffer
	byteOrder binary.ByteOrder
	position  int
}

// NewCDRMarshaller creates a new CDR marshaller with the specified byte order
func NewCDRMarshaller(byteOrder binary.ByteOrder) *CDRMarshaller {
	return &CDRMarshaller{
		buffer:    new(bytes.Buffer),
		byteOrder: byteOrder,
		position:  0,
	}
}

// Bytes returns the marshalled bytes
func (m *CDRMarshaller) Bytes() []byte {
	return m.buffer.Bytes()
}

// Size returns the current size of the marshalled data
func (m *CDRMarshaller) Size() int {
	return m.position
}

// align aligns the buffer position to the specified boundary
func (m *CDRMarshaller) align(alignment int) {
	if alignment <= 1 {
		return
	}

	padding := (alignment - (m.position % alignment)) % alignment
	if padding > 0 {
		padBytes := make([]byte, padding)
		m.buffer.Write(padBytes)
		m.position += padding
	}
}

// WriteBool writes a boolean value
func (m *CDRMarshaller) WriteBool(value bool) {
	m.align(Align1)
	var b byte = 0
	if value {
		b = 1
	}
	m.buffer.WriteByte(b)
	m.position++
}

// WriteOctet writes a byte value
func (m *CDRMarshaller) WriteOctet(value byte) {
	m.align(Align1)
	m.buffer.WriteByte(value)
	m.position++
}

// WriteChar writes a character value
func (m *CDRMarshaller) WriteChar(value byte) {
	m.align(Align1)
	m.buffer.WriteByte(value)
	m.position++
}

// WriteWChar writes a wide character value
func (m *CDRMarshaller) WriteWChar(value rune) {
	m.align(Align2)
	buf := make([]byte, 2)
	m.byteOrder.PutUint16(buf, uint16(value))
	m.buffer.Write(buf)
	m.position += 2
}

// WriteShort writes a 16-bit integer value
func (m *CDRMarshaller) WriteShort(value int16) {
	m.align(Align2)
	buf := make([]byte, 2)
	m.byteOrder.PutUint16(buf, uint16(value))
	m.buffer.Write(buf)
	m.position += 2
}

// WriteUShort writes a 16-bit unsigned integer value
func (m *CDRMarshaller) WriteUShort(value uint16) {
	m.align(Align2)
	buf := make([]byte, 2)
	m.byteOrder.PutUint16(buf, value)
	m.buffer.Write(buf)
	m.position += 2
}

// WriteLong writes a 32-bit integer value
func (m *CDRMarshaller) WriteLong(value int32) {
	m.align(Align4)
	buf := make([]byte, 4)
	m.byteOrder.PutUint32(buf, uint32(value))
	m.buffer.Write(buf)
	m.position += 4
}

// WriteULong writes a 32-bit unsigned integer value
func (m *CDRMarshaller) WriteULong(value uint32) {
	m.align(Align4)
	buf := make([]byte, 4)
	m.byteOrder.PutUint32(buf, value)
	m.buffer.Write(buf)
	m.position += 4
}

// WriteLongLong writes a 64-bit integer value
func (m *CDRMarshaller) WriteLongLong(value int64) {
	m.align(Align8)
	buf := make([]byte, 8)
	m.byteOrder.PutUint64(buf, uint64(value))
	m.buffer.Write(buf)
	m.position += 8
}

// WriteULongLong writes a 64-bit unsigned integer value
func (m *CDRMarshaller) WriteULongLong(value uint64) {
	m.align(Align8)
	buf := make([]byte, 8)
	m.byteOrder.PutUint64(buf, value)
	m.buffer.Write(buf)
	m.position += 8
}

// WriteFloat writes a 32-bit floating point value
func (m *CDRMarshaller) WriteFloat(value float32) {
	m.align(Align4)
	buf := make([]byte, 4)
	m.byteOrder.PutUint32(buf, math.Float32bits(value))
	m.buffer.Write(buf)
	m.position += 4
}

// WriteDouble writes a 64-bit floating point value
func (m *CDRMarshaller) WriteDouble(value float64) {
	m.align(Align8)
	buf := make([]byte, 8)
	m.byteOrder.PutUint64(buf, math.Float64bits(value))
	m.buffer.Write(buf)
	m.position += 8
}

// WriteString writes a string value
func (m *CDRMarshaller) WriteString(value string) {
	// Write the length first (including the NULL terminator)
	m.WriteULong(uint32(len(value) + 1))

	// Write the string content
	m.buffer.WriteString(value)
	m.position += len(value)

	// Write the NULL terminator
	m.buffer.WriteByte(0)
	m.position++
}

// WriteOctetSequence writes a sequence of bytes
func (m *CDRMarshaller) WriteOctetSequence(value []byte) {
	// Write the length first
	m.WriteULong(uint32(len(value)))

	// Write the bytes
	m.buffer.Write(value)
	m.position += len(value)
}

// WriteServiceContext writes a service context
func (m *CDRMarshaller) WriteServiceContext(ctx ServiceContext) {
	m.WriteULong(ctx.ID)
	m.WriteOctetSequence(ctx.Data)
}

// WriteServiceContextList writes a list of service contexts
func (m *CDRMarshaller) WriteServiceContextList(contexts ServiceContextList) {
	m.WriteULong(uint32(len(contexts)))
	for _, ctx := range contexts {
		m.WriteServiceContext(ctx)
	}
}

// WriteMessageHeader writes a GIOP message header
func (m *CDRMarshaller) WriteMessageHeader(header MessageHeader) {
	// Magic ("GIOP")
	m.buffer.Write(header.Magic[:])
	m.position += 4

	// Version (major, minor)
	m.buffer.Write(header.Version[:])
	m.position += 2

	// Flags
	m.buffer.WriteByte(header.Flags)
	m.position++

	// Message type
	m.buffer.WriteByte(header.MsgType)
	m.position++

	// Message size
	buf := make([]byte, 4)
	m.byteOrder.PutUint32(buf, header.MsgSize)
	m.buffer.Write(buf)
	m.position += 4
}

// WriteRequestHeader writes a GIOP request header
func (m *CDRMarshaller) WriteRequestHeader(header *RequestHeader) {
	// Service contexts
	m.WriteServiceContextList(header.ServiceContexts)

	// Request ID
	m.WriteULong(header.RequestID)

	// Response expected flag
	m.WriteBool(header.ResponseExpected)

	// Reserved bytes (3 bytes for alignment)
	m.buffer.Write([]byte{0, 0, 0})
	m.position += 3

	// Object key
	m.WriteOctetSequence(header.ObjectKey)

	// Operation
	m.WriteString(header.Operation)

	// Principal (deprecated in GIOP 1.2+, but included for compatibility)
	m.WriteOctetSequence(header.Principal)
}

// WriteReplyHeader writes a GIOP reply header
func (m *CDRMarshaller) WriteReplyHeader(header *ReplyHeader) {
	// Service contexts
	m.WriteServiceContextList(header.ServiceContexts)

	// Request ID
	m.WriteULong(header.RequestID)

	// Reply status
	m.WriteULong(header.ReplyStatus)
}

// WriteValue marshals a value based on its type
func (m *CDRMarshaller) WriteValue(value interface{}) error {
	if value == nil {
		return fmt.Errorf("cannot marshal nil value")
	}

	v := reflect.ValueOf(value)
	t := v.Type()

	switch t.Kind() {
	case reflect.Bool:
		m.WriteBool(v.Bool())
	case reflect.Int8:
		m.WriteOctet(byte(v.Int()))
	case reflect.Uint8:
		m.WriteOctet(byte(v.Uint()))
	case reflect.Int16:
		m.WriteShort(int16(v.Int()))
	case reflect.Uint16:
		m.WriteUShort(uint16(v.Uint()))
	case reflect.Int32:
		m.WriteLong(int32(v.Int()))
	case reflect.Uint32:
		m.WriteULong(uint32(v.Uint()))
	case reflect.Int64:
		m.WriteLongLong(v.Int())
	case reflect.Uint64:
		m.WriteULongLong(v.Uint())
	case reflect.Float32:
		m.WriteFloat(float32(v.Float()))
	case reflect.Float64:
		m.WriteDouble(v.Float())
	case reflect.String:
		m.WriteString(v.String())
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte is treated as an octet sequence
			m.WriteOctetSequence(v.Bytes())
		} else {
			// General sequence handling
			length := v.Len()
			m.WriteULong(uint32(length))

			for i := 0; i < length; i++ {
				if err := m.WriteValue(v.Index(i).Interface()); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unsupported type for marshalling: %v", t)
	}

	return nil
}

// CDRUnmarshaller unmarshals data from CDR format
type CDRUnmarshaller struct {
	reader    *bytes.Reader
	byteOrder binary.ByteOrder
	position  int
}

// NewCDRUnmarshaller creates a new CDR unmarshaller with the specified byte order
func NewCDRUnmarshaller(data []byte, byteOrder binary.ByteOrder) *CDRUnmarshaller {
	return &CDRUnmarshaller{
		reader:    bytes.NewReader(data),
		byteOrder: byteOrder,
		position:  0,
	}
}

// align aligns the reader position to the specified boundary
func (u *CDRUnmarshaller) align(alignment int) {
	if alignment <= 1 {
		return
	}

	padding := (alignment - (u.position % alignment)) % alignment
	if padding > 0 {
		_, err := u.reader.Seek(int64(padding), io.SeekCurrent)
		if err == nil {
			u.position += padding
		}
	}
}

// ReadBool reads a boolean value
func (u *CDRUnmarshaller) ReadBool() (bool, error) {
	u.align(Align1)
	b, err := u.reader.ReadByte()
	if err != nil {
		return false, err
	}
	u.position++
	return b != 0, nil
}

// ReadOctet reads a byte value
func (u *CDRUnmarshaller) ReadOctet() (byte, error) {
	u.align(Align1)
	b, err := u.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	u.position++
	return b, nil
}

// ReadChar reads a character value
func (u *CDRUnmarshaller) ReadChar() (byte, error) {
	return u.ReadOctet()
}

// ReadWChar reads a wide character value
func (u *CDRUnmarshaller) ReadWChar() (rune, error) {
	u.align(Align2)
	buf := make([]byte, 2)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 2
	return rune(u.byteOrder.Uint16(buf)), nil
}

// ReadShort reads a 16-bit integer value
func (u *CDRUnmarshaller) ReadShort() (int16, error) {
	u.align(Align2)
	buf := make([]byte, 2)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 2
	return int16(u.byteOrder.Uint16(buf)), nil
}

// ReadUShort reads a 16-bit unsigned integer value
func (u *CDRUnmarshaller) ReadUShort() (uint16, error) {
	u.align(Align2)
	buf := make([]byte, 2)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 2
	return u.byteOrder.Uint16(buf), nil
}

// ReadLong reads a 32-bit integer value
func (u *CDRUnmarshaller) ReadLong() (int32, error) {
	u.align(Align4)
	buf := make([]byte, 4)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 4
	return int32(u.byteOrder.Uint32(buf)), nil
}

// ReadULong reads a 32-bit unsigned integer value
func (u *CDRUnmarshaller) ReadULong() (uint32, error) {
	u.align(Align4)
	buf := make([]byte, 4)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 4
	return u.byteOrder.Uint32(buf), nil
}

// ReadLongLong reads a 64-bit integer value
func (u *CDRUnmarshaller) ReadLongLong() (int64, error) {
	u.align(Align8)
	buf := make([]byte, 8)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 8
	return int64(u.byteOrder.Uint64(buf)), nil
}

// ReadULongLong reads a 64-bit unsigned integer value
func (u *CDRUnmarshaller) ReadULongLong() (uint64, error) {
	u.align(Align8)
	buf := make([]byte, 8)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 8
	return u.byteOrder.Uint64(buf), nil
}

// ReadFloat reads a 32-bit floating point value
func (u *CDRUnmarshaller) ReadFloat() (float32, error) {
	u.align(Align4)
	buf := make([]byte, 4)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 4
	return math.Float32frombits(u.byteOrder.Uint32(buf)), nil
}

// ReadDouble reads a 64-bit floating point value
func (u *CDRUnmarshaller) ReadDouble() (float64, error) {
	u.align(Align8)
	buf := make([]byte, 8)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return 0, err
	}
	u.position += 8
	return math.Float64frombits(u.byteOrder.Uint64(buf)), nil
}

// ReadString reads a string value
func (u *CDRUnmarshaller) ReadString() (string, error) {
	// Read the length first (including the NULL terminator)
	length, err := u.ReadULong()
	if err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	// Read the string content (excluding NULL terminator)
	buf := make([]byte, length-1)
	if _, err := io.ReadFull(u.reader, buf); err != nil {
		return "", err
	}
	u.position += int(length - 1)

	// Skip the NULL terminator
	_, err = u.reader.ReadByte()
	if err != nil {
		return "", err
	}
	u.position++

	return string(buf), nil
}

// ReadOctetSequence reads a sequence of bytes
func (u *CDRUnmarshaller) ReadOctetSequence() ([]byte, error) {
	// Read the length first
	length, err := u.ReadULong()
	if err != nil {
		return nil, err
	}

	// Read the bytes
	buf := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(u.reader, buf); err != nil {
			return nil, err
		}
		u.position += int(length)
	}

	return buf, nil
}

// ReadServiceContext reads a service context
func (u *CDRUnmarshaller) ReadServiceContext() (ServiceContext, error) {
	var ctx ServiceContext
	var err error

	// Read the context ID
	if ctx.ID, err = u.ReadULong(); err != nil {
		return ctx, err
	}

	// Read the context data
	if ctx.Data, err = u.ReadOctetSequence(); err != nil {
		return ctx, err
	}

	return ctx, nil
}

// ReadServiceContextList reads a list of service contexts
func (u *CDRUnmarshaller) ReadServiceContextList() (ServiceContextList, error) {
	// Read the number of contexts
	count, err := u.ReadULong()
	if err != nil {
		return nil, err
	}

	// Read each context
	contexts := make(ServiceContextList, count)
	for i := uint32(0); i < count; i++ {
		if contexts[i], err = u.ReadServiceContext(); err != nil {
			return nil, err
		}
	}

	return contexts, nil
}

// ReadMessageHeader reads a GIOP message header
func (u *CDRUnmarshaller) ReadMessageHeader() (MessageHeader, error) {
	var header MessageHeader
	var err error

	// Read the magic
	if _, err = io.ReadFull(u.reader, header.Magic[:]); err != nil {
		return header, err
	}
	u.position += 4

	// Read the version
	if _, err = io.ReadFull(u.reader, header.Version[:]); err != nil {
		return header, err
	}
	u.position += 2

	// Read the flags
	if header.Flags, err = u.reader.ReadByte(); err != nil {
		return header, err
	}
	u.position++

	// Set the byte order based on the flags
	if header.IsLittleEndian() {
		u.byteOrder = binary.LittleEndian
	} else {
		u.byteOrder = binary.BigEndian
	}

	// Read the message type
	if header.MsgType, err = u.reader.ReadByte(); err != nil {
		return header, err
	}
	u.position++

	// Read the message size
	buf := make([]byte, 4)
	if _, err = io.ReadFull(u.reader, buf); err != nil {
		return header, err
	}
	u.position += 4
	header.MsgSize = u.byteOrder.Uint32(buf)

	// Validate the header
	if err = header.Validate(); err != nil {
		return header, err
	}

	return header, nil
}

// ReadRequestHeader reads a GIOP request header
func (u *CDRUnmarshaller) ReadRequestHeader() (*RequestHeader, error) {
	header := &RequestHeader{}
	var err error

	// Read the service contexts
	if header.ServiceContexts, err = u.ReadServiceContextList(); err != nil {
		return nil, err
	}

	// Read the request ID
	if header.RequestID, err = u.ReadULong(); err != nil {
		return nil, err
	}

	// Read the response expected flag
	if header.ResponseExpected, err = u.ReadBool(); err != nil {
		return nil, err
	}

	// Skip 3 reserved bytes
	buf := make([]byte, 3)
	if _, err = io.ReadFull(u.reader, buf); err != nil {
		return nil, err
	}
	u.position += 3

	// Read the object key
	if header.ObjectKey, err = u.ReadOctetSequence(); err != nil {
		return nil, err
	}

	// Read the operation name
	if header.Operation, err = u.ReadString(); err != nil {
		return nil, err
	}

	// Read the principal (deprecated in GIOP 1.2+, but included for compatibility)
	if header.Principal, err = u.ReadOctetSequence(); err != nil {
		return nil, err
	}

	return header, nil
}

// ReadReplyHeader reads a GIOP reply header
func (u *CDRUnmarshaller) ReadReplyHeader() (*ReplyHeader, error) {
	header := &ReplyHeader{}
	var err error

	// Read the service contexts
	if header.ServiceContexts, err = u.ReadServiceContextList(); err != nil {
		return nil, err
	}

	// Read the request ID
	if header.RequestID, err = u.ReadULong(); err != nil {
		return nil, err
	}

	// Read the reply status
	if header.ReplyStatus, err = u.ReadULong(); err != nil {
		return nil, err
	}

	return header, nil
}

// ReadValue unmarshals a value based on the expected type
func (u *CDRUnmarshaller) ReadValue(target interface{}) error {
	if target == nil {
		return fmt.Errorf("cannot unmarshal into nil target")
	}

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	v = v.Elem() // Dereference the pointer
	t := v.Type()

	switch t.Kind() {
	case reflect.Bool:
		val, err := u.ReadBool()
		if err != nil {
			return err
		}
		v.SetBool(val)

	case reflect.Int8:
		val, err := u.ReadOctet()
		if err != nil {
			return err
		}
		v.SetInt(int64(val))

	case reflect.Uint8:
		val, err := u.ReadOctet()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))

	case reflect.Int16:
		val, err := u.ReadShort()
		if err != nil {
			return err
		}
		v.SetInt(int64(val))

	case reflect.Uint16:
		val, err := u.ReadUShort()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))

	case reflect.Int32:
		val, err := u.ReadLong()
		if err != nil {
			return err
		}
		v.SetInt(int64(val))

	case reflect.Uint32:
		val, err := u.ReadULong()
		if err != nil {
			return err
		}
		v.SetUint(uint64(val))

	case reflect.Int64:
		val, err := u.ReadLongLong()
		if err != nil {
			return err
		}
		v.SetInt(val)

	case reflect.Uint64:
		val, err := u.ReadULongLong()
		if err != nil {
			return err
		}
		v.SetUint(val)

	case reflect.Float32:
		val, err := u.ReadFloat()
		if err != nil {
			return err
		}
		v.SetFloat(float64(val))

	case reflect.Float64:
		val, err := u.ReadDouble()
		if err != nil {
			return err
		}
		v.SetFloat(val)

	case reflect.String:
		val, err := u.ReadString()
		if err != nil {
			return err
		}
		v.SetString(val)

	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte is treated as an octet sequence
			val, err := u.ReadOctetSequence()
			if err != nil {
				return err
			}
			v.SetBytes(val)
		} else {
			// General sequence handling
			length, err := u.ReadULong()
			if err != nil {
				return err
			}

			slice := reflect.MakeSlice(t, int(length), int(length))
			for i := uint32(0); i < length; i++ {
				elemPtr := reflect.New(t.Elem())
				if err := u.ReadValue(elemPtr.Interface()); err != nil {
					return err
				}
				slice.Index(int(i)).Set(elemPtr.Elem())
			}
			v.Set(slice)
		}

	default:
		return fmt.Errorf("unsupported type for unmarshalling: %v", t)
	}

	return nil
}

// MarshalGIOPMessage marshals a GIOP message to bytes
func MarshalGIOPMessage(msg *Message) ([]byte, error) {
	// Determine byte order from the flags
	var byteOrder binary.ByteOrder = binary.BigEndian
	if (msg.Header.Flags & 0x01) == 1 {
		byteOrder = binary.LittleEndian
	}

	// Create a marshaller for the body
	bodyMarshaller := NewCDRMarshaller(byteOrder)

	// Marshal the message body based on the message type
	switch msg.Header.MsgType {
	case MsgRequest:
		if requestHeader, ok := msg.Body.(*RequestHeader); ok {
			bodyMarshaller.WriteRequestHeader(requestHeader)
		} else {
			return nil, fmt.Errorf("body is not a RequestHeader")
		}

	case MsgReply:
		if replyHeader, ok := msg.Body.(*ReplyHeader); ok {
			bodyMarshaller.WriteReplyHeader(replyHeader)
		} else {
			return nil, fmt.Errorf("body is not a ReplyHeader")
		}

	case MsgCancelRequest:
		if cancelHeader, ok := msg.Body.(*CancelRequestHeader); ok {
			bodyMarshaller.WriteULong(cancelHeader.RequestID)
		} else {
			return nil, fmt.Errorf("body is not a CancelRequestHeader")
		}

	case MsgLocateRequest:
		if locateHeader, ok := msg.Body.(*LocateRequestHeader); ok {
			bodyMarshaller.WriteULong(locateHeader.RequestID)
			bodyMarshaller.WriteOctetSequence(locateHeader.ObjectKey)
		} else {
			return nil, fmt.Errorf("body is not a LocateRequestHeader")
		}

	case MsgLocateReply:
		if locateHeader, ok := msg.Body.(*LocateReplyHeader); ok {
			bodyMarshaller.WriteULong(locateHeader.RequestID)
			bodyMarshaller.WriteULong(locateHeader.Status)
		} else {
			return nil, fmt.Errorf("body is not a LocateReplyHeader")
		}

	case MsgCloseConn:
		// No body for close connection message

	case MsgMessageError:
		if errorMsg, ok := msg.Body.(string); ok {
			bodyMarshaller.WriteString(errorMsg)
		} else {
			return nil, fmt.Errorf("body is not a string")
		}

	case MsgFragment:
		// Fragment handling depends on the implementation
		return nil, fmt.Errorf("fragment messages not fully implemented")

	default:
		return nil, fmt.Errorf("unknown message type: %d", msg.Header.MsgType)
	}

	// Update the message size in the header
	bodyBytes := bodyMarshaller.Bytes()
	msg.Header.MsgSize = uint32(len(bodyBytes))

	// Create a marshaller for the complete message
	completeMarshaller := NewCDRMarshaller(byteOrder)
	completeMarshaller.WriteMessageHeader(msg.Header)

	// Append the body to the result
	result := append(completeMarshaller.Bytes(), bodyBytes...)
	return result, nil
}

// UnmarshalGIOPMessage unmarshals a GIOP message from bytes
func UnmarshalGIOPMessage(data []byte) (*Message, error) {
	// Create a default unmarshaller (byte order will be adjusted after reading header)
	unmarshaller := NewCDRUnmarshaller(data, binary.BigEndian)

	// Read the message header
	header, err := unmarshaller.ReadMessageHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to read message header: %w", err)
	}

	// Create the message
	msg := &Message{Header: header}

	// Read the message body based on the message type
	switch header.MsgType {
	case MsgRequest:
		requestHeader, err := unmarshaller.ReadRequestHeader()
		if err != nil {
			return nil, fmt.Errorf("failed to read request header: %w", err)
		}
		msg.Body = requestHeader

	case MsgReply:
		replyHeader, err := unmarshaller.ReadReplyHeader()
		if err != nil {
			return nil, fmt.Errorf("failed to read reply header: %w", err)
		}
		msg.Body = replyHeader

	case MsgCancelRequest:
		cancelHeader := &CancelRequestHeader{}
		cancelHeader.RequestID, err = unmarshaller.ReadULong()
		if err != nil {
			return nil, fmt.Errorf("failed to read cancel request ID: %w", err)
		}
		msg.Body = cancelHeader

	case MsgLocateRequest:
		locateHeader := &LocateRequestHeader{}
		locateHeader.RequestID, err = unmarshaller.ReadULong()
		if err != nil {
			return nil, fmt.Errorf("failed to read locate request ID: %w", err)
		}
		locateHeader.ObjectKey, err = unmarshaller.ReadOctetSequence()
		if err != nil {
			return nil, fmt.Errorf("failed to read locate object key: %w", err)
		}
		msg.Body = locateHeader

	case MsgLocateReply:
		locateHeader := &LocateReplyHeader{}
		locateHeader.RequestID, err = unmarshaller.ReadULong()
		if err != nil {
			return nil, fmt.Errorf("failed to read locate reply request ID: %w", err)
		}
		locateHeader.Status, err = unmarshaller.ReadULong()
		if err != nil {
			return nil, fmt.Errorf("failed to read locate reply status: %w", err)
		}
		msg.Body = locateHeader

	case MsgCloseConn:
		// No body for close connection message

	case MsgMessageError:
		errorMsg, err := unmarshaller.ReadString()
		if err != nil {
			return nil, fmt.Errorf("failed to read error message: %w", err)
		}
		msg.Body = errorMsg

	case MsgFragment:
		// Fragment handling depends on the implementation
		return nil, fmt.Errorf("fragment messages not fully implemented")

	default:
		return nil, fmt.Errorf("unknown message type: %d", header.MsgType)
	}

	return msg, nil
}
