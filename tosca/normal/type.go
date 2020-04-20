package normal

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/reflection"
)

//
// Type
//

type Type struct {
	Name     string            `json:"-" yaml:"-"`
	Metadata map[string]string `json:"metadata" yaml:"metadata"`
}

func NewType(name string) *Type {
	return &Type{
		Name:     name,
		Metadata: make(map[string]string),
	}
}

//
// Types
//

type Types map[string]*Type

func NewTypes(names ...string) Types {
	types := make(Types)
	for _, name := range names {
		types[name] = NewType(name)
	}
	return types
}

func GetHierarchyTypes(hierarchy *tosca.Hierarchy) Types {
	types := make(Types)
	n := hierarchy
	for (n != nil) && (n.EntityPtr != nil) {
		name := n.GetContext().Name
		type_ := NewType(name)
		if metadata, ok := GetMetadata(n.EntityPtr); ok {
			type_.Metadata = metadata
		}
		types[name] = type_
		n = n.Parent
	}
	return types
}

func GetTypes(hierarchy *tosca.Hierarchy, entityPtr interface{}) (Types, bool) {
	if childHierarchy, ok := hierarchy.Find(entityPtr); ok {
		return GetHierarchyTypes(childHierarchy), true
	}
	return nil, false
}

func GetTypes2(entityPtr interface{}) (Types, bool) {

	if entityPtr == nil {
		return nil, false
	}

	xtypes := make(Types)

	currType := entityPtr
	for currType != nil {
		name := reflection.GetEntityName(currType)
		xtype := NewType(name)
		if metadata, ok := GetMetadata(currType); ok {
			xtype.Metadata = metadata
		}
		//ratype.Metadata = currType.Metadata
		xtypes[name] = xtype
		currType = reflection.GetEntityParent(currType)
	}

	return xtypes, true

}
