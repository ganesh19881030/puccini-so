package database

import (
	"encoding/json"
	"reflect"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/normal"
)

const VERSION = "1.0"
const toscaPrefix string = "tosca"

// DB3 defines an implementation of CloutDB
type DB3 struct {
	Dburl string
	clout bool
}

// NewDb3 creates a DB3 instance
func NewDb3(dburl string) CloutDB {
	return DB3{dburl, false}
}

// IsCloutCapable - true if it handles clout structure, false otherwise
func (db DB3) IsCloutCapable() bool {
	return db.clout
}

// SaveClout method implementation of CloutDB interface for CloutDB2 instance
func (db DB3) SaveClout(clout *clout.Clout, urlString string, grammarVersions string, internalImport string) error {
	return nil
}

// SaveServiceTemplate method implementation of CloutDB interface for CloutDB1 instance
func (db DB3) SaveServiceTemplate(s *normal.ServiceTemplate, urlString string, grammarVersions string, internalImport string) error {
	return nil
}

func processConstrainables(io *normal.Constrainables, iotype string) ([]ard.Map, error) {
	var toscaInputs []ard.Map
	var err error

	for key, input := range *io {
		toscaInput := make(ard.Map)
		toscaInput[addToscaPrefix("name")] = key
		toscaInput[addToscaPrefix("type")] = iotype
		v1 := reflect.ValueOf(input)
		v2 := v1.Elem()
		knd := v2.Kind()
		typ := v2.Type()
		Log.Debugf("kind = %s %s", knd.String(), typ.String())
		if typ.String() == "normal.Value" {
			val := input.(*normal.Value)
			if val != nil {
				toscaInput[addToscaPrefix("value")] = val.Value
				toscaInput[addToscaPrefix("description")] = val.Description
				toscaInput[addToscaPrefix("constraintype")] = typ.String()
			}
		} else if typ.String() == "normal.Function" {
			val := input.(*normal.FunctionCall)
			if val != nil {
				toscaInput[addToscaPrefix("description")] = val.Description
				toscaInput[addToscaPrefix("constraintype")] = typ.String()
				err = stringify(&toscaInput, addToscaPrefix("function"), val.FunctionCall)
				common.FailOnError(err)
				err = stringify(&toscaInput, addToscaPrefix("constraints"), val.Constraints)
				common.FailOnError(err)
			}

		} else if typ.String() == "normal.ConstrainableList" {
			val := input.(*normal.ConstrainableList)
			if val != nil {
				toscaInput[addToscaPrefix("description")] = val.Description
				toscaInput[addToscaPrefix("constraintype")] = typ.String()
				err = stringify(&toscaInput, addToscaPrefix("list"), val.List)
				common.FailOnError(err)
				err = stringify(&toscaInput, addToscaPrefix("constraints"), val.Constraints)
				common.FailOnError(err)
			}

		} else {
			Log.Debug("*** Unhandled constraint type = %s\n", typ.String())
		}
		toscaInputs = append(toscaInputs, toscaInput)
	}

	return toscaInputs, nil
}

func processNodeTemplates(nts *normal.NodeTemplates) ([]ard.Map, error) {
	var nodeTemplates []ard.Map

	for key, nodeTemplate := range *nts {
		toscaNodeTemplate := make(ard.Map)
		toscaNodeTemplate[addToscaPrefix("name")] = key
		toscaNodeTemplate[addToscaPrefix("type")] = "nodeTemplate"
		toscaNodeTemplate[addToscaPrefix("description")] = nodeTemplate.Description

		// types
		ntypes, err := processTypes(&nodeTemplate.Types, "ntype")
		if err != nil && ntypes != nil {
			toscaNodeTemplate[addToscaPrefix("ntypes")] = ntypes
		}

		//processDirectives()
		//processInterfaces()
		// properties
		props, err := processConstrainables(&nodeTemplate.Properties, "property")
		if err == nil && props != nil {
			toscaNodeTemplate[addToscaPrefix("properties")] = props
		}
		// attributes
		attribs, err := processConstrainables(&nodeTemplate.Attributes, "attribute")
		if err == nil && attribs != nil {
			toscaNodeTemplate[addToscaPrefix("attributes")] = attribs
		}
		// requirements
		requirements, err := processRequirements(&nodeTemplate.Requirements)
		if err == nil && requirements != nil {
			toscaNodeTemplate[addToscaPrefix("requirements")] = requirements
		}
		// capabilities
		capabilities, err := processCapabilities(&nodeTemplate.Capabilities)
		if err == nil && capabilities != nil {
			toscaNodeTemplate[addToscaPrefix("capabilities")] = capabilities
		}
		//processArtifacts()

		nodeTemplates = append(nodeTemplates, toscaNodeTemplate)

	}

	return nodeTemplates, nil

}

