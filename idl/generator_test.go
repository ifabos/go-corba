package idl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestGeneratorBasicStruct(t *testing.T) {
	mod := idl.NewModule("TestMod")
	st := &idl.StructType{
		Name:   "S",
		Module: "TestMod",
		Fields: []idl.StructField{{Name: "value", Type: &idl.SimpleType{Name: idl.TypeLong}}},
	}
	mod.AddType("S", st)

dir := t.TempDir()
	gen := idl.NewGenerator(mod, dir)
	gen.SetPackageName("testpkg")
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generator failed: %v", err)
	}
	// Check that the file was generated
	filename := filepath.Join(dir, "testmod", "s.go")
	if _, err := os.Stat(filename); err != nil {
		t.Errorf("Expected generated file %s, got error: %v", filename, err)
	}
}

func TestGeneratorEnum(t *testing.T) {
	mod := idl.NewModule("TestMod")
	en := &idl.EnumType{
		Name:     "Color",
		Module:   "TestMod",
		Elements: []string{"RED", "GREEN", "BLUE"},
	}
	mod.AddType("Color", en)

dir := t.TempDir()
	gen := idl.NewGenerator(mod, dir)
	gen.SetPackageName("testpkg")
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generator failed: %v", err)
	}
	filename := filepath.Join(dir, "testmod", "color.go")
	if _, err := os.Stat(filename); err != nil {
		t.Errorf("Expected generated file %s, got error: %v", filename, err)
	}
}
