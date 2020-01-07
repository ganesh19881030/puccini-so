package database

import (
	"fmt"
	"reflect"
	"strings"

	"encoding/json"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/tosca/normal"
)

// CloutDB1 defines an implementation of CloutDB
type CloutDB1 struct {
	Dburl string
	clout bool
}

// NewCloutDb1 creates a CloutDB1 instance
func NewCloutDb1(dburl string) CloutDB {
	return CloutDB1{dburl, true}
}

// SaveServiceTemplate method implementation of CloutDB interface for CloutDB1 instance
func (db CloutDB1) SaveServiceTemplate(s *normal.ServiceTemplate, urlString string, grammarVersion string, internalImport string) error {
	return nil
}

// IsCloutCapable - true if it handles clout structure, false otherwise
func (db CloutDB1) IsCloutCapable() bool {
	return db.clout
}

// SaveClout method implementation of CloutDB interface for CloutDB1 instance
//
// It is essentially a translation of graph.js plugin functionality to GO
// with a few tweaks
func (db CloutDB1) SaveClout(clout *clout.Clout, urlString string, grammarVersions string, internalImport string) error {
	var printout = true
	//	timestamp, err := common.Timestamp()
	//	if err != nil {
	//		return  err
	//	}
	//	var metadata1 interface{} = clout_.Metadata["puccini-js"]
	//	metadata2 := clout_.Metadata["puccini-tosca"]
	//
	//	tosca := clout_.Properties["tosca"]

	var dgraphset = DgraphSet{}

	var vertexItems []ard.Map
	var cloutItem = make(ard.Map)

	//	nodeTemplates := make(map[string]*clout.Vertex)
	//jscripts := clout.Metadata["puccini-js"].(ard.Map)
	/*
		for key, script := range jscripts {

			fmt.Println("key: ", key)
			fmt.Println("script: ", script)

		}
	*/
	//cloutItem["scripts"] = jscripts
	/*
		data := clout.Metadata["puccini-tosca"].(ard.Map)
		for key, value := range data {

			fmt.Println("key: ", key)
			fmt.Println("value: ", value)

		}

		data = clout.Properties["tosca"].(ard.Map)
		mdata := data["metadata"].(ard.Map)
		for key, value := range mdata {

			fmt.Println("key: ", key)
			fmt.Println("value: ", value)

		}
	*/
	// Node templates
	for _, vertex := range clout.Vertexes {
		//		v := clout_.NewVertex(clout.NewKey())

		ind := vertex.ID
		vxItem := make(ard.Map)
		vxItem["uid"] = "_:clout.vertex." + ind
		vxItem["tosca:vertexId"] = ind
		vxItem["clout:edge"] = make([]*ard.Map, 0)
		//vertexItems = append(vertexItems, vxItem)

		if isToscaVertex(vertex, "nodeTemplate") {
			fillNodeTemplate(&vxItem, &vertex.Properties)
		} else if isToscaVertex(vertex, "group") {
			//fillTosca(&vxItem, &vertex.Properties, "group", "")
			fillGroup(&vxItem, &vertex.Properties, "group", "")
		} else if isToscaVertex(vertex, "workflow") {
			fillTosca(&vxItem, &vertex.Properties, "workflow", "")
		} else if isToscaVertex(vertex, "workflowStep") {
			//fillWorkflowStep(&vxItem, &vertex.Properties, &vertex.EdgesOut, "workflowStep", "")
			fillTosca(&vxItem, &vertex.Properties, "workflowStep", "")
		} else if isToscaVertex(vertex, "workflowActivity") {
			//fillTosca(&vxItem, &vertex.Properties, "workflowActivity", "")
			fillWorkflowActivity(&vxItem, &vertex.Properties, "workflowActivity", "")
		} else if isToscaVertex(vertex, "policyTrigger") {
			fillPolicyTrigger(&vxItem, &vertex.Properties, "policyTrigger", "")
		} else if isToscaVertex(vertex, "policy") {
			fillTosca(&vxItem, &vertex.Properties, "policy", "")
		} else if isToscaVertex(vertex, "substitution") {
			fillSubstitution(&vxItem, &vertex.Properties, "substitution", "")
		}

		//		var vertexItem string = "{uid: '_:clout.vertex.'" + ind + ", 'clout:edge': []}";
		if isToscaVertex(vertex, "substitution") {
			for _, edge := range vertex.EdgesOut {
				fillSubstitutionEdge(&vxItem, edge)
			}
		} else {
			for _, edge := range vertex.EdgesOut {
				fillEdge(&vxItem, edge)
			}
		}

		vertexItems = append(vertexItems, vxItem)

	}

	cloutItem["clout:vertex"] = vertexItems

	topologyName := extractTopologyName(urlString)
	cloutItem["clout:name"] = topologyName
	cloutItem["clout:version"] = clout.Version
	cloutItem["clout:grammarversion"] = grammarVersions
	cloutItem["clout:import"] = internalImport

	props := clout.Properties["tosca"].(ard.Map)

	bytes, error := json.Marshal(props)
	if error == nil {
		cloutItem["clout:properties"] = string(bytes)
	}

	//for key, value := range props {
	//	fmt.Println("key: ", key, "value: ", value)
	//}
	dgraphset.Set = append(dgraphset.Set, cloutItem)

	// write out the Dgraph data in JSON format
	if printout {
		err := format.WriteOrPrint(dgraphset, "json", true, "")
		common.FailOnError(err)
		fmt.Println("-")
		fmt.Println("---------------------------------------------------")
		fmt.Println("-")
	}

	// save clout data into Dgraph
	SaveCloutGraph(&dgraphset, db.Dburl)
	return nil
}

