// Package corba provides a CORBA implementation in Go
package corba

import (
	"fmt"
)

// InterfaceRepositoryServant is a CORBA servant for the Interface Repository
type InterfaceRepositoryServant struct {
	ir InterfaceRepository
}

// NewInterfaceRepositoryServant creates a new servant for the Interface Repository
func NewInterfaceRepositoryServant(ir InterfaceRepository) *InterfaceRepositoryServant {
	return &InterfaceRepositoryServant{
		ir: ir,
	}
}

// Dispatch handles incoming CORBA method calls to the Interface Repository
func (irs *InterfaceRepositoryServant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "lookup_id":
		if len(args) < 1 {
			return nil, fmt.Errorf("lookup_id requires repository ID argument")
		}

		id, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("repository ID must be a string")
		}

		obj, err := irs.ir.GetRepository().LookupId(id)
		if err != nil {
			return nil, err
		}

		return obj, nil

	case "get_primitive_tc":
		if len(args) < 1 {
			return nil, fmt.Errorf("get_primitive_tc requires kind argument")
		}

		kindInt, ok := args[0].(int)
		if !ok {
			return nil, fmt.Errorf("kind must be an integer")
		}

		kind := DefinitionKind(kindInt)
		primitiveID := fmt.Sprintf("IDL:omg.org/CORBA/%s:1.0", kind)

		obj, err := irs.ir.GetRepository().LookupId(primitiveID)
		if err != nil {
			return nil, err
		}

		if tc, ok := obj.(TypeCode); ok {
			return tc, nil
		}

		return nil, fmt.Errorf("type not found")

	case "describe_interface":
		if len(args) < 1 {
			return nil, fmt.Errorf("describe_interface requires repository ID argument")
		}

		id, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("repository ID must be a string")
		}

		iface, err := irs.ir.LookupInterface(id)
		if err != nil {
			return nil, err
		}

		// Return detailed interface description as a map
		result := make(map[string]interface{})
		result["id"] = iface.Id()
		result["name"] = iface.Name()

		// Get operations
		operations := []map[string]interface{}{}
		for _, obj := range iface.Contents(DK_OPERATION) {
			if op, ok := obj.(OperationDef); ok {
				opMap := make(map[string]interface{})
				opMap["name"] = op.Name()
				opMap["id"] = op.Id()

				// Add parameters
				params := []map[string]interface{}{}
				for _, param := range op.Params() {
					paramMap := make(map[string]interface{})
					paramMap["name"] = param.Name
					paramMap["type_id"] = param.Type.Id()
					paramMap["mode"] = int(param.Mode)
					params = append(params, paramMap)
				}
				opMap["parameters"] = params

				// Add return type
				if op.Result() != nil {
					opMap["result_type"] = op.Result().Id()
				}

				operations = append(operations, opMap)
			}
		}
		result["operations"] = operations

		// Get attributes
		attributes := []map[string]interface{}{}
		for _, obj := range iface.Contents(DK_ATTRIBUTE) {
			if attr, ok := obj.(AttributeDef); ok {
				attrMap := make(map[string]interface{})
				attrMap["name"] = attr.Name()
				attrMap["id"] = attr.Id()
				attrMap["type_id"] = attr.Type().Id()
				attrMap["mode"] = attr.Mode()
				attributes = append(attributes, attrMap)
			}
		}
		result["attributes"] = attributes

		// Get base interfaces
		baseIds := []string{}
		for _, base := range iface.BaseInterfaces() {
			baseIds = append(baseIds, base.Id())
		}
		result["base_interfaces"] = baseIds

		return result, nil

	case "is_a":
		if len(args) < 2 {
			return nil, fmt.Errorf("is_a requires object and interface ID arguments")
		}

		obj := args[0]

		interfaceID, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("interface ID must be a string")
		}

		return irs.ir.IsA(obj, interfaceID)

	case "lookup_name":
		if len(args) < 1 {
			return nil, fmt.Errorf("lookup_name requires name argument")
		}

		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("name must be a string")
		}

		var levels int = 1 // Default to one level
		if len(args) > 1 {
			if l, ok := args[1].(int); ok {
				levels = l
			}
		}

		var kind DefinitionKind = DK_ALL // Default to all kinds
		if len(args) > 2 {
			if k, ok := args[2].(int); ok {
				kind = DefinitionKind(k)
			}
		}

		objects := irs.ir.GetRepository().LookupName(name, levels, kind)
		return objects, nil

	case "get_contents":
		var kind DefinitionKind = DK_ALL // Default to all kinds
		if len(args) > 0 {
			if k, ok := args[0].(int); ok {
				kind = DefinitionKind(k)
			}
		}

		contents := irs.ir.GetRepository().Contents(kind)
		return contents, nil

	case "create_module":
		if len(args) < 2 {
			return nil, fmt.Errorf("create_module requires id and name arguments")
		}

		id, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("id must be a string")
		}

		name, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("name must be a string")
		}

		return irs.ir.GetRepository().CreateModule(id, name)

	case "create_interface":
		if len(args) < 2 {
			return nil, fmt.Errorf("create_interface requires id and name arguments")
		}

		id, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("id must be a string")
		}

		name, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("name must be a string")
		}

		return irs.ir.GetRepository().CreateInterface(id, name)

	case "register_servant":
		if len(args) < 2 {
			return nil, fmt.Errorf("register_servant requires servant and id arguments")
		}

		servant := args[0]

		id, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("id must be a string")
		}

		return nil, irs.ir.RegisterServant(servant, id)

	default:
		return nil, fmt.Errorf("unknown operation: %s", methodName)
	}
}

// IRClient provides a client interface to the Interface Repository
type IRClient struct {
	objectRef *ObjectRef
}

// NewIRClient creates a client for the Interface Repository
func NewIRClient(objectRef *ObjectRef) *IRClient {
	return &IRClient{
		objectRef: objectRef,
	}
}

// LookupID finds an object in the Interface Repository by its repository ID
func (client *IRClient) LookupID(id string) (interface{}, error) {
	return client.objectRef.Invoke("lookup_id", id)
}

// DescribeInterface returns a description of an interface
func (client *IRClient) DescribeInterface(id string) (map[string]interface{}, error) {
	result, err := client.objectRef.Invoke("describe_interface", id)
	if err != nil {
		return nil, err
	}

	if desc, ok := result.(map[string]interface{}); ok {
		return desc, nil
	}

	return nil, fmt.Errorf("unexpected result type from describe_interface")
}

// IsA checks if an object supports a specific interface
func (client *IRClient) IsA(obj interface{}, interfaceID string) (bool, error) {
	result, err := client.objectRef.Invoke("is_a", obj, interfaceID)
	if err != nil {
		return false, err
	}

	if isA, ok := result.(bool); ok {
		return isA, nil
	}

	return false, fmt.Errorf("unexpected result type from is_a")
}

// LookupName finds objects in the Interface Repository by name
func (client *IRClient) LookupName(name string, levels int, kind DefinitionKind) ([]IRObject, error) {
	result, err := client.objectRef.Invoke("lookup_name", name, levels, int(kind))
	if err != nil {
		return nil, err
	}

	if objects, ok := result.([]IRObject); ok {
		return objects, nil
	}

	return nil, fmt.Errorf("unexpected result type from lookup_name")
}
