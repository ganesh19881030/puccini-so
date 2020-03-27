package tosca_v1_3

import (
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/normal"
)

//
// AttributeMapping
//
// Attaches to NotificationDefinition
//

type AttributeMapping struct {
	*Entity `name:"attribute mapping"`
	Name    string

	NodeTemplateName *string `require:"0"`
	CapabilityName   *string
	AttributeName    *string `require:"1"`
}

func NewAttributeMapping(context *tosca.Context) *AttributeMapping {
	return &AttributeMapping{
		Entity: NewEntity(context),
		Name:   context.Name,
	}
}

// tosca.Reader signature
func ReadAttributeMapping(context *tosca.Context) interface{} {
	self := NewAttributeMapping(context)

	stringList := context.ReadStringList()
	if len(*stringList) == 2 {
		self.NodeTemplateName = &(*stringList)[0]
		self.AttributeName = &(*stringList)[1]
	} else if len(*stringList) == 3 {
		self.NodeTemplateName = &(*stringList)[0]
		self.CapabilityName = &(*stringList)[1]
		self.AttributeName = &(*stringList)[2]
	}

	return self
}

// tosca.Mappable interface
func (self *AttributeMapping) GetKey() string {
	return self.Name
}

//
// AttributeMappings
//

type AttributeMappings map[string]*AttributeMapping

func (self AttributeMappings) Inherit(parent AttributeMappings) {
	for name, attributeMapping := range parent {
		if _, ok := self[name]; !ok {
			self[name] = attributeMapping
		}
	}
}

func (self AttributeMappings) Normalize(n *normal.NodeTemplate, m normal.AttributeMappings) {
	for name, attributeMapping := range self {
		nodeTemplateName := *attributeMapping.NodeTemplateName
		CapabilityName := attributeMapping.CapabilityName

		if CapabilityName == nil {
			CapabilityName = new(string)
		} else if _, ok := n.Capabilities[*CapabilityName]; !ok {
			attributeMapping.Context.ReportPathf("unknown capability reference in nodeTemplate \"%s\": %s", n.Name, *CapabilityName)
		}

		if nodeTemplateName == "SELF" {
			m[name] = n.NewAttributeMapping(*attributeMapping.AttributeName, *CapabilityName)
		} else {
			if nn, ok := n.ServiceTemplate.NodeTemplates[nodeTemplateName]; ok {
				m[name] = nn.NewAttributeMapping(*attributeMapping.AttributeName, *CapabilityName)
			}
		}
	}
}