func isTosca(metadata *ard.Map, etype string) bool {
	if *metadata != nil {
		if !reflect.ValueOf(metadata).IsNil() {

			entityType := (*metadata)["kind"]

			return (entityType == etype)
		}
	}

	return false
}
func isToscaVertex(vertex *clout.Vertex, etype string) bool {
	var data = vertex.Metadata["puccini-tosca"]
	if data != nil {
		var metadata = data.(ard.Map)
		return isTosca(&metadata, etype)
	}

	return false
}
func isToscaEdge(edge *clout.Edge, etype string) bool {
	var data = edge.Metadata["puccini-tosca"]
	if data != nil {
		var metadata = data.(ard.Map)
		return isTosca(&metadata, etype)
	}

	return false
}

func fillTosca(item *ard.Map, entity *ard.Map, ttype string, prefix string) error {
	//if prefix == nil {
	//	prefix = "";
	//}
	(*item)[prefix+"tosca:entity"] = ttype
	(*item)[prefix+"tosca:name"] = (*entity)["name"]
	(*item)[prefix+"tosca:description"] = (*entity)["description"]
	if (*entity)["types"] != nil {
		mapx := ((*entity)["types"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:types"] = string(bytes)
		}
	}

	if (*entity)["properties"] != nil {
		mapx := ((*entity)["properties"]).(ard.Map)
		//	propmap := make(ard.Map)
		//	for key, valuemap := range mapx {
		//		propmap[key] = valuemap.(ard.Map)["value"]
		//	}
		//	bytes, error = json.Marshal(propmap)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:properties"] = string(bytes)
			//(*item)[prefix+"tosca:properties"] = mapx
		}
	}
	if (*entity)["attributes"] != nil {
		mapx := (*entity)["attributes"].(ard.Map)
		//	propmap = make(ard.Map)
		//	for key, valuemap := range mapx {
		//		propmap[key] = valuemap.(ard.Map)["value"]
		//	}
		//	bytes, error = json.Marshal(propmap)
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)[prefix+"tosca:attributes"] = string(bytes)
			//(*item)[prefix+"tosca:attributes"] = mapx
		}
	}

	return nil
}

func fillToscaCapabilities(item *ard.Map, entity *ard.Map, type_ string, prefix string, key string) error {
	fillTosca(item, entity, "capability", "")
	(*item)[prefix+"tosca:key"] = key
	(*item)[prefix+"tosca:maxRelationshipCount"] = (*entity)["maxRelationshipCount"]
	(*item)[prefix+"tosca:minRelationshipCount"] = (*entity)["minRelationshipCount"]

	return nil
}

