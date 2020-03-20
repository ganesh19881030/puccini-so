package normal

import (
	"encoding/json"
)

//
// AttributeMapping
//

type AttributeMapping struct {
	NodeTemplate   *NodeTemplate
	CapabilityName string
	AttributeName  string
}

func (self *NodeTemplate) NewAttributeMapping(attributeName string, CapabilityName string) *AttributeMapping {
	return &AttributeMapping{
		NodeTemplate:   self,
		CapabilityName: CapabilityName,
		AttributeName:  attributeName,
	}
}

type MarshalableAttributeMapping struct {
	NodeTemplateName string `json:"nodeTemplateName" yaml:"nodeTemplateName"`
	CapabilityName   string `json:"capabilityName" yaml:"capabilityName"`
	AttributeName    string `json:"attributeName" yaml:"attributeName"`
}

func (self *AttributeMapping) Marshalable() interface{} {
	return &MarshalableAttributeMapping{
		NodeTemplateName: self.NodeTemplate.Name,
		CapabilityName:   self.CapabilityName,
		AttributeName:    self.AttributeName,
	}
}

// json.Marshaler interface
func (self *AttributeMapping) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.Marshalable())
}

// yaml.Marshaler interface
func (self *AttributeMapping) MarshalYAML() (interface{}, error) {
	return self.Marshalable(), nil
}

//
// AttributeMappings
//

type AttributeMappings map[string]*AttributeMapping
