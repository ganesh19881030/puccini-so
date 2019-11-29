package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// TriggerDefinitionCondition
//

type TriggerDefinitionCondition struct {
	*Entity `name:"trigger definition condition" json:"-" yaml:"-"`

	ConditionClauses ConditionClauses `read:"constraint,[]ConditionClause"`
	Period           *ScalarUnitTime  `read:"period,scalar-unit.time"`
	Evaluations      *int             `read:"evaluations"`
	Method           *string          `read:"method"`
}

func NewTriggerDefinitionCondition(context *tosca.Context) *TriggerDefinitionCondition {
	return &TriggerDefinitionCondition{Entity: NewEntity(context)}
}

// tosca.Reader signature
func ReadTriggerDefinitionCondition(context *tosca.Context) interface{} {
	self := NewTriggerDefinitionCondition(context)

	if context.Is("list") {
		// short notation

		context.ReadListItems(ReadConditionClause, func(item interface{}) {
			self.ConditionClauses = append(self.ConditionClauses, item.(*ConditionClause))
		})
	} else if context.Is("map") {
		context.ValidateUnsupportedFields(context.ReadFields(self))
	}
	return self
}

func (self *TriggerDefinitionCondition) Normalize(o *normal.Condition) {
	if self.ConditionClauses != nil {
		self.ConditionClauses.Normalize(o)
	}
}