func fillNodeTemplate(item *ard.Map, nodeTemplate *ard.Map) error {
	fillTosca(item, nodeTemplate, "nodeTemplate", "")

	itemCapabilities := make([]ard.Map, 0)
	var capabilities ard.Map = (*nodeTemplate)["capabilities"].(ard.Map)
	var cap ard.Map
	/*for _, capability := range capabilities {
		cap = capability.(ard.Map)
		capabilityItem := make(ard.Map)
		fillTosca(&capabilityItem, &cap, "capability", "")
		itemCapabilities = append(itemCapabilities, capabilityItem)
	}*/

	for k, capability := range capabilities {
		cap = capability.(ard.Map)
		capabilityItem := make(ard.Map)
		fillToscaCapabilities(&capabilityItem, &cap, "capability", "", k)
		itemCapabilities = append(itemCapabilities, capabilityItem)
	}

	(*item)["capabilities"] = itemCapabilities

	if (*nodeTemplate)["interfaces"] != nil {
		mapx := (*nodeTemplate)["interfaces"].(ard.Map)
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)["tosca:interfaces"] = string(bytes)
		}
	}

	if (*nodeTemplate)["requirements"] != nil {
		mapx := (*nodeTemplate)["requirements"].([]interface{})
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)["tosca:requirements"] = string(bytes)
		}
	}

	if (*nodeTemplate)["directives"] != nil {
		dir := (*nodeTemplate)["directives"]
		var bytes []byte
		var error error
		if reflect.TypeOf(dir).String() == "[]interface{}" {
			mapx := dir.([]interface{})
			bytes, error = json.Marshal(mapx)
		} else if reflect.TypeOf(dir).String() == "[]string" {
			mapx := dir.([]string)
			bytes, error = json.Marshal(mapx)
		}
		//mapx := (*nodeTemplate)["directives"].([]string)
		//bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)["tosca:directives"] = string(bytes)
		}
	}

	return nil
}

func fillEdge(item *ard.Map, edge *clout.Edge) error {

	edgeItem := make(ard.Map)
	edgeItem["uid"] = "_:clout.vertex." + edge.Target.ID
	prefix := "clout:edge|"

	if isToscaEdge(edge, "relationship") {
		fillRelationship(&edgeItem, &edge.Properties)
	} else if isToscaEdge(edge, "nodeTemplateTarget") {
		fillTosca(&edgeItem, &edge.Properties, "nodeTemplateTarget", prefix)
	} else if isToscaEdge(edge, "onSuccess") {
		fillTosca(&edgeItem, &edge.Properties, "onSuccess", prefix)
	} else if isToscaEdge(edge, "onFailure") {
		fillTosca(&edgeItem, &edge.Properties, "onFailure", prefix)
	} else if isToscaEdge(edge, "workflowActivity") {
		fillTosca(&edgeItem, &edge.Properties, "workflowActivity", prefix)
		edgeItem[prefix+"tosca:sequence"] = edge.Properties["sequence"]
	} else if isToscaEdge(edge, "groupTarget") {
		fillTosca(&edgeItem, &edge.Properties, "groupTarget", prefix)
	} else if isToscaEdge(edge, "policyTrigger") {
		fillTosca(&edgeItem, &edge.Properties, "policyTrigger", prefix)
	} else if isToscaEdge(edge, "capabilityMapping") {
		fillTosca(&edgeItem, &edge.Properties, "capabilityMapping", prefix)
		edgeItem[prefix+"tosca:capability"] = edge.Properties["capability"]
	} else if isToscaEdge(edge, "requirementMapping") {
		fillTosca(&edgeItem, &edge.Properties, "requirementMapping", prefix)
		edgeItem[prefix+"tosca:requirement"] = edge.Properties["requirement"]
		edgeItem[prefix+"tosca:requirementName"] = edge.Properties["requirementName"]
	} else if isToscaEdge(edge, "interfaceMapping") {
		fillTosca(&edgeItem, &edge.Properties, "interfaceMapping", prefix)
	}

	var edgeItems []*ard.Map
	var edges = (*item)["clout:edge"]
	if edges == nil {
		edgeItems = make([]*ard.Map, 0)
	} else {
		edgeItems = (*item)["clout:edge"].([]*ard.Map)
	}

	(*item)["clout:edge"] = append(edgeItems, &edgeItem)

	return nil
}

