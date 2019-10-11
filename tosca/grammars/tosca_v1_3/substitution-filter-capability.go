package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// SubstitutionFilterCapability
//
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.5.2
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.5.2
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.4.2
//

type SubstitutionFilterCapability struct {
	*Entity `name:"substitution filter capability"`
	Name    string

	PropertyFilters PropertyFilters `read:"properties,{}PropertyFilter"`
}

func NewSubstitutionFilterCapability(context *tosca.Context) *SubstitutionFilterCapability {
	return &SubstitutionFilterCapability{
		Entity:          NewEntity(context),
		Name:            context.Name,
		PropertyFilters: make(PropertyFilters),
	}
}

// tosca.Reader signature
func ReadSubstitutionFilterCapability(context *tosca.Context) interface{} {
	self := NewSubstitutionFilterCapability(context)
	context.ValidateUnsupportedFields(context.ReadFields(self))
	return self
}

// tosca.Mappable interface
func (self *SubstitutionFilterCapability) GetKey() string {
	return self.Name
}

func (self SubstitutionFilterCapability) Normalize(r *normal.SubstitutionFilter) normal.FunctionCallMap {
	if len(self.PropertyFilters) == 0 {
		return nil
	}

	functionCallMap := make(normal.FunctionCallMap)
	r.CapabilityPropertyConstraints[self.Name] = functionCallMap
	self.PropertyFilters.Normalize(functionCallMap)

	return functionCallMap
}

//
// SubstitutionFilterCapability
//

type SubstitutionFilterCapabilities map[string]*SubstitutionFilterCapability

func (self SubstitutionFilterCapabilities) Normalize(r *normal.SubstitutionFilter) {
	for _, substitutionFilterCapability := range self {
		substitutionFilterCapability.Normalize(r)
	}
}
