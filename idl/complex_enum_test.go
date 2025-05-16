// Test file for complex enum parsing scenarios
package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestComplexEnumParsing(t *testing.T) {
	testCases := []struct {
		name            string
		idlContent      string
		expectedModules map[string][]string // module -> enums
	}{
		{
			name: "Multiple enums with trailing commas",
			idlContent: `
module TestModule {
    enum Colors {
        RED,
        GREEN,
        BLUE,
    };
    
    enum Shapes {
        CIRCLE,
        SQUARE,
        TRIANGLE,
    };
};
`,
			expectedModules: map[string][]string{
				"TestModule": {"Colors", "Shapes"},
			},
		},
		{
			name: "Enum with no trailing comma",
			idlContent: `
module TestModule2 {
    enum Sizes {
        SMALL,
        MEDIUM,
        LARGE
    };
};
`,
			expectedModules: map[string][]string{
				"TestModule2": {"Sizes"},
			},
		},
		{
			name: "Nested modules with enums",
			idlContent: `
module Outer {
    module Inner {
        enum Status {
            PENDING,
            ACTIVE,
            COMPLETED,
        };
    };
};
`,
			expectedModules: map[string][]string{
				"Outer": {},
				"Inner": {"Status"},
			},
		},
	}

	for _, tc := range testCases {
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

			// Check each expected module
			for moduleName, expectedEnums := range tc.expectedModules {
				// Find the module (handling nested modules)
				var module *idl.Module
				if moduleName == "Inner" {
					// Special case for nested module
					outerModule, exists := rootModule.GetSubmodule("Outer")
					if !exists {
						t.Fatalf("Module Outer not found")
					}
					module, exists = outerModule.GetSubmodule("Inner")
					if !exists {
						t.Fatalf("Module Inner not found")
					}
				} else {
					var exists bool
					module, exists = rootModule.GetSubmodule(moduleName)
					if !exists {
						t.Fatalf("Module %s not found", moduleName)
					}
				}

				// Check for the expected enums
				for _, enumName := range expectedEnums {
					enumType, exists := module.Types[enumName]
					if !exists {
						t.Fatalf("Enum %s not found in module %s", enumName, moduleName)
					}

					// Verify it's an enum
					_, ok := enumType.(*idl.EnumType)
					if !ok {
						t.Fatalf("%s is not an enum type", enumName)
					}
				}
			}
		})
	}
}
