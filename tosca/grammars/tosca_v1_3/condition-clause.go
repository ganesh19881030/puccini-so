package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// ConditionClause
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.25
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.21
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.19
//

type ConditionClause struct {
	*Entity `name:"condition clause"`

	DirectAssertionDefinition *DirectAssertionDefinition
}

func NewConditionClause(context *tosca.Context) *ConditionClause {
	return &ConditionClause{Entity: NewEntity(context)}
}

// tosca.Reader signature
func ReadConditionClause(context *tosca.Context) interface{} {
	self := NewConditionClause(context)

	if context.ValidateType("map") {
		for _, childContext := range context.FieldChildren() {
			if self.readField(childContext) {
				return self
			}
		}
		self.DirectAssertionDefinition = ReadDirectAssertionDefinition(context).(*DirectAssertionDefinition)
	}

	return self
}

func (self *ConditionClause) readField(context *tosca.Context) bool {
	switch context.Name {
	case "and":
	case "or":
	case "assert":
	default:
		return false
	}
	return true
}

//
// ConditionClauses
//

type ConditionClauses []*ConditionClause

func (self *ConditionClause) Normalize(functionCallMap normal.FunctionCallMap) {
	if self.DirectAssertionDefinition != nil {
		self.DirectAssertionDefinition.Normalize(functionCallMap)
	}
}

func (self ConditionClauses) Normalize(condition *normal.Condition) {
	for _, ConditionClause := range self {
		ConditionClause.Normalize(condition.ConditionClauseConstraints)
	}
}
