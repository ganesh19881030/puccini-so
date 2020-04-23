package cmd

import (
	"fmt"
	//"io"
	"encoding/json"

	//"github.com/op/go-logging"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	. "github.com/tliron/puccini/tosca/database"
	//"github.com/tliron/puccini/url"
)

//var log = logging.MustGetLogger("puccini-js")

//var output string

func CreateInstance(clout_ *clout.Clout, template string, name string, uid string) error {
	var err error
	var printout = true
	var vertexItems []ard.Map
	var cloutItem = make(ard.Map)
	var dgraphset = DgraphSet{}

	cloutItem["clout:name"] = name
	cloutItem["clout:type"] = "service"

	/*tmpl := make(ard.Map)
	tmpl["clout:uid"] = uid
	tmpl["clout:name"] = template*/

	//cloutItem["clout:template"] = tmpl
	cloutItem["clout:templateUID"] = uid
	cloutItem["clout:templateName"] = template
	properties := clout_.Properties["tosca"].(ard.Map)

	bytes, error := json.Marshal(properties)
	if error == nil {
		cloutItem["clout:properties"] = string(bytes)
	}
	// Node templates
	for _, vertex := range clout_.Vertexes {
		//		v := clout_.NewVertex(clout.NewKey())
		/*ind := vertex.ID
		vxItem := make(ard.Map)
		vxItem["uid"] = "_:clout.vertex." + ind*/

		//vxItem["tosca:vertexId"] = ind
		//vxItem["clout:edge"] = make([]*ard.Map, 0)

		if isToscaVertex(vertex, "nodeTemplate") {
			ind := vertex.ID
			vxItem := make(ard.Map)
			vxItem["uid"] = "_:clout.vertex." + ind

			props := vertex.Properties
			vxItem["tosca:entity"] = "node"
			vxItem["tosca:name"] = props["name"]
			vxItem["tosca:description"] = props["description"]
			if props["types"] != nil {
				mapx := (props["types"]).(ard.Map)
				bytes, error := json.Marshal(mapx)
				if error == nil {
					vxItem["tosca:types"] = string(bytes)
				}
			}

			if (props)["properties"] != nil {
				mapx := props["properties"].(ard.Map)
				bytes, error := json.Marshal(mapx)
				if error == nil {
					(vxItem)["tosca:properties"] = string(bytes)
				}
			}
			if (props)["attributes"] != nil {
				mapx := props["attributes"].(ard.Map)
				map1 := make(ard.Map)
				for k, attr := range mapx {
					map2 := make(ard.Map)
					attr1 := attr.(ard.Map)
					map2["value"] = attr1["value"]
					map1[k] = map2
				}
				//bytes, error := json.Marshal(mapx)
				bytes, error := json.Marshal(map1)

				if error == nil {
					vxItem["tosca:attributes"] = string(bytes)
				}
			}
			if (props)["interfaces"] != nil {
				mapx := props["interfaces"].(ard.Map)
				bytes, error := json.Marshal(mapx)
				if error == nil {
					(vxItem)["tosca:interfaces"] = string(bytes)
				}
			}
			if (props)["artifacts"] != nil {
				mapx := props["artifacts"].(ard.Map)
				bytes, error := json.Marshal(mapx)
				if error == nil {
					(vxItem)["tosca:artifacts"] = string(bytes)
				}
			}
			vertexItems = append(vertexItems, vxItem)
		}

	}

	cloutItem["clout:vertex"] = vertexItems
	dgraphset.Set = append(dgraphset.Set, cloutItem)

	// write out the Dgraph data in JSON format
	if printout {
		err := format.WriteOrPrint(dgraphset, "json", true, "")
		common.FailOnError(err)
		fmt.Println("-")
		fmt.Println("---------------------------------------------------")
		fmt.Println("-")
	}
	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)
	//cloutdb1 := NewCloutDb1{dburl: }
	// save clout data into Dgraph
	SaveCloutGraph(&dgraphset, dburl)

	return err
}
