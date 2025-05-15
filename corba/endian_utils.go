// Package corba provides a CORBA implementation in Go
package corba

import (
	"encoding/binary"
	"fmt"
)

// CDRByteOrder represents the byte order of CDR encoded data
type CDRByteOrder byte

const (
	// CDRBigEndian represents big endian byte order in CDR encoding (value 0)
	CDRBigEndian CDRByteOrder = 0
	// CDRLittleEndian represents little endian byte order in CDR encoding (value 1)
	CDRLittleEndian CDRByteOrder = 1
)

// GetByteOrder returns the appropriate binary.ByteOrder based on the CDR byte order flag
func GetByteOrder(flag CDRByteOrder) binary.ByteOrder {
	if flag == CDRBigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// GetByteOrderFromData extracts the byte order from the first byte of a CDR encoded component
// Returns the byte order and the data with the flag byte removed
func GetByteOrderFromData(data []byte) (binary.ByteOrder, []byte, error) {
	if len(data) < 1 {
		return nil, nil, fmt.Errorf("data too short to determine byte order")
	}

	// The first byte contains the byte order flag
	flag := CDRByteOrder(data[0])
	if flag != CDRBigEndian && flag != CDRLittleEndian {
		return nil, nil, fmt.Errorf("invalid byte order flag: %d", flag)
	}

	byteOrder := GetByteOrder(flag)
	// Return the byte order and the data without the flag byte
	return byteOrder, data[1:], nil
}

// AddByteOrderFlag adds a byte order flag to the beginning of a data buffer
func AddByteOrderFlag(data []byte, byteOrder binary.ByteOrder) []byte {
	var flag byte = 0 // Big endian
	if byteOrder == binary.LittleEndian {
		flag = 1
	}

	result := make([]byte, len(data)+1)
	result[0] = flag
	copy(result[1:], data)
	return result
}

// ComponentNeedsEndianFlag determines if a particular component type needs endian processing
func ComponentNeedsEndianFlag(tag uint32) bool {
	switch tag {
	case TAG_CODE_SETS,
		TAG_ALTERNATE_IIOP_ADDRESS,
		TAG_SSL_SEC_TRANS,
		TAG_CSI_SEC_MECH_LIST,
		TAG_TLS_SEC_TRANS,
		TAG_POLICIES:
		return true
	default:
		return false
	}
}
