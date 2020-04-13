package db_tosca_v1_3

import (
	"errors"
	"strconv"
	"strings"

	"github.com/op/go-logging"
	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
)

var log = logging.MustGetLogger("dgraph.db_tosca_v1_3")

var DbObjectMap = make(dgraph.DbObjectMap)
var ErrorInvalidSaveDataObject string = "Data object passed to DbBuildInsertQuery function is not of type SaveFields."

func init() {
	DbObjectMap["ActivityDefinition"] = new(DbActivityDefinition)
	DbObjectMap["Artifact"] = new(DbArtifact)
	DbObjectMap["ArtifactDefinition"] = new(DbArtifactDefinition)
	//DbObjectMap["ArtifactType"] = new(DbArtifactType)
	DbObjectMap["AttributeDefinition"] = new(DbAttributeDefinition)
	DbObjectMap["AttributeMapping"] = new(DbAttributeMapping) // introduced in 1.3
	DbObjectMap["AttributeValue"] = new(DbAttributeValue)
	DbObjectMap["CapabilityAssignment"] = new(DbCapabilityAssignment)
	DbObjectMap["CapabilityDefinition"] = new(DbCapabilityDefinition)
	//DbObjectMap["CapabilityFilter"] = new(DbCapabilityFilter)
	DbObjectMap["CapabilityMapping"] = new(DbCapabilityMapping)
	DbObjectMap["CapabilityType"] = new(DbCapabilityType)
	DbObjectMap["ConditionClause"] = new(DbConditionClause)
	DbObjectMap["ConstraintClause"] = new(DbConstraintClause)
	DbObjectMap["DataType"] = new(DbDataType)
	DbObjectMap["DirectAssertionDefinition"] = new(DbDirectAssertionDefinition)

	// created for dgraph - this is not the same as tosca namespace
	DbObjectMap["DGNamespace"] = new(DbDGNamespace)

	DbObjectMap["EntrySchema"] = new(DbEntrySchema)
	/*DbObjectMap["EventFilter"] = new(DbEventFilter)
	DbObjectMap["Group"] = new(DbGroup)
	DbObjectMap["GroupType"] = new(DbGroupType)*/
	DbObjectMap["InterfaceAssignment"] = new(DbInterfaceAssignment)
	DbObjectMap["InterfaceDefinition"] = new(DbInterfaceDefinition)
	DbObjectMap["InterfaceMapping"] = new(DbInterfaceMapping) // introduced in 1.2
	DbObjectMap["InterfaceType"] = new(DbInterfaceType)
	DbObjectMap["Metadata"] = new(DbMetadata)
	DbObjectMap["NodeFilter"] = new(DbNodeFilter)
	DbObjectMap["NodeTemplate"] = new(DbNodeTemplate)
	DbObjectMap["NodeType"] = new(DbNodeType)
	//DbObjectMap["NotificationAssignment"] = new(DbNotificationAssignment) // introduced in 1.3
	DbObjectMap["NotificationDefinition"] = new(DbNotificationDefinition) // introduced in 1.3
	DbObjectMap["NotificationOutput"] = new(DbNotificationOutput)         // introduced in 1.3
	DbObjectMap["OperationAssignment"] = new(DbOperationAssignment)
	DbObjectMap["OperationDefinition"] = new(DbOperationDefinition)
	DbObjectMap["InterfaceImplementation"] = new(DbInterfaceImplementation)
	DbObjectMap["ParameterDefinition"] = new(DbParameterDefinition)
	DbObjectMap["Policy"] = new(DbPolicy)
	DbObjectMap["PolicyType"] = new(DbPolicyType)
	DbObjectMap["PropertyDefinition"] = new(DbPropertyDefinition)
	DbObjectMap["PropertyFilter"] = new(DbPropertyFilter)
	DbObjectMap["PropertyMapping"] = new(DbPropertyMapping) // introduced in 1.2
	DbObjectMap["Range"] = new(DbRange)
	DbObjectMap["RangeEntity"] = new(DbOccurrences)
	//DbObjectMap["RelationshipAssignment"] = new(DbRelationshipAssignment)
	DbObjectMap["RelationshipDefinition"] = new(DbRelationshipDefinition)
	//DbObjectMap["RelationshipTemplate"] = new(DbRelationshipTemplate)
	DbObjectMap["RelationshipType"] = new(DbRelationshipType)
	//DbObjectMap["Repository"] = new(DbRepository)
	DbObjectMap["RequirementAssignment"] = new(DbRequirementAssignment)
	DbObjectMap["RequirementDefinition"] = new(DbRequirementDefinition)
	DbObjectMap["RequirementMapping"] = new(DbRequirementMapping)
	/*DbObjectMap["scalar-unit.size"] = new(DbScalarUnitSize)
	DbObjectMap["scalar-unit.time"] = new(DbScalarUnitTime)
	DbObjectMap["scalar-unit.frequency"] = new(DbScalarUnitFrequency)*/
	DbObjectMap["ServiceTemplate"] = new(DbServiceTemplate)
	DbObjectMap["SubstitutionMappings"] = new(DbSubstitutionMappings)
	DbObjectMap["SubstitutionFilter"] = new(DbSubstitutionFilter)
	/*DbObjectMap["SubstitutionFilterCapability"] = new(DbSubstitutionFilterCapability)
	DbObjectMap["timestamp"] = new(DbTimestamp)*/
	DbObjectMap["TopologyTemplate"] = new(DbTopologyTemplate)
	DbObjectMap["TriggerDefinition"] = new(DbTriggerDefinition)
	DbObjectMap["TriggerDefinitionCondition"] = new(DbTriggerDefinitionCondition)
	//DbObjectMap["Unit"] = new(DbUnit)
	DbObjectMap["Value"] = new(DbValue)
	//DbObjectMap["version"] = new(DbVersion)
	//DbObjectMap["WorkflowActivityCallOperation"] = new(DbWorkflowActivityCallOperation) // introduced in 1.3
	DbObjectMap["ActivityDefinition"] = new(DbActivityDefinition)
	//DbObjectMap["WorkflowDefinition"] = new(DbWorkflowDefinition)
	//DbObjectMap["WorkflowPreconditionDefinition"] = new(DbWorkflowPreconditionDefinition)
	//DbObjectMap["WorkflowStepDefinition"] = new(DbWorkflowStepDefinition)

}