func processTypes(ntypes *normal.Types, ftype string) (ard.Map, error) {
	var toscaNTypeParent *ard.Map
	var toscaNTypeRoot *ard.Map

	for key, ntype := range *ntypes {
		toscaNType := make(ard.Map)
		toscaNType[addToscaPrefix("type")] = ftype
		toscaNType[addToscaPrefix("name")] = key
		toscaNType[addToscaPrefix("metadata")] = ntype.Metadata

		if toscaNTypeParent == nil {
			toscaNTypeParent = &toscaNType
		} else {
			(*toscaNTypeParent)["parent"] = toscaNType
			toscaNTypeParent = &toscaNType

		}
		if toscaNTypeRoot == nil {
			toscaNTypeRoot = toscaNTypeParent
		}
	}

	return *toscaNTypeRoot, nil
}

func processRequirements(reqs *normal.Requirements) ([]ard.Map, error) {
	var toscaReqs []ard.Map

	for key, req := range *reqs {
		toscaReq := make(ard.Map)
		toscaReq[addToscaPrefix("type")] = "ntype"
		toscaReq[addToscaPrefix("name")] = key
		toscaReq[addToscaPrefix("capabilityName")] = req.CapabilityName
		toscaReq[addToscaPrefix("capabilityTypeName")] = req.CapabilityTypeName
		toscaReq[addToscaPrefix("nodeTypeName")] = req.NodeTypeName
		//toscaReq[addToscaPrefix("nodeTemplate")] = req.NodeTemplate
		//req.NodeTemplatePropertyConstraints
		//req.CapabilityPropertyConstraints

		toscaReqs = append(toscaReqs, toscaReq)
	}

	return toscaReqs, nil
}

func processCapabilities(caps *normal.Capabilities) ([]ard.Map, error) {
	var toscaCaps []ard.Map

	for key, cap := range *caps {
		toscaCap := make(ard.Map)
		toscaCap[addToscaPrefix("type")] = "capability"
		toscaCap[addToscaPrefix("name")] = key
		toscaCap[addToscaPrefix("description")] = cap.Description
		toscaCap[addToscaPrefix("minRelationshipCount")] = cap.MinRelationshipCount
		toscaCap[addToscaPrefix("maxRelationshipCount")] = cap.MaxRelationshipCount

		// types
		ctypes, err := processTypes(&cap.Types, "ctype")
		if err != nil && ctypes != nil {
			toscaCap[addToscaPrefix("ctypes")] = ctypes
		}
		// properties
		props, err := processConstrainables(&cap.Properties, "property")
		if err == nil && props != nil {
			toscaCap[addToscaPrefix("properties")] = props
		}
		// attributes
		attribs, err := processConstrainables(&cap.Attributes, "attribute")
		if err == nil && attribs != nil {
			toscaCap[addToscaPrefix("attributes")] = attribs
		}

		toscaCaps = append(toscaCaps, toscaCap)
	}

	return toscaCaps, nil
}

func processGroups(ntypes *normal.Types) ([]ard.Map, error) {
	var toscaNTypes []ard.Map

	for key, ntype := range *ntypes {
		toscaNType := make(ard.Map)
		toscaNType[addToscaPrefix("type")] = "ntype"
		toscaNType[addToscaPrefix("name")] = key
		toscaNType[addToscaPrefix("metadata")] = ntype.Metadata

		toscaNTypes = append(toscaNTypes, toscaNType)
	}

	return toscaNTypes, nil
}

// SetMetadata sets metadata
func SetMetadata(entity clout.Entity, kind string) {
	metadata := make(ard.Map)
	metadata["version"] = VERSION
	metadata["kind"] = kind
	entity.GetMetadata()["puccini-tosca"] = metadata
}

func addToscaPrefix(orig string) string {
	return addPrefix(toscaPrefix, orig)
}
func addPrefix(prefix string, orig string) string {
	return prefix + ":" + orig
}
func stringify(omap *ard.Map, key string, obj interface{}) error {
	bytes, error := json.Marshal(obj)
	if error == nil {
		(*omap)[key] = string(bytes)
	}

	return error
}
