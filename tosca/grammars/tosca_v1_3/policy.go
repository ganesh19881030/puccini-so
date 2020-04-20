package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// Policy
//
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.8.6
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.7.6
//

type Policy struct {
	*Entity `name:"policy"`
	Name    string `namespace:""`

	PolicyTypeName                 *string            `read:"type" require:"type"`
	Description                    *string            `read:"description" inherit:"description,PolicyType"`
	Properties                     Values             `read:"properties,Value"`
	TargetNodeTemplateOrGroupNames *[]string          `read:"targets"`
	TriggerDefinitions             TriggerDefinitions `read:"triggers,TriggerDefinition" inherit:"triggers,PolicyType"`

	PolicyType          *PolicyType   `lookup:"type,PolicyTypeName,PolicyType" json:"-" yaml:"-"`
	TargetNodeTemplates NodeTemplates `lookup:"targets,TargetNodeTemplateOrGroupNames,NodeTemplate" json:"-" yaml:"-"`
	TargetGroups        Groups        `lookup:"targets,TargetNodeTemplateOrGroupNames,Group,no" json:"-" yaml:"-"`
}

func NewPolicy(context *tosca.Context) *Policy {
	return &Policy{
		Entity:             NewEntity(context),
		Name:               context.Name,
		Properties:         make(Values),
		TriggerDefinitions: make(TriggerDefinitions),
	}
}

// tosca.Reader signature
func ReadPolicy(context *tosca.Context) interface{} {
	self := NewPolicy(context)
	context.ValidateUnsupportedFields(context.ReadFields(self))
	return self
}

// tosca.Renderable interface
func (self *Policy) Render() {
	log.Infof("{render} policy: %s", self.Name)

	if self.PolicyType == nil {
		return
	}

	self.Properties.RenderProperties(self.PolicyType.PropertyDefinitions, "property", self.Context.FieldChild("properties", nil))

	// Validate targets
	for index, nodeTemplate := range self.TargetNodeTemplates {
		compatible := false
		for _, nodeType := range self.PolicyType.TargetNodeTypes {
			if self.Context.ReadFromDb {
				if IsCompatible(nodeType, nodeTemplate.NodeType) {
					compatible = true
					break
				}
			} else {
				if self.Context.Hierarchy.IsCompatible(nodeType, nodeTemplate.NodeType) {
					compatible = true
					break
				}
			}
		}
		if !compatible {
			childContext := self.Context.FieldChild("targets", nil).ListChild(index, nil)
			childContext.ReportIncompatible(nodeTemplate.Name, "policy", "target")
		}
	}
	for index, group := range self.TargetGroups {
		compatible := false
		for _, groupType := range self.PolicyType.TargetGroupTypes {
			if self.Context.ReadFromDb {
				if IsCompatibleGroup(groupType, group.GroupType) {
					compatible = true
					break
				}
			} else {
				if self.Context.Hierarchy.IsCompatible(groupType, group.GroupType) {
					compatible = true
					break
				}
			}
		}
		if !compatible {
			childContext := self.Context.FieldChild("targets", nil).ListChild(index, nil)
			childContext.ReportIncompatible(group.Name, "policy", "target")
		}
	}
}

func (self *Policy) Normalize(s *normal.ServiceTemplate) *normal.Policy {
	log.Infof("{normalize} policy: %s", self.Name)

	p := s.NewPolicy(self.Name)

	if self.Description != nil {
		p.Description = *self.Description
	}

	if self.GetContext().ReadFromDb {
		p.Types = determinePolicyHierarchyTypes(self)
	} else {
		if types, ok := normal.GetTypes(self.Context.Hierarchy, self.PolicyType); ok {
			p.Types = types
		}
	}

	self.Properties.Normalize(p.Properties)

	for _, nodeTemplate := range self.TargetNodeTemplates {
		if n, ok := s.NodeTemplates[nodeTemplate.Name]; ok {
			p.NodeTemplateTargets = append(p.NodeTemplateTargets, n)
		}
	}

	for _, group := range self.TargetGroups {
		if g, ok := s.Groups[group.Name]; ok {
			p.GroupTargets = append(p.GroupTargets, g)
		}
	}

	self.TriggerDefinitions.Normalize(p, s)

	return p
}

//
// Policies
//

type Policies []*Policy

func (self Policies) Normalize(s *normal.ServiceTemplate) {
	for _, policy := range self {
		s.Policies[policy.Name] = policy.Normalize(s)
	}
}

func determinePolicyHierarchyTypes(policy *Policy) normal.Types {

	ptypes := make(normal.Types)

	currType := policy.PolicyType
	for currType != nil {
		ptype := normal.NewType(currType.Name)
		ptype.Metadata = currType.Metadata
		ptypes[currType.Name] = ptype
		currType = currType.Parent
	}

	return ptypes

}

func IsCompatible(nt1 *NodeType, nt2 *NodeType) bool {
	compatible := false
	nt := nt2
	for nt != nt1 && nt != nil {
		nt = nt.Parent
	}
	compatible = (nt == nt1)
	if !compatible {
		nt = nt1
		for nt != nt2 && nt != nil {
			nt = nt.Parent
		}
		compatible = (nt == nt2)
	}
	return compatible
}
func IsCompatibleGroup(nt1 *GroupType, nt2 *GroupType) bool {
	compatible := false
	nt := nt2
	for nt != nt1 && nt != nil {
		nt = nt.Parent
	}
	compatible = (nt == nt1)
	if !compatible {
		nt = nt1
		for nt != nt2 && nt != nil {
			nt = nt.Parent
		}
		compatible = (nt == nt2)
	}
	return compatible
}
