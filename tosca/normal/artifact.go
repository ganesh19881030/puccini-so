package normal

//
// Artifact
//

type Artifact struct {
	NodeTemplate *NodeTemplate  `json:"-" yaml:"-"`
	Name         string         `json:"-" yaml:"-"`
	Description  string         `json:"description" yaml:"description"`
	Types        Types          `json:"types" yaml:"types"`
	Properties   Constrainables `json:"properties" yaml:"properties"`
	SourcePath   string         `json:"sourcePath" yaml:"sourcePath"`
	TargetPath   string         `json:"targetPath" yaml:"targetPath"`
}

func (self *NodeTemplate) NewArtifact(name string) *Artifact {
	artifact := &Artifact{
		NodeTemplate: self,
		Name:         name,
		Types:        make(Types),
		Properties:   make(Constrainables),
	}
	self.Artifacts[name] = artifact
	return artifact
}

func (self *Operation) NewArtifact(name string) *Artifact {
	artifact := &Artifact{
		Name:       name,
		Types:      make(Types),
		Properties: make(Constrainables),
	}
	self.Dependencies = append(self.Dependencies, artifact)

	return artifact
}

func (self *Notification) NewNotificationArtifact(name string) *Artifact {
	artifact := &Artifact{
		Name:       name,
		Types:      make(Types),
		Properties: make(Constrainables),
	}
	self.Dependencies = append(self.Dependencies, artifact)

	return artifact
}

//
// Artifacts
//

type Artifacts map[string]*Artifact
type ArtifactList []*Artifact