func fillSubstitutionEdge(item *ard.Map, edge *clout.Edge) error {

	edgeItem := make(ard.Map)
	//edgeItem["uid"] = "_:clout.vertex." + edge.Target.ID
	
	prefix := "clout:edge|"
	edgeItem[prefix+"tosca:vertexId"] = edge.Target.ID

	if isToscaEdge(edge, "capabilityMapping") {
		fillTosca(&edgeItem, &edge.Properties, "capabilityMapping", prefix)
		edgeItem[prefix+"tosca:capability"] = edge.Properties["capability"]
		edgeItem["uid"] = "_:clout.vertex." + edge.Target.ID + "." + edge.Properties["capability"].(string)
	} else if isToscaEdge(edge, "requirementMapping") {
		fillTosca(&edgeItem, &edge.Properties, "requirementMapping", prefix)
		edgeItem[prefix+"tosca:requirement"] = edge.Properties["requirement"]
		edgeItem[prefix+"tosca:requirementName"] = edge.Properties["requirementName"]
		edgeItem["uid"] = "_:clout.vertex." + edge.Target.ID + "." + edge.Properties["requirement"].(string)
	} else if isToscaEdge(edge, "interfaceMapping") {
		fillTosca(&edgeItem, &edge.Properties, "interfaceMapping", prefix)
	}

	var edgeItems []*ard.Map
	var edges = (*item)["clout:edge"]
	if edges == nil {
		edgeItems = make([]*ard.Map, 0)
	} else {
		edgeItems = (*item)["clout:edge"].([]*ard.Map)
	}

	(*item)["clout:edge"] = append(edgeItems, &edgeItem)

	return nil
}

func fillRelationship(item *ard.Map, relationship *ard.Map) error {
	// As facets
	prefix := "clout:edge|"
	fillTosca(item, relationship, "relationship", prefix)

	(*item)[prefix+"tosca:capability"] = (*relationship)["capability"]
	if (*relationship)["interfaces"] != nil {
		mapx := (*relationship)["interfaces"].(ard.Map)
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)[prefix+"tosca:interfaces"] = string(bytes)
		}
	}

	return nil
}

func fillGroup(item *ard.Map, props *ard.Map, type_ string, prefix string) error {

	fillTosca(item, props, type_, "")

	if (*props)["interfaces"] != nil {
		mapx := (*props)["interfaces"].(ard.Map)
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)["tosca:interfaces"] = string(bytes)
		}
	}

	return nil
}

func fillWorkflowActivity(item *ard.Map, entity *ard.Map, ttype string, prefix string) error {
	fillTosca(item, entity, "workflowActivity", "")

	if (*entity)["callOperation"] != nil {
		mapx := ((*entity)["callOperation"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:callOperation"] = string(bytes)
		}
	}
	(*item)[prefix+"tosca:setNodeState"] = (*entity)["setNodeState"]

	return nil
}

func fillPolicyTrigger(item *ard.Map, entity *ard.Map, ttype string, prefix string) error {
	fillTosca(item, entity, "policyTrigger", "")

	if (*entity)["operation"] != nil {
		mapx := ((*entity)["operation"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:operation"] = string(bytes)
		}
	}

	(*item)[prefix+"tosca:event_type"] = (*entity)["event_type"]

	if (*entity)["condition"] != nil {
		mapx := ((*entity)["condition"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:condition"] = string(bytes)
		}
	}

	if (*entity)["action"] != nil {
		mapx := ((*entity)["action"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:action"] = string(bytes)
		}
	}

	return nil
}

func fillSubstitution(item *ard.Map, entity *ard.Map, ttype string, prefix string) error {
	fillTosca(item, entity, "substitution", "")

	if (*entity)["inputs"] != nil {
		mapx := ((*entity)["inputs"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:inputs"] = string(bytes)
		}
	}
	(*item)[prefix+"tosca:type"] = (*entity)["type"]
	(*item)[prefix+"tosca:dependencies"] = (*entity)["dependencies"]

	if (*entity)["typeMetadata"] != nil {
		mapx := ((*entity)["typeMetadata"]).(ard.Map)
		bytes, error := json.Marshal(mapx)
		if error == nil {
			(*item)[prefix+"tosca:typeMetadata"] = string(bytes)
		}
	}

	if (*entity)["substitutionFilter"] != nil {
		mapx := (*entity)["substitutionFilter"].([]interface{})
		bytes, error := json.Marshal(mapx)

		if error == nil {
			(*item)[prefix+"tosca:substitutionFilter"] = string(bytes)
		}
	}

	return nil
}

func extractTopologyName(urlString string) string {

	ind := strings.LastIndex(urlString, "/")

	var topologyName = urlString
	if ind == -1 {
		ind = strings.LastIndex(urlString, "\\")
	}
	if ind > -1 {
		topologyName = urlString[ind+1:]
	}
	ind = strings.LastIndex(topologyName, ".")
	if ind > -1 {
		topologyName = topologyName[:ind]
	}

	return topologyName
}
