package corba

import (
	"encoding/binary"
	"fmt"
)

// Component processor functions for various component types

// CodeSets represents the CodeSets component structure from CORBA spec
type CodeSets struct {
	NativeCharCodeSet  uint32
	NativeWCharCodeSet uint32
	ConvCharCodeSets   []uint32
	ConvWcharCodeSets  []uint32
}

// DecodeCodeSetsComponent decodes a TAG_CODE_SETS component
func DecodeCodeSetsComponent(data []byte) (*CodeSets, error) {
	// CODE_SET_COMPONENT requires endianness handling
	if len(data) < 1 {
		return nil, fmt.Errorf("code sets component data too short")
	}

	// Get the byte order from the first byte
	byteOrder, data, err := GetByteOrderFromData(data)
	if err != nil {
		return nil, err
	}

	// Make sure we have enough data for the fixed part (2 uint32 values)
	if len(data) < 8 {
		return nil, fmt.Errorf("code sets component data too short after byte order flag")
	}

	result := &CodeSets{}
	pos := 0

	// Native char code set
	result.NativeCharCodeSet = byteOrder.Uint32(data[pos : pos+4])
	pos += 4

	// Native wchar code set
	result.NativeWCharCodeSet = byteOrder.Uint32(data[pos : pos+4])
	pos += 4

	// Handle conversion char code sets if we have more data
	if pos+4 <= len(data) {
		count := byteOrder.Uint32(data[pos : pos+4])
		pos += 4

		if count > 0 {
			result.ConvCharCodeSets = make([]uint32, count)
			for i := uint32(0); i < count; i++ {
				if pos+4 > len(data) {
					return nil, fmt.Errorf("code sets component data corrupted")
				}
				result.ConvCharCodeSets[i] = byteOrder.Uint32(data[pos : pos+4])
				pos += 4
			}
		}
	}

	// Handle conversion wchar code sets if we have more data
	if pos+4 <= len(data) {
		count := byteOrder.Uint32(data[pos : pos+4])
		pos += 4

		if count > 0 {
			result.ConvWcharCodeSets = make([]uint32, count)
			for i := uint32(0); i < count; i++ {
				if pos+4 > len(data) {
					return nil, fmt.Errorf("code sets component data corrupted")
				}
				result.ConvWcharCodeSets[i] = byteOrder.Uint32(data[pos : pos+4])
				pos += 4
			}
		}
	}

	return result, nil
}

// EncodeCodeSetsComponent encodes a CodeSets structure into a component
func EncodeCodeSetsComponent(codeSets *CodeSets, byteOrder binary.ByteOrder) []byte {
	// Calculate buffer size
	size := 8 // 2 uint32 values (native char/wchar)

	// Add space for conversion char code sets if present
	if codeSets.ConvCharCodeSets != nil {
		size += 4 + (4 * len(codeSets.ConvCharCodeSets))
	} else {
		size += 4 // Just the count (0)
	}

	// Add space for conversion wchar code sets if present
	if codeSets.ConvWcharCodeSets != nil {
		size += 4 + (4 * len(codeSets.ConvWcharCodeSets))
	} else {
		size += 4 // Just the count (0)
	}

	// Create buffer
	buf := make([]byte, size)
	pos := 0

	// Native char code set
	byteOrder.PutUint32(buf[pos:pos+4], codeSets.NativeCharCodeSet)
	pos += 4

	// Native wchar code set
	byteOrder.PutUint32(buf[pos:pos+4], codeSets.NativeWCharCodeSet)
	pos += 4

	// Conversion char code sets
	if codeSets.ConvCharCodeSets != nil {
		byteOrder.PutUint32(buf[pos:pos+4], uint32(len(codeSets.ConvCharCodeSets)))
		pos += 4

		for _, code := range codeSets.ConvCharCodeSets {
			byteOrder.PutUint32(buf[pos:pos+4], code)
			pos += 4
		}
	} else {
		byteOrder.PutUint32(buf[pos:pos+4], 0)
		pos += 4
	}

	// Conversion wchar code sets
	if codeSets.ConvWcharCodeSets != nil {
		byteOrder.PutUint32(buf[pos:pos+4], uint32(len(codeSets.ConvWcharCodeSets)))
		pos += 4

		for _, code := range codeSets.ConvWcharCodeSets {
			byteOrder.PutUint32(buf[pos:pos+4], code)
			pos += 4
		}
	} else {
		byteOrder.PutUint32(buf[pos:pos+4], 0)
		pos += 4
	}

	// Add the byte order flag to the beginning
	return AddByteOrderFlag(buf, byteOrder)
}

// SSLData represents the SSL secure transport component structure
type SSLData struct {
	TargetSupports uint16
	TargetRequires uint16
	Port           uint16
}

// DecodeSSLComponent decodes a TAG_SSL_SEC_TRANS component
func DecodeSSLComponent(data []byte) (*SSLData, error) {
	// SSL component requires endianness handling
	if len(data) < 1 {
		return nil, fmt.Errorf("SSL component data too short")
	}

	// Get the byte order from the first byte
	byteOrder, data, err := GetByteOrderFromData(data)
	if err != nil {
		return nil, err
	}

	// Make sure we have enough data (3 uint16 values)
	if len(data) < 6 {
		return nil, fmt.Errorf("SSL component data too short after byte order flag")
	}

	result := &SSLData{}
	pos := 0

	// Target supports
	result.TargetSupports = byteOrder.Uint16(data[pos : pos+2])
	pos += 2

	// Target requires
	result.TargetRequires = byteOrder.Uint16(data[pos : pos+2])
	pos += 2

	// Port
	result.Port = byteOrder.Uint16(data[pos : pos+2])
	pos += 2

	return result, nil
}

// EncodeSSLComponent encodes an SSLData structure into a component
func EncodeSSLComponent(ssl *SSLData, byteOrder binary.ByteOrder) []byte {
	// Create buffer for 3 uint16 values
	buf := make([]byte, 6)
	pos := 0

	// Target supports
	byteOrder.PutUint16(buf[pos:pos+2], ssl.TargetSupports)
	pos += 2

	// Target requires
	byteOrder.PutUint16(buf[pos:pos+2], ssl.TargetRequires)
	pos += 2

	// Port
	byteOrder.PutUint16(buf[pos:pos+2], ssl.Port)
	pos += 2

	// Add the byte order flag to the beginning
	return AddByteOrderFlag(buf, byteOrder)
}

// DecodeComponent decodes a component based on its tag
func DecodeComponent(tag uint32, data []byte) (interface{}, error) {
	switch tag {
	case TAG_CODE_SETS:
		return DecodeCodeSetsComponent(data)
	case TAG_SSL_SEC_TRANS:
		return DecodeSSLComponent(data)
	// Add more component decoders as needed
	default:
		// For unknown components, just return the raw data
		return data, nil
	}
}
