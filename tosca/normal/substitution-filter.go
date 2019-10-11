package normal

import (
	"encoding/json"
)

//
// SubstitutionFilter
//

type SubstitutionFilter struct {
	CapabilityPropertyConstraints FunctionCallMapMap
	PropertyFilterConstraints     FunctionCallMap
}

func (self *Substitution) NewSubstitutionFilter() *SubstitutionFilter {
	filter := &SubstitutionFilter{
		CapabilityPropertyConstraints: make(FunctionCallMapMap),
		PropertyFilterConstraints:     make(FunctionCallMap),
	}
	self.SubstitutionFilters = append(self.SubstitutionFilters, filter)
	return filter
}

func (self *SubstitutionFilter) Marshalable() interface{} {

	return &struct {
		CapabilityPropertyConstraints FunctionCallMapMap `json:"capabilityPropertyConstraints" yaml:"capabilityPropertyConstraints"`
		PropertyFilterConstraints     FunctionCallMap    `json:"propertyFilterConstraints" yaml:"propertyFilterConstraints"`
	}{
		CapabilityPropertyConstraints: self.CapabilityPropertyConstraints,
		PropertyFilterConstraints:     self.PropertyFilterConstraints,
	}
}

// json.Marshaler interface
func (self *SubstitutionFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.Marshalable())
}

// yaml.Marshaler interface
func (self *SubstitutionFilter) MarshalYAML() (interface{}, error) {
	return self.Marshalable(), nil
}

//
// SubstitutionFilters
//

type SubstitutionFilters []*SubstitutionFilter
