package normal

//
// Interface
//

type Interface struct {
	NodeTemplate  *NodeTemplate  `json:"-" yaml:"-"`
	Group         *Group         `json:"-" yaml:"-"`
	Relationship  *Relationship  `json:"-" yaml:"-"`
	Name          string         `json:"-" yaml:"-"`
	Description   string         `json:"description" yaml:"description"`
	Types         Types          `json:"types" yaml:"types"`
	Inputs        Constrainables `json:"inputs" yaml:"inputs"`
	Operations    Operations     `json:"operations" yaml:"operations"`
	Notifications Notifications  `json:"notifications" yaml:"notifications"`
}

func (self *NodeTemplate) NewInterface(name string) *Interface {
	intr := &Interface{
		NodeTemplate:  self,
		Name:          name,
		Types:         make(Types),
		Inputs:        make(Constrainables),
		Operations:    make(Operations),
		Notifications: make(Notifications),
	}
	self.Interfaces[name] = intr
	return intr
}

func (self *Group) NewInterface(name string) *Interface {
	intr := &Interface{
		Group:         self,
		Name:          name,
		Types:         make(Types),
		Inputs:        make(Constrainables),
		Operations:    make(Operations),
		Notifications: make(Notifications),
	}
	self.Interfaces[name] = intr
	return intr
}

func (self *Relationship) NewInterface(name string) *Interface {
	intr := &Interface{
		Relationship:  self,
		Name:          name,
		Types:         make(Types),
		Inputs:        make(Constrainables),
		Operations:    make(Operations),
		Notifications: make(Notifications),
	}
	self.Interfaces[name] = intr
	return intr
}

//
// Interfaces
//

type Interfaces map[string]*Interface

//
// Operation
//

type Operation struct {
	Interface      *Interface        `json:"-" yaml:"-"`
	PolicyTrigger  *PolicyTrigger    `json:"-" yaml:"-"`
	Name           string            `json:"-" yaml:"-"`
	Description    string            `json:"description" yaml:"description"`
	Implementation string            `json:"implementation" yaml:"implementation"`
	Dependencies   ArtifactList      `json:"dependencies" yaml:"dependencies"`
	Inputs         Constrainables    `json:"inputs" yaml:"inputs"`
	Timeout        int64             `json:"timeout" yaml:"timeout"`
	Host           string            `json:"host" yaml:"host"`
	Outputs        AttributeMappings `json:"outputs" yaml:"outputs"`
}

func (self *Interface) NewOperation(name string) *Operation {
	operation := &Operation{
		Interface: self,
		Name:      name,
		Inputs:    make(Constrainables),
		Outputs:   make(AttributeMappings),
		Timeout:   -1,
	}
	self.Operations[name] = operation
	return operation
}

func (self *PolicyTrigger) NewOperation() *Operation {
	self.Operation = &Operation{
		PolicyTrigger: self,
		Inputs:        make(Constrainables),
	}
	return self.Operation
}

//
// Operations
//

type Operations map[string]*Operation
