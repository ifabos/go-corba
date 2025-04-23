// Package naming provides a CORBA Naming Service implementation
package corba

import (
	"fmt"
	"strings"
)

// NamingServiceServant is a CORBA servant that implements the Naming Service
type NamingServiceServant struct {
	rootContext *NamingContext
}

// NewNamingServiceServant creates a new naming service servant
func NewNamingServiceServant(orb *ORB) *NamingServiceServant {
	return &NamingServiceServant{
		rootContext: NewNamingContext(orb, "NameService"),
	}
}

// GetRootContext returns the root naming context
func (ns *NamingServiceServant) GetRootContext() *NamingContext {
	return ns.rootContext
}

// Dispatch handles incoming CORBA method calls to the naming service
func (ns *NamingServiceServant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "bind":
		if len(args) < 2 {
			return nil, fmt.Errorf("bind requires 2 arguments")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		obj := args[1]
		return nil, ns.rootContext.Bind(name, obj)

	case "rebind":
		if len(args) < 2 {
			return nil, fmt.Errorf("rebind requires 2 arguments")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		obj := args[1]
		return nil, ns.rootContext.Rebind(name, obj)

	case "bind_context":
		if len(args) < 2 {
			return nil, fmt.Errorf("bind_context requires 2 arguments")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		ctx, ok := args[1].(*NamingContext)
		if !ok {
			return nil, ErrInvalidContext
		}

		return nil, ns.rootContext.BindContext(name, ctx)

	case "rebind_context":
		if len(args) < 2 {
			return nil, fmt.Errorf("rebind_context requires 2 arguments")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		ctx, ok := args[1].(*NamingContext)
		if !ok {
			return nil, ErrInvalidContext
		}

		return nil, ns.rootContext.RebindContext(name, ctx)

	case "resolve":
		if len(args) < 1 {
			return nil, fmt.Errorf("resolve requires 1 argument")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		return ns.rootContext.Resolve(name)

	case "unbind":
		if len(args) < 1 {
			return nil, fmt.Errorf("unbind requires 1 argument")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		return nil, ns.rootContext.Unbind(name)

	case "list":
		return ns.rootContext.List(), nil

	case "new_context":
		// Create a new context that is not bound to the naming tree
		return NewNamingContext(ns.rootContext.orb, "temp"), nil

	case "bind_new_context":
		if len(args) < 1 {
			return nil, fmt.Errorf("bind_new_context requires 1 argument")
		}

		name, err := parseCorbaName(args[0])
		if err != nil {
			return nil, err
		}

		// Create new context
		newContext := NewNamingContext(ns.rootContext.orb, "nc_"+name.String())

		// Bind it to the specified name
		err = ns.rootContext.BindContext(name, newContext)
		if err != nil {
			return nil, err
		}

		return newContext, nil

	default:
		return nil, fmt.Errorf("method %s not supported by naming service", methodName)
	}
}

// parseCorbaName parses a CORBA name from a string or interface{} representation
func parseCorbaName(nameArg interface{}) (Name, error) {
	switch n := nameArg.(type) {
	case Name:
		return n, nil
	case string:
		return parseStringName(n)
	case []interface{}:
		// Assume array of name components
		result := make(Name, 0, len(n))
		for _, comp := range n {
			m, ok := comp.(map[string]string)
			if !ok {
				return nil, ErrInvalidNameFormat
			}

			id, ok := m["id"]
			if !ok {
				return nil, ErrInvalidNameFormat
			}

			kind := m["kind"] // kind is optional

			result = append(result, NameComponent{ID: id, Kind: kind})
		}
		return result, nil
	default:
		return nil, ErrInvalidNameFormat
	}
}

// parseStringName parses a string into a CORBA Name
// Format: "id1.kind1/id2.kind2/id3.kind3"
// Kind is optional: "id1/id2/id3"
func parseStringName(s string) (Name, error) {
	if s == "" {
		return nil, ErrInvalidNameFormat
	}

	components := strings.Split(s, "/")
	result := make(Name, 0, len(components))

	for _, comp := range components {
		if comp == "" {
			continue // Skip empty components
		}

		parts := strings.SplitN(comp, ".", 2)
		id := parts[0]

		var kind string
		if len(parts) > 1 {
			kind = parts[1]
		}

		result = append(result, NameComponent{ID: id, Kind: kind})
	}

	if len(result) == 0 {
		return nil, ErrInvalidNameFormat
	}

	return result, nil
}
