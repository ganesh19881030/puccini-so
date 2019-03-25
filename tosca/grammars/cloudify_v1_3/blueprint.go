package cloudify_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// Blueprint
//
// [https://docs.cloudify.co/4.5.5/developer/blueprints/]
//

type Blueprint struct {
	*Unit `name:"blueprint"`

	Description *string `read:"description"` // not in spec, but in code
	Groups      Groups  `read:"groups,Group"`
}

func NewBlueprint(context *tosca.Context) *Blueprint {
	return &Blueprint{Unit: NewUnit(context)}
}

// tosca.Reader signature
func ReadBlueprint(context *tosca.Context) interface{} {
	self := NewBlueprint(context)
	context.ScriptNamespace.Merge(DefaultScriptNamespace)
	context.ValidateUnsupportedFields(append(context.ReadFields(self), "dsl_definitions"))
	return self
}

// parser.HasInputs interface
func (self *Blueprint) SetInputs(inputs map[string]interface{}) {
	context := self.Context.FieldChild("inputs", nil)
	for name, data := range inputs {
		childContext := context.MapChild(name, data)
		input, ok := self.Inputs[name]
		if !ok {
			childContext.ReportUndefined("input")
			continue
		}

		input.Value = ReadValue(childContext).(*Value)
	}
}

// tosca.Normalizable interface
func (self *Blueprint) Normalize() *normal.ServiceTemplate {
	log.Info("{normalize} blueprint")

	s := normal.NewServiceTemplate()

	if self.Description != nil {
		s.Description = *self.Description
	}

	s.ScriptNamespace = self.Context.ScriptNamespace

	self.Inputs.Normalize(s.Inputs, self.Context.FieldChild("inputs", nil))
	self.Outputs.Normalize(s.Outputs)

	for _, nodeTemplate := range self.NodeTemplates {
		s.NodeTemplates[nodeTemplate.Name] = nodeTemplate.Normalize(s)
	}

	for _, nodeTemplate := range self.NodeTemplates {
		nodeTemplate.NormalizeRelationships(s)
	}

	for _, workflow := range self.Workflows {
		s.Workflows[workflow.Name] = workflow.Normalize(s)
	}

	for _, policy := range self.Policies {
		s.Policies[policy.Name] = policy.Normalize(s)
	}

	return s
}
