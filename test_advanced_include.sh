#!/bin/bash
# Test the IDL processor with advanced include functionality

echo "Testing advanced IDL include functionality..."

# Create output directory
mkdir -p test_advanced_output

# 1. Test nested includes (types.idl -> common.idl -> calculator.idl)
echo -e "\n1. Testing nested includes..."
go run cmd/idlgen/main.go \
   -i examples/idl/calculator.idl \
   -o test_advanced_output/nested \
   -package calculator \
   -I examples/idl

# 2. Test with another set of nested includes (types.idl -> common.idl -> service.idl)
echo -e "\n2. Testing with service.idl..."
go run cmd/idlgen/main.go \
   -i examples/idl/service.idl \
   -o test_advanced_output/service \
   -package service \
   -I examples/idl

# 3. Test circular includes (circular_a.idl <-> circular_b.idl)
echo -e "\n3. Testing circular includes..."
go run cmd/idlgen/main.go \
   -i examples/idl/circular_a.idl \
   -o test_advanced_output/circular \
   -package circular \
   -I examples/idl

# Check if the parser correctly handled circular references
echo -e "\nAll tests completed. Please check the output files in test_advanced_output/"
