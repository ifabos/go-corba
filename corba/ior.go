// Package corba provides a CORBA implementation in Go
package corba

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// IIOP version information
type IIOPVersion struct {
	Major byte
	Minor byte
}

// String returns the string representation of an IIOP version
func (v IIOPVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// TaggedProfile represents a profile with a specific tag in an IOR
type TaggedProfile struct {
	Tag     uint32
	Profile []byte
}

// TaggedComponent represents a component with a specific tag in an IOR profile
type TaggedComponent struct {
	Tag       uint32
	Component []byte
	// DecodedData stores the decoded component data when available
	DecodedData interface{}
}

// ProfileBody_1_1 represents the profile body for IIOP 1.1 and later
type ProfileBody_1_1 struct {
	Version    IIOPVersion
	Host       string
	Port       uint16
	ObjectKey  []byte
	Components []TaggedComponent
}

// Known profile tags from the CORBA specification
const (
	TAG_INTERNET_IOP        uint32 = 0 // Standard IIOP profile
	TAG_MULTIPLE_COMPONENTS uint32 = 1 // For multiple components
	TAG_SCCP_IOP            uint32 = 2 // For SCCP transport
	TAG_UIPMC               uint32 = 3 // For unreliable multicast
)

// Known component tags from the CORBA specification
const (
	TAG_ORB_TYPE                 uint32 = 0  // The ORB type
	TAG_CODE_SETS                uint32 = 1  // Character and wide character code sets
	TAG_POLICIES                 uint32 = 2  // Policies associated with the object
	TAG_ALTERNATE_IIOP_ADDRESS   uint32 = 3  // Alternative IIOP address
	TAG_ASSOCIATION_OPTIONS      uint32 = 13 // Security association options
	TAG_SEC_NAME                 uint32 = 14 // Security name component
	TAG_SPKM_1_SEC_MECH          uint32 = 15 // SPKM security mechanism
	TAG_SPKM_2_SEC_MECH          uint32 = 16 // SPKM security mechanism
	TAG_KerberosV5_SEC_MECH      uint32 = 17 // Kerberos 5 security mechanism
	TAG_CSI_ECMA_SECRET_SEC_MECH uint32 = 18 // CSI ECMA security mechanism
	TAG_CSI_ECMA_HYBRID_SEC_MECH uint32 = 19 // CSI ECMA security mechanism
	TAG_SSL_SEC_TRANS            uint32 = 20 // SSL security transport
	TAG_CSI_ECMA_PUBLIC_SEC_MECH uint32 = 21 // CSI ECMA security mechanism
	TAG_GENERIC_SEC_MECH         uint32 = 22 // Generic security mechanism
	TAG_JAVA_CODEBASE            uint32 = 25 // Java codebase URL
	TAG_TRANSACTION_POLICY       uint32 = 26 // Transaction policy
	TAG_MESSAGE_ROUTERS          uint32 = 30 // Message routers
	TAG_OTS_POLICY               uint32 = 31 // OTS policy
	TAG_INV_POLICY               uint32 = 32 // Invocation policy
	TAG_CSI_SEC_MECH_LIST        uint32 = 33 // CSI security mechanism list
	TAG_NULL_TAG                 uint32 = 34 // Null tag
	TAG_SECIOP_SEC_TRANS         uint32 = 35 // SECIOP security transport
	TAG_TLS_SEC_TRANS            uint32 = 36 // TLS security transport
)

// IOR represents a CORBA Interoperable Object Reference
type IOR struct {
	TypeID   string
	Profiles []TaggedProfile
}

// NewIOR creates a new IOR with specified type ID
func NewIOR(typeID string) *IOR {
	return &IOR{
		TypeID:   typeID,
		Profiles: []TaggedProfile{},
	}
}

// AddIIOPProfile adds a new IIOP profile to the IOR
func (ior *IOR) AddIIOPProfile(version IIOPVersion, host string, port uint16, objectKey []byte) {
	// Create a new IIOP profile
	profile := createIIOPProfile(version, host, port, objectKey, nil)

	// Add it to the profiles list
	ior.Profiles = append(ior.Profiles, profile)
}

// createIIOPProfile creates a standard IIOP profile
func createIIOPProfile(version IIOPVersion, host string, port uint16, objectKey []byte, components []TaggedComponent) TaggedProfile {
	// Calculate the buffer size needed for the profile
	bufSize := 2 + // Version (major, minor)
		4 + len(host) + // Host length + host string
		2 + // Port
		4 + len(objectKey) // Object key length + key bytes

	// Add components size if present
	if version.Major > 1 || (version.Major == 1 && version.Minor >= 1) {
		bufSize += 4 // Components count
		for _, comp := range components {
			bufSize += 4 + 4 + len(comp.Component) // Tag + component length + component data
		}
	}

	// Create the buffer for the profile data
	buf := make([]byte, 0, bufSize)

	// Version
	buf = append(buf, version.Major, version.Minor)

	// Host
	hostLen := uint32(len(host))
	hostLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(hostLenBytes, hostLen)
	buf = append(buf, hostLenBytes...)
	buf = append(buf, []byte(host)...)

	// Port
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	buf = append(buf, portBytes...)

	// Object key
	keyLen := uint32(len(objectKey))
	keyLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(keyLenBytes, keyLen)
	buf = append(buf, keyLenBytes...)
	buf = append(buf, objectKey...)

	// Components (for IIOP 1.1 and later)
	if version.Major > 1 || (version.Major == 1 && version.Minor >= 1) {
		// Number of components
		compCountBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(compCountBytes, uint32(len(components)))
		buf = append(buf, compCountBytes...)

		// Each component
		for _, comp := range components {
			// Tag
			tagBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(tagBytes, comp.Tag)
			buf = append(buf, tagBytes...)

			// Component data length
			compLenBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(compLenBytes, uint32(len(comp.Component)))
			buf = append(buf, compLenBytes...)

			// Component data - handle specially for components with endianness requirements
			componentData := comp.Component

			// For components that have encoded data with their own endianness
			if comp.DecodedData != nil && ComponentNeedsEndianFlag(comp.Tag) {
				// Re-encode using the proper endianness
				switch comp.Tag {
				case TAG_CODE_SETS:
					if codeSets, ok := comp.DecodedData.(*CodeSets); ok {
						componentData = EncodeCodeSetsComponent(codeSets, binary.BigEndian)
					}
				case TAG_SSL_SEC_TRANS:
					if ssl, ok := comp.DecodedData.(*SSLData); ok {
						componentData = EncodeSSLComponent(ssl, binary.BigEndian)
					}
					// Add other component types as needed
				}
			}

			buf = append(buf, componentData...)
		}
	}

	return TaggedProfile{
		Tag:     TAG_INTERNET_IOP,
		Profile: buf,
	}
}

// Encode serializes the IOR into its binary representation
func (ior *IOR) Encode() []byte {
	// Calculate the needed buffer size
	bufSize := 4 + len(ior.TypeID) + // Type ID length + Type ID string
		4 // Profile count

	// Add size for each profile
	for _, profile := range ior.Profiles {
		bufSize += 4 + // Tag
			4 + // Profile data length
			len(profile.Profile) // Profile data
	}

	// Create the buffer
	buf := make([]byte, 0, bufSize)

	// Type ID
	typeIDLen := uint32(len(ior.TypeID))
	typeIDLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(typeIDLenBytes, typeIDLen)
	buf = append(buf, typeIDLenBytes...)
	buf = append(buf, []byte(ior.TypeID)...)

	// Profile count
	profileCountBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(profileCountBytes, uint32(len(ior.Profiles)))
	buf = append(buf, profileCountBytes...)

	// Each profile
	for _, profile := range ior.Profiles {
		// Tag
		tagBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(tagBytes, profile.Tag)
		buf = append(buf, tagBytes...)

		// Profile data length
		profileLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(profileLenBytes, uint32(len(profile.Profile)))
		buf = append(buf, profileLenBytes...)

		// Profile data
		buf = append(buf, profile.Profile...)
	}

	return buf
}

// Decode deserializes a binary representation into an IOR
func DecodeIOR(data []byte) (*IOR, error) {
	// Check minimum length
	if len(data) < 8 { // Need at least 4 bytes for type ID length and 4 bytes for profile count
		return nil, fmt.Errorf("data too short to be valid IOR")
	}

	ior := &IOR{}
	pos := 0

	// Type ID
	typeIDLen := binary.BigEndian.Uint32(data[pos : pos+4])
	pos += 4
	if pos+int(typeIDLen) > len(data) {
		return nil, fmt.Errorf("invalid type ID length")
	}
	ior.TypeID = string(data[pos : pos+int(typeIDLen)])
	pos += int(typeIDLen)

	// Profile count
	if pos+4 > len(data) {
		return nil, fmt.Errorf("data too short to contain profile count")
	}
	profileCount := binary.BigEndian.Uint32(data[pos : pos+4])
	pos += 4

	// Profiles
	ior.Profiles = make([]TaggedProfile, 0, profileCount)
	for i := uint32(0); i < profileCount; i++ {
		// Check if we have enough bytes for tag and profile length
		if pos+8 > len(data) {
			return nil, fmt.Errorf("data too short to contain profile #%d", i+1)
		}

		// Tag
		tag := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		// Profile data length
		profileLen := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		// Profile data
		if pos+int(profileLen) > len(data) {
			return nil, fmt.Errorf("invalid profile data length for profile #%d", i+1)
		}
		profile := data[pos : pos+int(profileLen)]
		pos += int(profileLen)

		ior.Profiles = append(ior.Profiles, TaggedProfile{
			Tag:     tag,
			Profile: profile,
		})
	}

	return ior, nil
}

// DecodeIIOPProfile extracts IIOP profile information
func DecodeIIOPProfile(profile []byte) (*ProfileBody_1_1, error) {
	// Check minimum profile length
	if len(profile) < 8 { // Need at least version (2), host len (4), port (2)
		return nil, fmt.Errorf("profile data too short")
	}

	pos := 0

	// IIOP version
	version := IIOPVersion{
		Major: profile[pos],
		Minor: profile[pos+1],
	}
	pos += 2

	// Host
	if pos+4 > len(profile) {
		return nil, fmt.Errorf("invalid profile format: missing host length")
	}
	hostLen := binary.BigEndian.Uint32(profile[pos : pos+4])
	pos += 4
	if pos+int(hostLen) > len(profile) {
		return nil, fmt.Errorf("invalid host length")
	}
	host := string(profile[pos : pos+int(hostLen)])
	pos += int(hostLen)

	// Port
	if pos+2 > len(profile) {
		return nil, fmt.Errorf("invalid profile format: missing port")
	}
	port := binary.BigEndian.Uint16(profile[pos : pos+2])
	pos += 2

	// Object Key
	if pos+4 > len(profile) {
		return nil, fmt.Errorf("invalid profile format: missing object key length")
	}
	keyLen := binary.BigEndian.Uint32(profile[pos : pos+4])
	pos += 4
	if pos+int(keyLen) > len(profile) {
		return nil, fmt.Errorf("invalid object key length")
	}
	objectKey := make([]byte, keyLen)
	copy(objectKey, profile[pos:pos+int(keyLen)])
	pos += int(keyLen)

	result := &ProfileBody_1_1{
		Version:    version,
		Host:       host,
		Port:       port,
		ObjectKey:  objectKey,
		Components: []TaggedComponent{},
	}

	// Components (for IIOP 1.1 and later)
	if version.Major > 1 || (version.Major == 1 && version.Minor >= 1) {
		// If there are more bytes, parse components
		if pos+4 <= len(profile) {
			compCount := binary.BigEndian.Uint32(profile[pos : pos+4])
			pos += 4

			for i := uint32(0); i < compCount; i++ {
				if pos+8 > len(profile) { // Need tag (4) + length (4)
					return nil, fmt.Errorf("invalid component data in profile")
				}

				tag := binary.BigEndian.Uint32(profile[pos : pos+4])
				pos += 4

				compLen := binary.BigEndian.Uint32(profile[pos : pos+4])
				pos += 4

				if pos+int(compLen) > len(profile) {
					return nil, fmt.Errorf("invalid component length in profile")
				}

				compData := make([]byte, compLen)
				copy(compData, profile[pos:pos+int(compLen)])
				pos += int(compLen)

				// Create a component with the raw data
				component := TaggedComponent{
					Tag:       tag,
					Component: compData,
				}

				// If this component type needs endianness processing, decode it
				if ComponentNeedsEndianFlag(tag) {
					if decoded, err := DecodeComponent(tag, compData); err == nil {
						component.DecodedData = decoded
					}
				}

				result.Components = append(result.Components, component)
			}
		}
	}

	return result, nil
}

// ParseIOR parses a stringified IOR format
func ParseIOR(iorString string) (*IOR, error) {
	// Check the IOR prefix
	if !strings.HasPrefix(iorString, "IOR:") {
		return nil, fmt.Errorf("invalid IOR string format, must start with 'IOR:'")
	}

	// Remove the prefix
	hexString := strings.TrimPrefix(iorString, "IOR:")

	// Decode the hex string
	data, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, fmt.Errorf("invalid IOR hex format: %w", err)
	}

	// Decode the binary data
	return DecodeIOR(data)
}

