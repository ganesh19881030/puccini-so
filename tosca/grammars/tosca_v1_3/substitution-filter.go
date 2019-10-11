package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// SubstitutionFilter
//
// [TOSCA-Simple-Profile-YAML-v1.3] @ 2.12.3

type SubstitutionFilter struct {
	*Entity `name:"substitution filter"`

	PropertyFilters                PropertyFilters                `read:"properties,{}PropertyFilter"`
	SubstitutionFilterCapabilities SubstitutionFilterCapabilities `read:"capabilities,SubstitutionFilterCapability"`
}

func NewSubstitutionFilter(context *tosca.Context) *SubstitutionFilter {
	return &SubstitutionFilter{
		Entity:                         NewEntity(context),
		PropertyFilters:                make(PropertyFilters),
		SubstitutionFilterCapabilities: make(SubstitutionFilterCapabilities),
	}
}

// tosca.Reader signature
func ReadSubstitutionFilter(context *tosca.Context) interface{} {
	self := NewSubstitutionFilter(context)
	context.ValidateUnsupportedFields(context.ReadFields(self))
	return self
}

func (self *SubstitutionFilter) Normalize(r *normal.SubstitutionFilter) {
	self.PropertyFilters.Normalize(r.PropertyFilterConstraints)
	self.SubstitutionFilterCapabilities.Normalize(r)
}
