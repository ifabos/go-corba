#!/bin/bash
# Test the IDL processor with include functionality

echo "Testing IDL include functionality..."

# Create output directory
mkdir -p test_output

# Compile the IDL file with include directive
go run cmd/idlgen/main.go \
   -i examples/idl/calculator.idl \
   -o test_output \
   -package calculator \
   -I examples/idl

# Check if the generated files exist
if [ -f "test_output/calculator.go" ] && [ -f "test_output/common.go" ]; then
    echo "SUCCESS: Generated both calculator.go and common.go"
else
    echo "FAILED: Did not generate expected files"
    exit 1
fi

# Check if the Common module types are referenced in calculator.go
if grep -q "Common.Status" test_output/calculator.go; then
    echo "SUCCESS: Common module references found in calculator.go"
else
    echo "FAILED: Common module references not found in calculator.go"
    exit 1
fi

echo "All tests passed successfully!"