// ToString converts an IOR to its stringified representation
func (ior *IOR) ToString() string {
	// Encode the IOR to its binary form
	data := ior.Encode()

	// Convert to hex string with IOR: prefix
	return "IOR:" + strings.ToUpper(hex.EncodeToString(data))
}

// GetIIOPProfiles returns all IIOP profiles in the IOR
func (ior *IOR) GetIIOPProfiles() ([]*ProfileBody_1_1, error) {
	result := make([]*ProfileBody_1_1, 0, len(ior.Profiles))

	for _, profile := range ior.Profiles {
		if profile.Tag == TAG_INTERNET_IOP {
			iiopProfile, err := DecodeIIOPProfile(profile.Profile)
			if err != nil {
				return nil, err
			}
			result = append(result, iiopProfile)
		}
	}

	return result, nil
}

// GetPrimaryIIOPProfile returns the primary (first) IIOP profile
func (ior *IOR) GetPrimaryIIOPProfile() (*ProfileBody_1_1, error) {
	for _, profile := range ior.Profiles {
		if profile.Tag == TAG_INTERNET_IOP {
			return DecodeIIOPProfile(profile.Profile)
		}
	}

	return nil, fmt.Errorf("no IIOP profile found in IOR")
}

// GetComponent retrieves a specific component from an IIOP profile
func (profile *ProfileBody_1_1) GetComponent(tag uint32) (*TaggedComponent, error) {
	for i, comp := range profile.Components {
		if comp.Tag == tag {
			return &profile.Components[i], nil
		}
	}
	return nil, fmt.Errorf("component with tag %d not found", tag)
}

