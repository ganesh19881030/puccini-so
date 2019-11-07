package normal

import (
	"encoding/json"
)

//
// Substitution
//

type Substitution struct {
	ServiceTemplate     *ServiceTemplate
	Type                string
	TypeMetadata        map[string]string
	CapabilityMappings  map[*NodeTemplate]*Capability
	RequirementMappings map[*NodeTemplate]string
	PropertyMappings    Constrainables
	InterfaceMappings   map[*NodeTemplate]string
	SubstitutionFilters SubstitutionFilters
}

func (self *ServiceTemplate) NewSubstitution() *Substitution {
	substitutionMappings := &Substitution{
		ServiceTemplate:     self,
		TypeMetadata:        make(map[string]string),
		CapabilityMappings:  make(map[*NodeTemplate]*Capability),
		RequirementMappings: make(map[*NodeTemplate]string),
		PropertyMappings:    make(Constrainables),
		InterfaceMappings:   make(map[*NodeTemplate]string),
		SubstitutionFilters: make(SubstitutionFilters, 0),
	}
	self.Substitution = append(self.Substitution, substitutionMappings)
	return substitutionMappings
}

type Substitutions []*Substitution

func (self *Substitution) Marshalable() interface{} {
	capabilityMappings := make(map[string]string)
	for n, c := range self.CapabilityMappings {
		capabilityMappings[n.Name] = c.Name
	}

	requirementMappings := make(map[string]string)
	for n, r := range self.RequirementMappings {
		requirementMappings[n.Name] = r
	}

	interfaceMappings := make(map[string]string)
	for n, i := range self.InterfaceMappings {
		interfaceMappings[n.Name] = i
	}

	return &struct {
		Type                string              `json:"type" yaml:"type"`
		TypeMetadata        map[string]string   `json:"typeMetadata" yaml:"typeMetadata"`
		CapabilityMappings  map[string]string   `json:"capabilityMappings" yaml:"capabilityMappings"`
		RequirementMappings map[string]string   `json:"requirementMappings" yaml:"requirementMappings"`
		PropertyMappings    Constrainables      `json:"propertyMappings" yaml:"propertyMappings"`
		InterfaceMappings   map[string]string   `json:"interfaceMappings" yaml:"interfaceMappings"`
		SubstitutionFilters SubstitutionFilters `json:"substitutionFilters" yaml:"substitutionFilters"`
	}{
		Type:                self.Type,
		TypeMetadata:        self.TypeMetadata,
		CapabilityMappings:  capabilityMappings,
		RequirementMappings: requirementMappings,
		PropertyMappings:    make(Constrainables),
		InterfaceMappings:   interfaceMappings,
		SubstitutionFilters: make(SubstitutionFilters, 0),
	}
}

// json.Marshaler interface
func (self *Substitution) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.Marshalable())
}

// yaml.Marshaler interface
func (self *Substitution) MarshalYAML() (interface{}, error) {
	return self.Marshalable(), nil
}
