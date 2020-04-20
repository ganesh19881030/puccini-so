package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
	"github.com/tliron/puccini/tosca/parser"
	"github.com/tliron/puccini/tosca/reflection"
)

// TravelBag struct holds parameters that are passed along as we traverse the tree
type TravelBag struct {
	Uid             string
	Predicate       string
	ChildName       string
	ChildDgraphType string
	ChildNspace     string
	Level           int
	StNamespaceUid  string
	Mapkey          string
	Dgt             *dgraph.DgraphTemplate
}

// DbTraverser is type of function executed for each node in the tree
type DbTraverser func(interface{}, *TravelBag) bool

var linkmap map[string]bool
var entitymap map[interface{}]bool

/// Traverse SaveServiceTemplate method implementation of CloutDB interface for CloutDB1 instance
func Traverse(pcontext *parser.Context) error {
	var err error
	var uid string
	var linkexists bool
	var nurl string
	var version string
	var stname string
	var name string
	var ctxt *tosca.Context

	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)

	epbatchsize := 100
	epcount := 0
	eplimit := epcount + epbatchsize
	dgraphClient, conn, err := GetDgraphClient1(dburl)
	defer conn.Close()
	ctx := context.Background()
	urlmap := make(map[string]string)
	dgt := dgraph.DgraphTemplate{
		Client: dgraphClient,
		Ctxt:   ctx,
	}

	if err == nil {
		//schemaPresent, err := dbread.IsSchemaDefined(ctx, dgraphClient)
		//if err == nil && !schemaPresent {
		err = dgraph.SaveToscaSchema(ctx, dgraphClient)
		common.FailOnError(err)

		//}
		linkmap = make(map[string]bool)
		entitymap = make(map[interface{}]bool)
		traverseContext("TEST", pcontext, func(entityPtr interface{}, bag *TravelBag) bool {
			if entityPtr == nil {
				return false
			} else {
				epcount++
				strType := GetEntityType(entityPtr)
				Log.Infof("processing [%s] entity...", strType)
				_, ok := entityPtr.(tosca.Contextual)
				if ok {

					ctxt = tosca.GetContext(entityPtr)

					name = reflection.GetEntityName(entityPtr)
					if name == "" {
						name = ctxt.Name
					}
					Log.Debugf("name: %s, url: %s, path: %s\n", name, ctxt.URL, ctxt.Path)
					Log.Debugf("Child Name: %s, uid: %s, level: %d\n", bag.ChildName, bag.Uid, bag.Level)

					nurl = strings.Replace(ctxt.URL.String(), "\\", "/", -1)
					version = ctxt.Version
					if bag.Level == 0 {
						stname = ExtractTopologyName(ctxt.URL.String())
					}

				} else {
					nurl = bag.ChildNspace
					name = bag.Predicate
				}
				bag.Dgt = &dgt
				PersistNamespace(nurl, version, urlmap, &dgt)
				if bag.Level == 0 {
					bag.StNamespaceUid = urlmap[nurl]
					bag.Uid, name, err = PersistToscaComponent(entityPtr, stname, "ServiceTemplate", nurl, urlmap, bag)
					bag.ChildName = "servicetemplate"
					bag.ChildNspace = nurl
				} else {
					dgraphType := strType

					if dgraphType == "Value" {
						Log.Debugf("*** dgraphType is Value ***")
						if bag.Predicate == "default" {
							AddFieldToComponent(entityPtr, bag)
							return false
						}
						/*} else if dgraphType == "RelationshipDefinition" {
								pathelem := ctxt.Path[len(ctxt.Path)-4]
								name = pathelem.Value.(string)
								name = bag.Mapkey
								Log.Debugf("Changed name for relationshipdefinition to %s", name)
							} else if dgraphType == "EntrySchema" {
								pathelem := ctxt.Path[len(ctxt.Path)-2]
								name = pathelem.Value.(string)
								Log.Debugf("Changed name for entry_schema to %s, bag,Mapkey: %s", name, bag.Mapkey)
							} else if dgraphType == "SubstitutionMappings" {
								Log.Debugf("*** dgraphType is %s ***", dgraphType)
								var val reflect.Value
								val, err = GetFieldValue(entityPtr, "NodeTypeName")
								if err == nil {
									name = val.Interface().(string)
								}
						} else if dgraphType == "NodeTemplate" {
							if name == "sdwan_hub_spoke" {
								Log.Debugf("*** dgraphType is %s ***", dgraphType)
							}*/
					} else if dgraphType == "" {
						Log.Errorf("*** dgraphType for predicate [%s] is undefined ***", bag.Predicate)
					}
					uid, name, err = PersistToscaComponent(entityPtr, name, dgraphType, nurl, urlmap, bag)
					common.FailOnError(err)
					bag.ChildName = name
					bag.ChildDgraphType = dgraphType
					bag.ChildNspace = nurl
					linkexists, err = LinkToscaComponents(uid, bag)
					common.FailOnError(err)
					if linkexists {
						return false
					}
					bag.Uid = uid
				}
				if epcount >= eplimit {
					eplimit = epcount + epbatchsize
				}
				_, processed := entitymap[entityPtr]
				if processed {
					return false
				} else {
					entitymap[entityPtr] = true
				}
			}
			return true
		})
	}

	Log.Infof("=> Processed %d entities <=", epcount)
	return err
}

func traverseContext(phase string, pcontext *parser.Context, traverse DbTraverser) {
	var bag TravelBag
	bag.Level = 0

	t := func(entityPtr interface{}, bag *TravelBag) bool {
		return traverse(entityPtr, bag)
	}

	Traverse2(pcontext.ServiceTemplate.EntityPtr, t, bag, pcontext.ServiceTemplate)
}