// GetComponentData retrieves and decodes a specific component from an IIOP profile
func (profile *ProfileBody_1_1) GetComponentData(tag uint32) (interface{}, error) {
	comp, err := profile.GetComponent(tag)
	if err != nil {
		return nil, err
	}

	// If already decoded, return that
	if comp.DecodedData != nil {
		return comp.DecodedData, nil
	}

	// Otherwise try to decode it
	return DecodeComponent(tag, comp.Component)
}

// AddComponent adds a component to an IIOP profile
func (profile *ProfileBody_1_1) AddComponent(component TaggedComponent) {
	profile.Components = append(profile.Components, component)
}

// AddComponentData adds a component to an IIOP profile using the structured data
func (profile *ProfileBody_1_1) AddComponentData(tag uint32, data interface{}) {
	component := CreateTaggedComponent(tag, data)
	profile.AddComponent(component)
}

// GetCodeSets retrieves the CodeSets component if available
func (profile *ProfileBody_1_1) GetCodeSets() (*CodeSets, error) {
	data, err := profile.GetComponentData(TAG_CODE_SETS)
	if err != nil {
		return nil, err
	}

	if codeSets, ok := data.(*CodeSets); ok {
		return codeSets, nil
	}

	return nil, fmt.Errorf("invalid CodeSets component data")
}

