// Test file for enums inside interfaces
package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestEnumInsideInterface(t *testing.T) {
	idlContent := `
module TestMod {
    interface MyInterface {
        enum Status {
            OK,
            ERROR,
            UNKNOWN
        };
        Status getStatus();
    };
};
`
	parser := idl.NewParser()
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}
	rootModule := parser.GetRootModule()
	testMod, exists := rootModule.GetSubmodule("TestMod")
	if !exists {
		t.Fatal("TestMod module not found")
	}
	ifaceType, exists := testMod.Types["MyInterface"]
	if !exists {
		t.Fatal("MyInterface not found in TestMod")
	}
	iface, ok := ifaceType.(*idl.InterfaceType)
	if !ok {
		t.Fatal("MyInterface is not an interface type")
	}
	// Check for the inner enum
	statusType, exists := iface.Types["Status"]
	if !exists {
		t.Fatal("Status enum not found in MyInterface")
	}
	enum, ok := statusType.(*idl.EnumType)
	if !ok {
		t.Fatal("Status is not an enum type")
	}
	expected := []string{"OK", "ERROR", "UNKNOWN"}
	if len(enum.Elements) != len(expected) {
		t.Fatalf("Expected %d elements in Status, got %d", len(expected), len(enum.Elements))
	}
	for i, v := range expected {
		if enum.Elements[i] != v {
			t.Errorf("Expected element %d to be %s, got %s", i, v, enum.Elements[i])
		}
	}
}
