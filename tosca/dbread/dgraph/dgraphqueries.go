package dgraph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/reflection"
)

type Namespace struct {
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

var uniqueInstancesMap map[string]bool

func init() {
	uniqueInstancesMap = make(map[string]bool)
	uniqueInstancesMap["DataType"] = true
	uniqueInstancesMap["CapabilityType"] = true
	uniqueInstancesMap["NodeType"] = true
	uniqueInstancesMap["NodeTemplate"] = true
	uniqueInstancesMap["InterfaceType"] = true
	uniqueInstancesMap["RelationshipType"] = true
	uniqueInstancesMap["ArtifactType"] = true
	uniqueInstancesMap["PolicyType"] = true
	uniqueInstancesMap["GroupType"] = true
	uniqueInstancesMap["ServiceTemplate"] = true
	uniqueInstancesMap["RelationshipTemplate"] = true
	uniqueInstancesMap["CapabilityAssignment"] = true
}

func (dgt *DgraphTemplate) FindCompByTypeAndNamespace(dgtype string, name string, namespace string) (bool, string, string, error) {
	var uid string
	var version string
	var err error
	var fnd bool

	query1 := `
	query {
		comp(func: eq(dgraph.type,"%s")) @cascade @filter (eq(name,"%s")){
			uid
			name
			namespace @filter (eq(url,"%s")){
				uid
				url
				version
			}
		  }
	}`

	nquad := fmt.Sprintf(query1, dgtype, name, namespace)

	if resp, err := dgt.ExecQuery(nquad); err == nil {

		type NamespaceType struct {
			Uid     string `json:"uid"`
			Url     string `json:"url"`
			Version string `json:"version"`
		}
		type UidType struct {
			Uid       string        `json:"uid"`
			Namespace NamespaceType `json:"namespace"`
		}
		type Root struct {
			Uids []UidType `json:"comp"`
		}

		var r Root
		err = json.Unmarshal(resp.Json, &r)

		if err == nil {
			uids := r.Uids
			if len(uids) > 0 {
				uid = uids[0].Uid
				version = uids[0].Namespace.Version
				fnd = true
			}

		}
	}

	return fnd, uid, version, err
}
func (dgt *DgraphTemplate) FindCompByUid(uid string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
		comp(func: uid(<%s>)) {
			expand(_all_){
				expand(_all_){}
			}
		  }
	}`

	nquad := fmt.Sprintf(query1, uid)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}

func (dgt *DgraphTemplate) FindNestedComp(uid string, key string, name string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s @filter (eq(name,"%s")){
	    	uid
	    	expand(_all_){
	      		expand(_all_){}
			}
		}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key, name)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}
func (dgt *DgraphTemplate) FindNestedComps(uid string, key string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s {
	    	uid
	    	expand(_all_){
	      		expand(_all_){}
			}
		}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}

func (dgt *DgraphTemplate) FindAttributeComps(uid string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		attributes {
			uid
			name
			functionname
			fnarguments
		}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}

func (dgt *DgraphTemplate) FindConditionComps(uid string, key string, name string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
		comp(func: uid(<%s>)) {
			 %s @filter (eq(name,"%s")){
			 uid
			  conditionclauses {
				directassertiondefinition{
					name
					constraintclause {
						operator
						arguments
				    }
				}
			  }
			}
		}
	  }
	`
	nquad := fmt.Sprintf(query1, uid, key, name)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}

func (dgt *DgraphTemplate) FindValueComp(uid string, key string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s {
			uid
			name
			description
			myvalue
			myvaluetype
			functionname
			fnarguments
			constraintclauses{
			  uid
			  name
			}
			rendered
		}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}