// GetSSLData retrieves the SSL component if available
func (profile *ProfileBody_1_1) GetSSLData() (*SSLData, error) {
	data, err := profile.GetComponentData(TAG_SSL_SEC_TRANS)
	if err != nil {
		return nil, err
	}

	if ssl, ok := data.(*SSLData); ok {
		return ssl, nil
	}

	return nil, fmt.Errorf("invalid SSL component data")
}

// FormatRepositoryID formats a repository ID according to CORBA standards
// Format: "IDL:<interface_name>:<version>"
func FormatRepositoryID(interfaceName string, version string) string {
	if version == "" {
		version = "1.0" // Default version
	}

	// If it's already in the correct format, return it
	if strings.HasPrefix(interfaceName, "IDL:") && strings.Contains(interfaceName, ":") {
		return interfaceName
	}

	// Remove any IDL: prefix if present
	name := strings.TrimPrefix(interfaceName, "IDL:")

	// Replace any dots with slashes for scoping
	name = strings.Replace(name, ".", "/", -1)

	// Format the repository ID
	return fmt.Sprintf("IDL:%s:%s", name, version)
}

// ObjectKeyFromString creates an object key from a string
func ObjectKeyFromString(key string) []byte {
	return []byte(key)
}

// ObjectKeyToString converts an object key to a string
func ObjectKeyToString(key []byte) string {
	return string(key)
}

// GenerateObjectKey generates a unique object key
// In a real implementation, this would use more sophisticated techniques
func GenerateObjectKey(prefix string) []byte {
	if prefix == "" {
		prefix = "OBJ_"
	}
	return []byte(fmt.Sprintf("%s%d", prefix, GetNextObjectID()))
}

// Global object ID counter for simple object ID generation
var nextObjectID uint64 = 1
var objectIDMutex sync.Mutex

// GetNextObjectID returns the next unique object ID
func GetNextObjectID() uint64 {
	objectIDMutex.Lock()
	defer objectIDMutex.Unlock()

	id := nextObjectID
	nextObjectID++
	return id
}

// CreateTaggedComponent creates a new TaggedComponent with proper endianness handling
func CreateTaggedComponent(tag uint32, data interface{}) TaggedComponent {
	component := TaggedComponent{
		Tag:         tag,
		DecodedData: data,
	}

	// Handle different component types
	switch tag {
	case TAG_CODE_SETS:
		if codeSets, ok := data.(*CodeSets); ok {
			component.Component = EncodeCodeSetsComponent(codeSets, binary.BigEndian)
		}
	case TAG_SSL_SEC_TRANS:
		if ssl, ok := data.(*SSLData); ok {
			component.Component = EncodeSSLComponent(ssl, binary.BigEndian)
		}
	default:
		// For raw data
		if rawData, ok := data.([]byte); ok {
			component.Component = rawData
		}
	}

	return component
}
