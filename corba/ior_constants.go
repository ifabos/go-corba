// Package corba provides a CORBA implementation in Go
package corba

// IORConstants provides version constants for IOR implementation
const (
	// IIOP protocol versions
	IIOP_VERSION_1_0_MAJOR byte = 1
	IIOP_VERSION_1_0_MINOR byte = 0
	IIOP_VERSION_1_1_MAJOR byte = 1
	IIOP_VERSION_1_1_MINOR byte = 1
	IIOP_VERSION_1_2_MAJOR byte = 1
	IIOP_VERSION_1_2_MINOR byte = 2

	// Alignment constants for CDR encoding
	CDR_ALIGN_1 = 1 // octet, boolean, char
	CDR_ALIGN_2 = 2 // short, unsigned short
	CDR_ALIGN_4 = 4 // long, unsigned long, float
	CDR_ALIGN_8 = 8 // long long, unsigned long long, double

	// Character set encoding identifiers (for TAG_CODE_SETS)
	CHARSET_ISO8859_1 uint32 = 0x00010001 // ISO 8859-1 (Latin-1)
	CHARSET_UTF8      uint32 = 0x05010001 // UTF-8
	CHARSET_UTF16     uint32 = 0x00010109 // UTF-16
	CHARSET_UCS2      uint32 = 0x00010100 // UCS-2
	CHARSET_UCS4      uint32 = 0x00010104 // UCS-4

	// SSL/TLS option flags (for TAG_SSL_SEC_TRANS/TAG_TLS_SEC_TRANS)
	SSL_OPTION_NO_AUTH      uint16 = 1
	SSL_OPTION_SERVER_AUTH  uint16 = 2
	SSL_OPTION_CLIENT_AUTH  uint16 = 4
	SSL_OPTION_CONFIDENTIAL uint16 = 8
	SSL_OPTION_INTEGRITY    uint16 = 16
)

// NewStandardIIOPVersion returns the IIOP version for the specified major and minor version numbers
func NewStandardIIOPVersion(major, minor byte) IIOPVersion {
	return IIOPVersion{
		Major: major,
		Minor: minor,
	}
}

// StandardIIOPVersions provides standard IIOP versions
var (
	IIOP_1_0 = NewStandardIIOPVersion(IIOP_VERSION_1_0_MAJOR, IIOP_VERSION_1_0_MINOR)
	IIOP_1_1 = NewStandardIIOPVersion(IIOP_VERSION_1_1_MAJOR, IIOP_VERSION_1_1_MINOR)
	IIOP_1_2 = NewStandardIIOPVersion(IIOP_VERSION_1_2_MAJOR, IIOP_VERSION_1_2_MINOR)
)

// GetStandardCodeSets returns a standard CodeSets component with UTF-8 and UTF-16 support
func GetStandardCodeSets() *CodeSets {
	return &CodeSets{
		NativeCharCodeSet:  CHARSET_UTF8,
		NativeWCharCodeSet: CHARSET_UTF16,
		ConvCharCodeSets:   []uint32{CHARSET_ISO8859_1, CHARSET_UTF8},
		ConvWcharCodeSets:  []uint32{CHARSET_UTF16, CHARSET_UCS2},
	}
}

// IsIIOPVersionAtLeast checks if a given IIOP version is at least the specified major/minor
func IsIIOPVersionAtLeast(version IIOPVersion, major, minor byte) bool {
	if version.Major > major {
		return true
	}
	if version.Major == major && version.Minor >= minor {
		return true
	}
	return false
}

// IsIIOP11OrLater checks if a given IIOP version is 1.1 or later
func IsIIOP11OrLater(version IIOPVersion) bool {
	return IsIIOPVersionAtLeast(version, IIOP_VERSION_1_1_MAJOR, IIOP_VERSION_1_1_MINOR)
}

// IsIIOP12OrLater checks if a given IIOP version is 1.2 or later
func IsIIOP12OrLater(version IIOPVersion) bool {
	return IsIIOPVersionAtLeast(version, IIOP_VERSION_1_2_MAJOR, IIOP_VERSION_1_2_MINOR)
}
