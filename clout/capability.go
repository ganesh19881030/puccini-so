package clout

import (
	"github.com/tliron/puccini/ard"
)

//
// Vertex
//

type Capability struct {
	//Key                 string  `json:"-" yaml:"-"`
	Attributes           ard.Map `json:"attributes" yaml:"attributes"`
	Description          string  `json:"description" yaml:"description"`
	MaxRelationshipCount float64 `json:"maxRelationshipCount" yaml:"maxRelationshipCount"`
	MinRelationshipCount float64 `json:"minRelationshipCount" yaml:"minRelationshipCount"`
	Properties           ard.Map `json:"properties" yaml:"properties"`
	Types                ard.Map `json:"types" yaml:"types"`
}

func NewCapability() *Capability {
	cap := &Capability{
		//Key:                  name,
		Attributes:           make(ard.Map),
		Description:          "",
		MaxRelationshipCount: 0,
		MinRelationshipCount: 0,
		Properties:           make(ard.Map),
		Types:                make(ard.Map),
	}
	//capability[name] = cap
	return cap
}
