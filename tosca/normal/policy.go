package normal

//
// Policy
//

type Policy struct {
	ServiceTemplate     *ServiceTemplate `json:"-" yaml:"-"`
	Name                string           `json:"-" yaml:"-"`
	Description         string           `json:"description" yaml:"description"`
	Types               Types            `json:"types" yaml:"types"`
	Properties          Constrainables   `json:"properties" yaml:"properties"`
	GroupTargets        []*Group         `json:"-" yaml:"-"`
	NodeTemplateTargets []*NodeTemplate  `json:"-" yaml:"-"`
	Triggers            []*PolicyTrigger `json:"-" yaml:"-"`
}

func (self *ServiceTemplate) NewPolicy(name string) *Policy {
	policy := &Policy{
		ServiceTemplate:     self,
		Name:                name,
		Types:               make(Types),
		Properties:          make(Constrainables),
		GroupTargets:        make([]*Group, 0),
		NodeTemplateTargets: make([]*NodeTemplate, 0),
	}
	self.Policies[name] = policy
	return policy
}

//
// Policies
//

type Policies map[string]*Policy

//
// PolicyTrigger
//

type PolicyTrigger struct {
	Policy      *Policy           `json:"-" yaml:"-"`
	Name        string            `json:"-" yaml:"-"`
	Description string            `json:"description" yaml:"description"`
	EventType   string            `json:"eventType" yaml:"eventType"`
	Operation   *Operation        `json:"operation" yaml:"operation"`
	Workflow    *Workflow         `json:"workflow" yaml:"workflow"`
	Condition   *Condition        `json:"condition" yaml:"condition"`
	Action      *WorkflowActivity `json:"action" yaml:"action"`
	// TODO: missing fields
}

func (self *Policy) NewTrigger(name string) *PolicyTrigger {
	trigger := &PolicyTrigger{
		Name:   name,
		Policy: self,
	}
	self.Triggers = append(self.Triggers, trigger)
	return trigger
}

//
// Condition
//

type Condition struct {
	ConditionClauseConstraints FunctionCallMap `json:"-" yaml:"-"`
}

func (self *PolicyTrigger) NewCondition() *Condition {
	condition := &Condition{
		ConditionClauseConstraints: make(FunctionCallMap),
	}
	self.Condition = condition
	return condition
}
