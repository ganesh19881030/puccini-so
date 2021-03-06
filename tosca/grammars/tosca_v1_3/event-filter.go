package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
)

//
// EventFilter
//
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.17
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.15
//

type EventFilter struct {
	*Entity `name:"event filter" json:"-" yaml:"-"`

	NodeTemplateNameOrTypeName *string `read:"node"`
	RequirementName            *string `read:"requirement"`
	CapabilityName             *string `read:"capability"`

	NodeTemplate *NodeTemplate `lookup:"node,NodeTemplateNameOrTypeName,NodeTemplate,no" json:"-" yaml:"-"`
	NodeType     *NodeType     `lookup:"node,NodeTemplateNameOrTypeName,NodeType,no" json:"-" yaml:"-"`
}

func NewEventFilter(context *tosca.Context) *EventFilter {
	return &EventFilter{Entity: NewEntity(context)}
}

// tosca.Reader signature
func ReadEventFilter(context *tosca.Context) interface{} {
	self := NewEventFilter(context)
	context.ReadFields(self)
	return self
}