func (dgt *DgraphTemplate) FindNodeTemplateComp(uid string, key string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s {
			uid
			name
			description
			namespace
			properties {
				uid
				name
			}
			nodetypename
			nodetype {
				uid
				name
			}
			directives
			copynodetemplatename
			copynodetemplate {
				uid
				name
			}
			attributes {
				uid
				name
			}
			capabilities {
				uid
				name
			}
			requirements {
				uid
				name
			}
			requirementtargetsnodefilter {
				uid
				name
			}
			interfaces {
				uid
				name
			}
			artifacts {
				uid
				name
			}
			}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}
func (dgt *DgraphTemplate) FindPolicyComp(uid string, key string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s {
			uid
			name
			namespace {
				uid
				name
			}
			policytypename
			description
			properties {
				uid
				name
			}
			targetnodetemplateorgroupnames
			triggerdefinitions {
				uid
				name
			}
			policytype  {
				uid
				name
				metadata
			}
			targetnodetemplates {
				uid
				name
			}
			targetgroups {
				uid
				name
			}
			}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}
func (dgt *DgraphTemplate) FindPolicyTypeComp(uid string, key string) (map[string]interface{}, error) {

	var err error
	var resp *api.Response

	query1 := `
	query {
	  comp(func: uid(<%s>)) {
		   %s {
			uid
			name
			namespace {
				uid
				name
			}
			propertydefinitions {
				uid
				name
			}
			targetnodetypeorgrouptypenames
			triggerdefinitions {
				uid
				name
			}
			targetnodetypes {
				uid
				name
			}
			targetgrouptypes {
				uid
				name
			}
			parent {
				uid
				name
			}
			parentname
			version
			description
			metadata {
				expand(_all_){}
			}
			}
	  }
	}
	`
	nquad := fmt.Sprintf(query1, uid, key)

	if resp, err = dgt.ExecQuery(nquad); err == nil {
		aMap := make(map[string]interface{})

		err = json.Unmarshal(resp.GetJson(), &aMap)

		return aMap, err

	}

	return nil, err
}

func (dgt DgraphTemplate) FindNamespace(nurl string) (bool, string, error) {
	var uid string
	var err error
	var fnd bool

	//if !dbsave {
	//	return false, uid, nil
	//}

	query := `
	query {
		namespace(func: eq(url, "%s")){
			uid
			url
			version
		}
	}`

	nquad := fmt.Sprintf(query, nurl)
	resp, err := dgt.ExecQuery(nquad)

	if err == nil {

		for key, value := range resp.GetUids() {
			uid = value
			Log.Debugf("name: %s,  value: %s", key, value)
		}
		Log.Debugf("json: %s", string(resp.GetJson()))
		type Nspace struct {
			Uid string `json:"uid,omitempty"`
			Url string `json:"url,omitempty"`
		}
		type Root struct {
			Me []Nspace `json:"namespace"`
		}

		var r Root
		err = json.Unmarshal(resp.Json, &r)
		if err == nil {
			nss := r.Me
			if len(nss) > 0 {
				uid = nss[0].Uid
				fnd = true
			}

		}
	}
	return fnd, uid, err
}