func ConvertDbResponseArrayToMap(mdata *[]interface{}, readerName string) ard.Map {
	mapData := make(ard.Map)
	for _, data := range *mdata {
		data.(ard.Map)["readername"] = readerName
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			mapData[mkey] = data
		}
	}

	return mapData

}

func ConvertDbResponseArrayToSequencedList(childData interface{}, readerName string) []interface{} {
	var xdata []interface{}
	var xmap ard.Map

	for _, data := range childData.([]interface{}) {
		data.(ard.Map)["readername"] = readerName
		mkey, ok := data.(ard.Map)["name"].(string)
		if ok {
			xmap = make(ard.Map)
			xmap[mkey] = data
			xdata = append(xdata, xmap)
		}
	}

	return xdata
}

func ParseArguments(args string) (ard.List, error) {
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

func ParseMapArguments(args string) (ard.Map, error) {
	beg := 0
	if strings.HasPrefix(args, "[map[") {
		beg = beg + 5
	}
	end := 0
	if strings.HasSuffix(args, "]]") {
		end = end + 2
	}
	str := args[beg : len(args)-end]
	argums := strings.Split(str, " ")

	argMap := make(ard.Map)
	for _, argEntry := range argums {
		kvpairs := strings.Split(argEntry, ":")
		argMap[kvpairs[0]] = kvpairs[1]
	}

	return argMap, nil
}
func ParseScalarUnitArguments(args string) (ard.List, error) {
	beg := 0
	if strings.HasPrefix(args, "[") {
		beg++
	}
	end := 0
	if strings.HasSuffix(args, "]") {
		end++
	}
	str := args[beg : len(args)-end]
	argums := strings.Split(str, ",")

	var argList ard.List
	for _, arg := range argums {
		argList = append(argList, arg)
	}

	return argList, nil
}

// transformValueData - transforms value data in dgraph to what is expected by Puccini
func TransformValueData(childData interface{}) interface{} {
	var ok bool
	var fname string
	var vdata ard.Map
	fname, ok = childData.(ard.Map)["functionname"].(string)
	if ok {
		vdata = make(ard.Map)
		vdata[fname] = childData.(ard.Map)["fnarguments"]
		return vdata
	} else {
		var val interface{}
		var typ interface{}
		if val, ok = childData.(ard.Map)["myvalue"]; ok {
			if typ, ok = childData.(ard.Map)["myvaluetype"]; ok {
				strval := val.(string)
				val = ParseValue(strval, typ.(string))
				return val
			}
		}
	}

	return childData
}

func AddReaderNameToData(childData *interface{}, readerName string) interface{} {
	cData := *childData
	switch cData.(type) {
	case ard.Map:
		cData.(ard.Map)["readername"] = readerName
		break
	case []interface{}:
		if len(cData.([]interface{})) == 1 {
			cData = cData.([]interface{})[0]
			cData.(ard.Map)["readername"] = readerName
		} else {
			for _, data := range cData.([]interface{}) {
				data.(ard.Map)["readername"] = readerName
			}
		}
		break
	}

	return cData
}

// TransformConditionData - transforms condition data in dgraph to what is expected by Puccini
func TransformConditionData(childData interface{}) interface{} {
	cdData := childData.(ard.Map)["conditionclauses"]
	var cList ard.List
	for _, cData := range cdData.([]interface{}) {
		dadMap := cData.(ard.Map)["directassertiondefinition"]
		dadName := dadMap.(ard.Map)["name"].(string)
		conList := dadMap.(ard.Map)["constraintclause"]
		//dadMap[dadName] = conList
		var nconList ard.List
		for _, conData := range conList.([]interface{}) {
			conMap := make(ard.Map)
			op := conData.(ard.Map)["operator"].(string)
			args := conData.(ard.Map)["arguments"].(string)
			argList, err := ParseArguments(args)
			if err == nil {
				conMap[op] = argList
				nconList = append(nconList, conMap)
			}
		}
		ndadMap := make(ard.Map)
		ndadMap[dadName] = nconList
		cList = append(cList, ndadMap)
	}

	return cList

}

func ExtractNameFromFieldData(cData *interface{}) (string, error) {
	var name string
	var ok bool
	if cData != nil {
		switch (*cData).(type) {
		case ard.Map:
			name, ok = ((*cData).(ard.Map)["name"]).(string)
			break
		case []interface{}:
			data := (*cData).([]interface{})[0]
			name, ok = (data.(ard.Map)["name"]).(string)
			break
		}
	}
	if ok {
		return name, nil
	} else {
		return "", errors.New("No name field found in data.")
	}
}

func ParseValue(strval string, valtype string) interface{} {
	var err error
	var value interface{}

	switch valtype {
	case "bool":
		value, err = strconv.ParseBool(strval)
		common.FailOnError(err)
	case "int":
		value, err = strconv.ParseInt(strval, 10, 32)
		common.FailOnError(err)
	case "float":
		value, err = strconv.ParseFloat(strval, 64)
		common.FailOnError(err)
	default:
		value = strval
	}

	return value

}
