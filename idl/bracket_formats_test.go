package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

// TestEnumWithSameLineBrackets tests an enum with brackets on the same line
func TestEnumWithSameLineBrackets(t *testing.T) {
	idlContent := `
module Test {
    enum SameLineBrackets { ONE, TWO, THREE };
};`

	parser := idl.NewParser()
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}

	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("Test")
	if !exists {
		t.Fatal("Test module not found")
	}

	enumType, exists := testModule.Types["SameLineBrackets"]
	if !exists {
		t.Fatal("SameLineBrackets enum not found")
	}

	enum, ok := enumType.(*idl.EnumType)
	if !ok {
		t.Fatal("SameLineBrackets is not an enum type")
	}

	expectedElements := []string{"ONE", "TWO", "THREE"}
	if len(enum.Elements) != len(expectedElements) {
		t.Fatalf("Expected %d elements, got %d", len(expectedElements), len(enum.Elements))
	}

	for i, expected := range expectedElements {
		if enum.Elements[i] != expected {
			t.Errorf("Expected element %d to be %s, got %s", i, expected, enum.Elements[i])
		}
	}
}

// TestEnumWithMixedBrackets tests an enum with opening bracket on same line, closing on new line
func TestEnumWithMixedBrackets(t *testing.T) {
	idlContent := `
module Test {
    enum MixedBrackets { 
        ALPHA, 
        BETA, 
        GAMMA
    };
};`

	parser := idl.NewParser()
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}

	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("Test")
	if !exists {
		t.Fatal("Test module not found")
	}

	enumType, exists := testModule.Types["MixedBrackets"]
	if !exists {
		t.Fatal("MixedBrackets enum not found")
	}

	enum, ok := enumType.(*idl.EnumType)
	if !ok {
		t.Fatal("MixedBrackets is not an enum type")
	}

	expectedElements := []string{"ALPHA", "BETA", "GAMMA"}
	if len(enum.Elements) != len(expectedElements) {
		t.Fatalf("Expected %d elements, got %d", len(expectedElements), len(enum.Elements))
	}

	for i, expected := range expectedElements {
		if enum.Elements[i] != expected {
			t.Errorf("Expected element %d to be %s, got %s", i, expected, enum.Elements[i])
		}
	}
}

// TestEnumWithNewLineBrackets tests an enum with both brackets on new lines
func TestEnumWithNewLineBrackets(t *testing.T) {
	idlContent := `
module Test {
    enum NewLineBrackets 
    {
        FIRST,
        SECOND,
        THIRD
    };
};`

	parser := idl.NewParser()
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}

	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("Test")
	if !exists {
		t.Fatal("Test module not found")
	}

	enumType, exists := testModule.Types["NewLineBrackets"]
	if !exists {
		t.Fatal("NewLineBrackets enum not found")
	}

	enum, ok := enumType.(*idl.EnumType)
	if !ok {
		t.Fatal("NewLineBrackets is not an enum type")
	}

	expectedElements := []string{"FIRST", "SECOND", "THIRD"}
	if len(enum.Elements) != len(expectedElements) {
		t.Fatalf("Expected %d elements, got %d", len(expectedElements), len(enum.Elements))
	}

	for i, expected := range expectedElements {
		if enum.Elements[i] != expected {
			t.Errorf("Expected element %d to be %s, got %s", i, expected, enum.Elements[i])
		}
	}
}