func (dgt DgraphTemplate) FindComp(name string, dgtype string, nsuid string, subjuid string, predicate string) (bool, string, error) {
	var uid string
	var err error
	var fnd bool
	//if !dbsave {
	//	return false, uid, nil
	//}

	query1 := `
	query {
		comp(func: eq(name,"%s")) @filter(eq(dgraph.type,"%s")) @cascade {
			uid
			namespace @filter (uid(<%s>))
		  }
	}`

	query2 := `
	query {
	    comp(func: uid(<%s>))@cascade  {
		%s @filter(eq(name,"%s")){
		  	uid
		  	name
			}
		}
	}`

	_, isType := uniqueInstancesMap[dgtype]
	//isQuery1 := bag.Uid == "" || bag.StNamespaceUid != nsuid || isType
	isQuery1 := subjuid == "" || isType

	/*if !isType {
		_, isType = query2Map[dgtype]
		isQuery1 = !isType
	}*/

	var nquad string
	if isQuery1 {
		nquad = fmt.Sprintf(query1, name, dgtype, nsuid)
	} else {
		nquad = fmt.Sprintf(query2, subjuid, predicate, name)
	}
	Log.Debugf("nquad: %s", nquad)

	resp, err := dgt.ExecQuery(nquad)

	if err == nil {

		Log.Debugf("json: %s", string(resp.GetJson()))
		type UidType struct {
			Uid string `json:"uid"`
		}
		type Root struct {
			Uids []UidType `json:"comp"`
		}

		if isQuery1 {
			var r Root
			err = json.Unmarshal(resp.Json, &r)

			if err == nil {
				uids := r.Uids
				if len(uids) > 0 {
					uid = uids[0].Uid
					fnd = true
				}

			}
		} else {
			var a interface{}
			err = json.Unmarshal(resp.Json, &a)
			if err == nil {
				b := a.(map[string]interface{})
				uidlist := b["comp"].([]interface{})
				if len(uidlist) > 0 {
					c := uidlist[0].(map[string]interface{})
					d := c[predicate].(interface{})
					var e map[string]interface{}
					_, ok := d.([]interface{})
					if ok {
						d1 := d.([]interface{})
						e = d1[0].(map[string]interface{})
					} else {
						e, ok = d.(map[string]interface{})
					}
					if ok {
						uid = e["uid"].(string)
						fnd = len(uid) > 0
					}
				}
			}

		}
	}
	return fnd, uid, err
}
func addOneField(dgtype string, nam string, nquad string, field reflect.Value, fieldType reflect.Type) string {

	if reflection.IsPtrToStruct(fieldType) ||
		reflection.IsSliceOfPtrToStruct(fieldType) ||
		reflection.IsMapOfStringToPtrToStruct(fieldType) ||
		(dgtype == "ConstraintClause" &&
			(nam == "NativeArgumentIndexes" || nam == "DataType")) {
		fmt.Printf("%s\n", nam)
	} else if field.CanInterface() {
		if fq, ok := FieldRegistries.QueryRegistry[nam]; ok {
			nquad = nquad + fq.Query(field)
		} else {
			f := field.Interface()
			val := reflect.ValueOf(f)
			if !val.IsValid() {
				fmt.Printf("%s :\n", strings.ToLower(nam))
			} else {
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}
				if !val.IsValid() {
					fmt.Printf("%s :\n", strings.ToLower(nam))
				} else {

					if ar, ok := FieldRegistries.ArrayRegistry[nam]; ok && ar {
						valstr := fmt.Sprintf("%v", val)
						dirList, err := ParseArrayString(valstr)
						common.FailOnError(err)
						for _, dir := range dirList {
							nquad = nquad + fmt.Sprintf(`
	_:comp <%s> "%s" .`, strings.ToLower(nam), dir)

						}
					} else {
						valstr := fmt.Sprintf("%v", val)
						valstr = strings.ReplaceAll(valstr, "\"", "\\\"")
						valstr = strings.Replace(valstr, "\n", "\\n", -1)
						fmt.Printf("%s : %s\n", strings.ToLower(nam), valstr)
						triple := fmt.Sprintf(`
	_:comp <%s> "%s" .`, strings.ToLower(nam), valstr)
						nquad = nquad + triple
						//Log.Debugf("nquad: %s", nquad)
					}
				}
			}
		}
	}
	return nquad
}

