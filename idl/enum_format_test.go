package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestEnumParserFormats(t *testing.T) {
	tests := []struct {
		name            string
		idlContent      string
		expectedModule  string
		expectedEnum    string
		expectedMembers []string
	}{
		{
			name: "Standard enum with trailing comma",
			idlContent: `
module Test {
    enum Colors {
        RED,
        GREEN,
        BLUE,
    };
};`,
			expectedModule:  "Test",
			expectedEnum:    "Colors",
			expectedMembers: []string{"RED", "GREEN", "BLUE"},
		},
		{
			name: "Enum without trailing comma",
			idlContent: `
module Test {
    enum Status {
        PENDING,
        ACTIVE,
        COMPLETED
    };
};`,
			expectedModule:  "Test",
			expectedEnum:    "Status",
			expectedMembers: []string{"PENDING", "ACTIVE", "COMPLETED"},
		},
		{
			name: "Enum with brackets on same line",
			idlContent: `
module Test {
    enum SameLineBrackets { ONE, TWO, THREE };
};`,
			expectedModule:  "Test",
			expectedEnum:    "SameLineBrackets",
			expectedMembers: []string{"ONE", "TWO", "THREE"},
		},
		{
			name: "Enum with opening bracket on same line, closing bracket on new line",
			idlContent: `
module Test {
    enum MixedBrackets { 
        ALPHA, 
        BETA, 
        GAMMA
    };
};`,
			expectedModule:  "Test",
			expectedEnum:    "MixedBrackets",
			expectedMembers: []string{"ALPHA", "BETA", "GAMMA"},
		},
		{
			name: "Enum with both brackets on different lines",
			idlContent: `
module Test {
    enum NewLineBrackets 
    {
        FIRST,
        SECOND,
        THIRD
    };
};`,
			expectedModule:  "Test",
			expectedEnum:    "NewLineBrackets",
			expectedMembers: []string{"FIRST", "SECOND", "THIRD"},
		},
		{
			name: "Enum with single element",
			idlContent: `
module Test {
    enum SingleElement { SOLO };
};`,
			expectedModule:  "Test",
			expectedEnum:    "SingleElement",
			expectedMembers: []string{"SOLO"},
		},
		{
			name: "Enum with single element and trailing comma",
			idlContent: `
module Test {
    enum SingleElementTrailingComma { 
        SOLO, 
    };
};`,
			expectedModule:  "Test",
			expectedEnum:    "SingleElementTrailingComma",
			expectedMembers: []string{"SOLO"},
		},
		{
			name: "Enum with elements on same line",
			idlContent: `
module Test {
    enum SameLineElements { FIRST, SECOND, THIRD };
};`,
			expectedModule:  "Test",
			expectedEnum:    "SameLineElements",
			expectedMembers: []string{"FIRST", "SECOND", "THIRD"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a parser
			parser := idl.NewParser()

			// Parse the IDL from a byte buffer
			err := parser.Parse(bytes.NewBufferString(tc.idlContent))
			if err != nil {
				t.Fatalf("Error parsing IDL: %v", err)
			}

			// Get the parsed module
			rootModule := parser.GetRootModule()
			testModule, exists := rootModule.GetSubmodule(tc.expectedModule)
			if !exists {
				t.Fatalf("Module %s not found", tc.expectedModule)
			}

			// Check for the enum
			enumType, exists := testModule.Types[tc.expectedEnum]
			if !exists {
				t.Fatalf("Enum %s not found", tc.expectedEnum)
			}

			// Verify it's an enum
			enum, ok := enumType.(*idl.EnumType)
			if !ok {
				t.Fatalf("%s is not an enum type", tc.expectedEnum)
			}

			// Verify the number of elements
			if len(enum.Elements) != len(tc.expectedMembers) {
				t.Fatalf("Expected %d elements in enum %s, got %d",
					len(tc.expectedMembers), tc.expectedEnum, len(enum.Elements))
			}

			// Verify each element
			for i, expected := range tc.expectedMembers {
				if enum.Elements[i] != expected {
					t.Errorf("Expected element %d to be %s, got %s", i, expected, enum.Elements[i])
				}
			}
		})
	}
}
