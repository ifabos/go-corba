// Package gocorba is the root package for the Go CORBA SDK.
// It provides a Go implementation of the Common Object Request Broker Architecture (CORBA).
package gocorba

import (
	"github.com/ifabos/go-corba/corba"
)

// Init initializes and returns a new CORBA ORB
func Init() *corba.ORB {
	return corba.Init()
}

// NewContext creates a new CORBA context
func NewContext() *corba.Context {
	return corba.NewContext()
}

// NewInterfaceRepository creates a new Interface Repository
func NewInterfaceRepository() corba.InterfaceRepository {
	return corba.NewInterfaceRepository()
}

// Re-export important types from the corba package
type (
	// ORB represents the Object Request Broker
	ORB = corba.ORB

	// Server represents a CORBA server
	Server = corba.Server

	// Client represents a CORBA client
	Client = corba.Client

	// Context represents a CORBA context
	Context = corba.Context

	// ObjectRef represents a reference to a CORBA object
	ObjectRef = corba.ObjectRef

	// ServerBinding represents a binding between an object and a service name
	ServerBinding = corba.ServerBinding

	// Interface Repository types
	IRObject            = corba.IRObject
	Container           = corba.Container
	Contained           = corba.Contained
	Repository          = corba.Repository
	InterfaceDef        = corba.InterfaceDef
	OperationDef        = corba.OperationDef
	TypeCode            = corba.TypeCode
	AttributeDef        = corba.AttributeDef
	StructDef           = corba.StructDef
	ExceptionDef        = corba.ExceptionDef
	EnumDef             = corba.EnumDef
	UnionDef            = corba.UnionDef
	AliasDef            = corba.AliasDef
	InterfaceRepository = corba.InterfaceRepository
	IRClient            = corba.IRClient
)

// Re-export enumerations
const (
	// Parameter modes
	PARAM_IN    = corba.PARAM_IN
	PARAM_OUT   = corba.PARAM_OUT
	PARAM_INOUT = corba.PARAM_INOUT

	// Definition kinds
	DK_NONE      = corba.DK_NONE
	DK_ALL       = corba.DK_ALL
	DK_INTERFACE = corba.DK_INTERFACE
	DK_OPERATION = corba.DK_OPERATION
	DK_ATTRIBUTE = corba.DK_ATTRIBUTE
	DK_EXCEPTION = corba.DK_EXCEPTION
)
