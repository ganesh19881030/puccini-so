package tosca_v1_3

import (
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// DirectAssertionDefinition
//
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.25.3
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.21.3
//

type DirectAssertionDefinition struct {
	*Entity `name:"direct assertion definition"`

	Name             *string
	ConstraintClause ConstraintClauses
}

func NewDirectAssertionDefinition(context *tosca.Context) *DirectAssertionDefinition {
	return &DirectAssertionDefinition{Entity: NewEntity(context)}
}

// tosca.Reader signature
func ReadDirectAssertionDefinition(context *tosca.Context) interface{} {
	self := NewDirectAssertionDefinition(context)
	if context.ValidateType("map") {
		contextData := context.Data.(ard.Map)

		for name, data := range contextData {
			self.Name = &name
			context.Data = data
			context.ReadListItems(ReadConstraintClause, func(item interface{}) {
				self.ConstraintClause = append(self.ConstraintClause, item.(*ConstraintClause))
			})
		}
	}
	return self
}

func (self *DirectAssertionDefinition) Normalize(functionCallMap normal.FunctionCallMap) normal.FunctionCalls {
	functionCalls := self.ConstraintClause.Normalize(self.Context)
	functionCallMap[*self.Name] = functionCalls
	return functionCalls
}
