package tosca_v1_3

import (
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// ActivityDefinition
//
// [TOSCA-Simple-Profile-YAML-v1.3] @ 3.6.23
// [TOSCA-Simple-Profile-YAML-v1.2] @ 3.6.19
// [TOSCA-Simple-Profile-YAML-v1.1] @ 3.5.17
//

type ActivityDefinition struct {
	*Entity `name:"workflow activity definition"`

	DelegateWorkflowDefinitionName *string
	InlineWorkflowDefinitionName   *string
	SetNodeState                   *string
	CallOperationSpec              *string
	Update                         ParameterDefinitions `read:"update,ParameterDefinition"`

	DelegateWorkflowDefinition *WorkflowDefinition  `lookup:"delegate,DelegateWorkflowDefinitionName,WorkflowDefinition" json:"-" yaml:"-"`
	InlineWorkflowDefinition   *WorkflowDefinition  `lookup:"inline,InlineWorkflowDefinitionName,WorkflowDefinition" json:"-" yaml:"-"`
	CallInterface              *InterfaceAssignment `json:"-" yaml:"-"`
	CallOperation              *OperationAssignment `json:"-" yaml:"-"`
}

func NewActivityDefinition(context *tosca.Context) *ActivityDefinition {
	return &ActivityDefinition{
		Entity: NewEntity(context),
		Update: make(ParameterDefinitions),
	}
}

// tosca.Reader signature
func ReadActivityDefinition(context *tosca.Context) interface{} {
	self := NewActivityDefinition(context)

	if context.ValidateType("map") {
		map_ := context.Data.(ard.Map)
		if (!context.ReadFromDb && len(map_) != 1) || (context.ReadFromDb && len(map_) != 2) {
			context.ReportValueMalformed("workflow activity definition", "map length not 1")
			return self
		}

		for operator, value := range map_ {
			if operator == "uid" {
				continue
			}
			childContext := context.FieldChild(operator, value)

			switch operator {
			case "delegate":
				self.DelegateWorkflowDefinitionName = childContext.ReadString()
			case "inline":
				self.InlineWorkflowDefinitionName = childContext.ReadString()
			case "set_state":
				self.SetNodeState = childContext.ReadString()
			case "call_operation":
				self.CallOperationSpec = childContext.ReadString()
			case "update":
				context.ValidateUnsupportedFields(context.ReadFields(self))
			default:
				context.ReportValueMalformed("workflow activity definition", "unsupported operator")
				return self
			}

			// We have only one key
			break
		}
	}

	return self
}

func (self *ActivityDefinition) Render(stepDefinition *WorkflowStepDefinition) {
	if self.CallOperationSpec == nil {
		return
	}

	// Parse operation spec
	s := strings.SplitN(*self.CallOperationSpec, ".", 2)
	if len(s) != 2 {
		self.Context.FieldChild("call_operation", *self.CallOperationSpec).ReportValueWrongFormat("interface.operation")
		return
	}

	var ok bool

	// Lookup interface by name
	if stepDefinition.TargetNodeTemplate != nil {
		if self.CallInterface, ok = stepDefinition.TargetNodeTemplate.Interfaces[s[0]]; !ok {
			self.Context.FieldChild("call_operation", s[0]).ReportReferenceNotFound("interface", stepDefinition.TargetNodeTemplate)
			return
		}
	} else if stepDefinition.TargetGroup != nil {
		if self.CallInterface, ok = stepDefinition.TargetGroup.Interfaces[s[0]]; !ok {
			self.Context.FieldChild("call_operation", s[0]).ReportReferenceNotFound("interface", stepDefinition.TargetGroup)
			return
		}
	} else {
		// There was a lookup problem (neither node template nor group)
		return
	}

	// Lookup operation by name
	if self.CallOperation, ok = self.CallInterface.Operations[s[1]]; !ok {
		self.Context.FieldChild("call_operation", s[1]).ReportReferenceNotFound("operation", self.CallInterface)
	}
}

func (self *ActivityDefinition) Normalize(st *normal.WorkflowStep, s *normal.ServiceTemplate) *normal.WorkflowActivity {
	log.Info("{normalize} workflow activity")

	a := st.NewActivity()
	if self.DelegateWorkflowDefinition != nil {
		a.DelegateWorkflow = s.Workflows[self.DelegateWorkflowDefinition.Name]
	} else if self.InlineWorkflowDefinition != nil {
		a.InlineWorkflow = s.Workflows[self.InlineWorkflowDefinition.Name]
	} else if self.SetNodeState != nil {
		a.SetNodeState = *self.SetNodeState
	} else if self.CallOperation != nil {
		var i *normal.Interface
		if st.TargetNodeTemplate != nil {
			i = st.TargetNodeTemplate.Interfaces[self.CallInterface.Name]
		} else if st.TargetGroup != nil {
			i = st.TargetGroup.Interfaces[self.CallInterface.Name]
		} else {
			return a
		}
		a.CallOperation = i.Operations[self.CallOperation.Name]
	}

	return a
}

//
// ActivityDefinitions
//

type ActivityDefinitions []*ActivityDefinition

func (self ActivityDefinitions) Normalize(action *normal.WorkflowActivity) {
	for _, definition := range self {
		definition.NormalizeUpdate(action.Update)
	}
}

func (self *ActivityDefinition) NormalizeUpdate(constraints normal.Constrainables) {
	self.Update.Normalize(constraints, self.Context)
}

func (self ActivityDefinitions) Render() {
	for _, definition := range self {
		definition.RenderUpdate()
	}
}

func (self *ActivityDefinition) RenderUpdate() {
	if self.Update != nil {
		self.Update.Render("update", self.Context)
	}
}