func AddFields(entityPtr interface{}, nquad string, dgtype string) string {

	entity := reflect.ValueOf(entityPtr).Elem()
	tags := reflection.GetAllFieldTagsForValue(entity)
	var keys []string
	for _, tag := range tags {
		t := strings.Split(tag, ",")
		keys = append(keys, t[0])
	}

	num := entity.NumField()
	fmt.Printf("\nnum of fields: %d\n", num)

	for ind := 0; ind < num; ind++ {
		//field := value.Type().Field(ind)
		field := entity.Field(ind)
		fieldType := field.Type()
		nam := entity.Type().Field(ind).Name
		/*if nam == "Range" {
			nam = nam + ""
		} else if strings.ToLower(nam) == "required" {
			nam = nam + ""
		} else if strings.ToLower(nam) == "propertydefinitions" {
			nam = nam + ""
		} else if nam == "operator" || strings.ToLower(nam) == "nativeargumentindexes" {
			nam = nam + ""
		}*/
		//if (dgtype == "DataType" ||
		//	dgtype == "InterfaceType" ||
		//	dgtype == "CapabilityType" ||
		//	dgtype == "NodeType") &&
		/*if nam == "Type" {
			nquad = AddFields(field.Interface(), nquad, nam)
		} else if dgtype == "PropertyDefinition" && nam == "AttributeDefinition" {
			nquad = AddFields(field.Interface(), nquad, nam)
		} else if dgtype == "ParameterDefinition" && nam == "PropertyDefinition" {
			nquad = AddFields(field.Interface(), nquad, nam)
		} else if dgtype == "AttributeDefinition" && nam == "Default" {
			if tsmethod, ok := ToscaMethodRegistry[QueryAttribValueKey]; ok {
				nquad = tsmethod.Process(&entityPtr, field.Type().Name())
			}
		} else if dgtype == "Artifact" && nam == "ArtifactDefinition" {
			nquad = AddFields(field.Interface(), nquad, nam)
		}*/
		key := dgtype + "." + nam
		if merge, ok := FieldRegistries.MergeRegistry[key]; ok && merge {
			nquad = AddFields(field.Interface(), nquad, nam)
		}
		if tsmkey, ok := FieldRegistries.ToscaMethodRegistry[key]; ok {
			if tsmethod, ok := ToscaMethodRegistry[tsmkey]; ok {
				nquad = nquad + tsmethod.Process(&entityPtr, nam)
			}
		}
		nquad = addOneField(dgtype, nam, nquad, field, fieldType)
	}
	fmt.Printf("\n")

	return nquad
}
func getRangeValues(vrange interface{}) string {
	value := reflect.ValueOf(vrange)
	num := value.NumField()
	fmt.Printf("\nnum of range fields: %d\n", num)
	var nquad string
	for ind := 0; ind < num; ind++ {
		field := value.Field(ind)
		//fieldType := field.Type()
		nam := value.Type().Field(ind).Name
		f := field.Interface()
		val := reflect.ValueOf(f)
		triple := fmt.Sprintf(`
	_:comp <%s> "%d" .`, strings.ToLower(nam), val)
		nquad = nquad + triple

	}
	return nquad
}
func getMetadataValues(field reflect.Value) string {
	var nquad string
	nquad = nquad + `
	_:mdata <dgraph.type> "Metadata" .`
	iter := field.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		triple := fmt.Sprintf(`
	_:mdata <%s> "%s" .`, k, v)
		nquad = nquad + triple
	}
	nquad = nquad + `
	_:comp <metadata> _:mdata .`

	return nquad
}

func ParseArrayString(args string) (ard.List, error) {
	beg := 0
	if strings.HasPrefix(args, "[") {
		beg++
	}
	end := 0
	if strings.HasSuffix(args, "]") {
		end++
	}
	str := args[beg : len(args)-end]
	argums := strings.Split(str, " ")

	var argList ard.List
	for _, arg := range argums {
		argList = append(argList, arg)
	}

	return argList, nil
}

func BuildInsertQuery(saveFields SaveFields) string {

	nquad := `_:comp <dgraph.type> "%s" .
	_:comp <name> "%s" .
	_:comp <namespace> <%s> .`
	nquad = fmt.Sprintf(nquad, saveFields.DgType, saveFields.Name, saveFields.Nsuid)
	nquad = AddFields(saveFields.EntityPtr, nquad, saveFields.DgType)
	Log.Debugf("nquad: %s", nquad)
	return nquad
}
