package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	// Parse command line flags
	inputFile := flag.String("i", "", "Input IDL file to process")
	outputDir := flag.String("o", "generated", "Output directory for generated Go files")
	packageName := flag.String("package", "generated", "Go package name for generated files")
	includesStr := flag.String("include", "", "Comma-separated list of import paths to include")
	includeDirs := flag.String("I", "", "Comma-separated list of directories to search for included IDL files")
	help := flag.Bool("help", false, "Show help message")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *version {
		showVersion()
		return
	}

	if *inputFile == "" {
		fmt.Println("Error: Input file is required")
		showHelp()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Parse the IDL file
	parser := idl.NewParser()

	// Set up include path list
	var includePathList []string
	if *includeDirs != "" {
		includePathList = strings.Split(*includeDirs, ",")
	}

	// Add the directory of the input file to the include path
	inputDir := filepath.Dir(*inputFile)
	includePathList = append([]string{inputDir}, includePathList...)

	// Set the include directories in the parser
	parser.SetIncludeDirs(includePathList)

	// Set the current file being processed
	parser.SetCurrentFile(*inputFile)

	// Set up include handler
	parser.SetIncludeHandler(func(path string) (io.Reader, error) {
		// This handler will be called when an #include directive is encountered
		// But most of the include resolution logic is now in the parser itself
		for _, dir := range includePathList {
			if dir == "" {
				continue
			}

			fullPath := filepath.Join(dir, path)
			file, err := os.Open(fullPath)
			if err == nil {
				return file, nil
			}
		}

		// Try to open the file directly as a last resort
		return os.Open(path)
	})

	// Read and parse the IDL file
	idlData, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	if err := parser.Parse(strings.NewReader(string(idlData))); err != nil {
		fmt.Printf("Error parsing IDL: %v\n", err)
		os.Exit(1)
	}

	// Create a code generator
	generator := idl.NewGenerator(parser.GetRootModule(), *outputDir)
	generator.SetPackageName(*packageName)

	// Add any requested imports
	if *includesStr != "" {
		includes := strings.Split(*includesStr, ",")
		for _, inc := range includes {
			if inc != "" {
				generator.AddInclude(inc)
			}
		}
	}

	// Always include the CORBA package
	generator.AddInclude("github.com/ifabos/go-corba/corba")

	// Generate the code
	if err := generator.Generate(); err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated Go code in %s\n", *outputDir)
}

func showHelp() {
	fmt.Println("IDL to Go Code Generator")
	fmt.Println("Usage: idlgen [options]")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Include Handling:")
	fmt.Println("  The IDL compiler supports both system includes (#include <file.idl>) and")
	fmt.Println("  user includes (#include \"file.idl\"). System includes are searched in the")
	fmt.Println("  include path specified with -I. User includes are first searched relative")
	fmt.Println("  to the including file, then in the include path.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  Basic usage:")
	fmt.Println("    idlgen -i myservice.idl -o ./generated -package myservice")
	fmt.Println("")
	fmt.Println("  With include directories:")
	fmt.Println("    idlgen -i myservice.idl -I /path/to/includes,/another/path -o ./generated")
}

func showVersion() {
	fmt.Println("CORBA IDL-to-Go Generator v1.0.0")
	fmt.Println("Part of the Go CORBA SDK")
}
