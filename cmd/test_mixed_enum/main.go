package main

import (
"fmt"
"os"

"github.com/ifabos/go-corba/idl"
)

func main() {
// Create a parser
parser := idl.NewParser()

// Open the IDL file
file, err := os.Open("examples/idl/mixed_enum_formats.idl")
if err != nil {
fmt.Printf("Error opening file: %v\n", err)
os.Exit(1)
}
defer file.Close()

// Parse the IDL file
err = parser.Parse(file)
if err != nil {
fmt.Printf("Error parsing IDL: %v\n", err)
os.Exit(1)
}

// Get the parsed module
rootModule := parser.GetRootModule()
mixedModule, exists := rootModule.GetSubmodule("MixedTest")
if !exists {
fmt.Println("MixedTest module not found")
os.Exit(1)
}

// Check first enum (with trailing comma)
enum1Type, exists := mixedModule.Types["WithTrailingComma"]
if !exists {
fmt.Println("WithTrailingComma enum not found")
os.Exit(1)
}

// Verify it's an enum
enum1, ok := enum1Type.(*idl.EnumType)
if !ok {
fmt.Println("WithTrailingComma is not an enum type")
os.Exit(1)
}

// Print the elements
fmt.Println("Enum WithTrailingComma has elements:")
for i, elem := range enum1.Elements {
fmt.Printf("%d: %s\n", i, elem)
}

// Check second enum (without trailing comma)
enum2Type, exists := mixedModule.Types["WithoutTrailingComma"]
if !exists {
fmt.Println("WithoutTrailingComma enum not found")
os.Exit(1)
}

// Verify it's an enum
enum2, ok := enum2Type.(*idl.EnumType)
if !ok {
fmt.Println("WithoutTrailingComma is not an enum type")
os.Exit(1)
}

// Print the elements
fmt.Println("\nEnum WithoutTrailingComma has elements:")
for i, elem := range enum2.Elements {
fmt.Printf("%d: %s\n", i, elem)
}

fmt.Println("\nMixed enum formats test passed successfully!")
}
