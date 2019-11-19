package tosca_v1_3

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// SubstitutionMappings
//
// [TOSCA-Simple-Profile-YAML-v1.2] @ 2.10
// [TOSCA-Simple-Profile-YAML-v1.2] @ 2.11
// [TOSCA-Simple-Profile-YAML-v1.1] @ 2.10
// [TOSCA-Simple-Profile-YAML-v1.1] @ 2.11
//

type SubstitutionMappings struct {
	*Entity `name:"substitution mappings"`

	NodeTypeName        *string             `read:"node_type" require:"node_type"`
	CapabilityMappings  CapabilityMappings  `read:"capabilities,CapabilityMapping"`
	RequirementMappings RequirementMappings `read:"requirements,RequirementMapping"`
	PropertyMappings    Values              `read:"properties,Value"`
	InterfaceMappings   InterfaceMappings   `read:"interfaces,InterfaceMapping"`
	SubstitutionFilter  *SubstitutionFilter `read:"substitution_filter,SubstitutionFilter"`

	NodeType *NodeType `lookup:"node_type,NodeTypeName" json:"-" yaml:"-"`
}

func NewSubstitutionMappings(context *tosca.Context) *SubstitutionMappings {
	return &SubstitutionMappings{
		Entity:           NewEntity(context),
		PropertyMappings: make(Values),
	}
}

// tosca.Reader signature
func ReadSubstitutionMappings(context *tosca.Context) interface{} {
	if context.HasQuirk("substitution_mappings.requirements.list") {
		if context.ReadOverrides == nil {
			context.ReadOverrides = make(map[string]string)
		}
		context.ReadOverrides["RequirementMappings"] = "requirements,{}RequirementMapping"
	}

	self := NewSubstitutionMappings(context)
	if context.Is("map") {
		oldMap := context.Data.(ard.Map)
		newMap := make(ard.Map)
		newMap[context.Name] = oldMap
		context.Data = newMap
		context.ValidateUnsupportedFields(context.ReadFields(self))
	} else if context.ValidateType("map", "string") {
		self.NodeTypeName = context.FieldChild("node_type", context.Data).ReadString()
	}

	return self
}

func (self *SubstitutionMappings) IsRequirementMapped(nodeTemplate *NodeTemplate, requirementName string) bool {
	for _, mapping := range self.RequirementMappings {
		if mapping.NodeTemplate == nodeTemplate {
			if (mapping.RequirementName != nil) && (*mapping.RequirementName == requirementName) {
				return true
			}
		}
	}
	return false
}

func (self *SubstitutionMappings) Normalize(s *normal.ServiceTemplate) *normal.Substitution {
	log.Info("{normalize} substitution mappings")

	if self.NodeType == nil {
		return nil
	}

	t := s.NewSubstitution()

	t.Type = self.NodeType.Name

	if metadata, ok := self.NodeType.GetMetadata(); ok {
		t.TypeMetadata = metadata
	}

	for _, mapping := range self.CapabilityMappings {
		if (mapping.NodeTemplate == nil) || (mapping.CapabilityName == nil) {
			continue
		}

		if n, ok := s.NodeTemplates[mapping.NodeTemplate.Name]; ok {
			if c, ok := n.Capabilities[*mapping.CapabilityName]; ok {
				t.CapabilityMappings[n] = c
			}
		}
	}

	for _, mapping := range self.RequirementMappings {
		if (mapping.NodeTemplate == nil) || (mapping.RequirementName == nil) {
			continue
		}

		if n, ok := s.NodeTemplates[mapping.NodeTemplate.Name]; ok {
			for _, requirement := range n.Requirements {
				if requirement.Name == *mapping.RequirementName {
					t.RequirementMappings[mapping.Entity.Context.Name] = requirement
				}
			}
		}
	}

	if self.PropertyMappings != nil {
		self.PropertyMappings.Normalize(t.PropertyMappings)
	}

	if self.SubstitutionFilter != nil {
		self.SubstitutionFilter.Normalize(t.NewSubstitutionFilter())
	}

	for _, mapping := range self.InterfaceMappings {
		if (mapping.NodeTemplate == nil) || (mapping.InterfaceName == nil) {
			continue
		}

		if n, ok := s.NodeTemplates[mapping.NodeTemplate.Name]; ok {
			t.InterfaceMappings[n] = *mapping.InterfaceName
		}
	}

	return t
}

func (self SubstitutionMappingsList) Normalize(s *normal.ServiceTemplate) {
	for _, sub := range self {
		sub.Normalize(s)
	}
}

//SubstitutionMappingsList ...
type SubstitutionMappingsList []*SubstitutionMappings
