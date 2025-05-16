package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestLongLongTypeParsing(t *testing.T) {
	idlContent := `
module TestMod {
    struct S {
        long long value;
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
	sType, exists := testMod.Types["S"]
	if !exists {
		t.Fatal("Struct S not found in TestMod")
	}
	st, ok := sType.(*idl.StructType)
	if !ok {
		t.Fatal("S is not a struct type")
	}
	if len(st.Fields) != 1 {
		t.Fatalf("Expected 1 field in struct S, got %d", len(st.Fields))
	}
	field := st.Fields[0]
	if field.Name != "value" {
		t.Errorf("Expected field name 'value', got '%s'", field.Name)
	}
	if field.Type.TypeName() != "long long" {
		t.Errorf("Expected field type 'long long', got '%s'", field.Type.TypeName())
	}
}
