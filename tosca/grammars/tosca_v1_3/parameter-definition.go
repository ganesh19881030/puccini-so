package tosca_v1_3

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// ParameterDefinition
//
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.14
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.13
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.12
//

type ParameterDefinition struct {
	*PropertyDefinition `name:"parameter definition"`

	Value *Value `read:"value,Value"`
}

func NewParameterDefinition(context *tosca.Context) *ParameterDefinition {
	return &ParameterDefinition{PropertyDefinition: NewPropertyDefinition(context)}
}

// tosca.Reader signature
func ReadParameterDefinition(context *tosca.Context) interface{} {
	self := NewParameterDefinition(context)

	if isShortNotation(context) {
		// Short notation
		self.Value = ReadValue(context.FieldChild("value", context.Data)).(*Value)
	} else {
		// Extended notation
		context.ValidateUnsupportedFields(context.ReadFields(self))
	}
	return self
}

// function to determine if the type of notation being used is short notation
// it checks for presence of one or more extended notation keys
func isShortNotation(context *tosca.Context) bool {

	parameterDefinitionKeys := [...]string{"type", "description", "value", "required", "default", "status",
		"constraints", "key_schema ", "entry_schema"}

	contextDataMap, _ := context.Data.(ard.Map)

	if context.ReadFromDb {
		if _, ok := contextDataMap["constraintclauses"]; ok {
			return false
		}
	}

	for key := range contextDataMap {
		for _, paramDefKey := range parameterDefinitionKeys {
			if key == paramDefKey {
				return false
			}
		}
	}
	return true
}

func (self *ParameterDefinition) Render(kind string) {
	// TODO: what to do if there is no "type"?

	if self.Value == nil {
		self.Value = self.Default
	}

	if self.Value == nil {
		// PropertyDefinition.Required defaults to true
		required := (self.Required == nil) || *self.Required
		if required {
			self.Context.ReportPropertyRequired(kind)
			return
		}
	} else if self.DataType != nil {
		self.Value.RenderProperty(self.DataType, self.PropertyDefinition)
	}
}

func (self *ParameterDefinition) Normalize(context *tosca.Context) normal.Constrainable {
	var value *Value
	if self.Value != nil {
		value = self.Value
	} else {
		// Parameters should always appear, even if they have no default value
		value = NewValue(context.MapChild(self.Name, nil))
	}
	return value.Normalize()
}

//
// ParameterDefinitions
//

type ParameterDefinitions map[string]*ParameterDefinition

func (self ParameterDefinitions) Render(kind string, context *tosca.Context) {
	for _, definition := range self {
		definition.Render(kind)
	}
}

func (self ParameterDefinitions) Normalize(c normal.Constrainables, context *tosca.Context) {
	for key, definition := range self {
		c[key] = definition.Normalize(context)
	}
}

func (self *ParameterDefinition) Inherit(parentDefinition *ParameterDefinition) {

	// type is not required in ParameterDefinition
	// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.14
	if self.DataTypeName == nil {
		self.typeMissingProblemReported = true
	}

	if parentDefinition != nil {
		self.PropertyDefinition.Inherit(parentDefinition.PropertyDefinition)

		if (self.Value == nil) && (parentDefinition.Value != nil) {
			self.Value = parentDefinition.Value
		}

	} else {
		self.PropertyDefinition.Inherit(nil)

	}
}

func (self ParameterDefinitions) Inherit(parentDefinitions ParameterDefinitions) {
	for name, definition := range parentDefinitions {
		if _, ok := self[name]; !ok {
			self[name] = definition
		}
	}

	for name, definition := range self {
		if parentDefinitions != nil {
			if parentDefinition, ok := parentDefinitions[name]; ok {
				if definition != parentDefinition {
					definition.Inherit(parentDefinition)
				}
				continue
			}
		}

		definition.Inherit(nil)
	}
}
