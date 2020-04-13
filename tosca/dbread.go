package tosca

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"

	"github.com/tliron/puccini/tosca/dbread"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
	"github.com/tliron/puccini/tosca/reflection"
)

type SliceHolder struct {
	Sfield   *ReadField
	Scontext *Context
	IsList   bool
}

//var lookupReaders *dgraph.LookupReaders
var visitedNodes ard.Map
var inprogressCompFields ard.Map
var inwaitCompFields map[string]ard.List
var EntityStack common.Stack

var dbObjectMap dgraph.DbObjectMap

//var DbGrammar dgraph.DbGrammarMap = make(dgraph.DbGrammarMap)

func init() {
	visitedNodes = make(ard.Map)

	inprogressCompFields = make(ard.Map)

	inwaitCompFields = make(map[string]ard.List)

	EntityStack = make([]interface{}, 100)

	// hard coded for now
	// TODO: fetch tosca grammar version from config or cmd line options
	dbObjectMap = dbread.DbGrammar["dgraph.tosca_v1_3"]
}

// DbReadFields - reads TOSCA fields from DGraph database
// 				  From "read" tags
func (self *Context) DbReadFields(entityPtr interface{}) []string {

	var keys []string
	var tags map[string]string
	var uid string

	//log.Debugf("\nReading %s %s",self.Name,self.Path.String())
	entity := reflect.ValueOf(entityPtr).Elem()

	uid = self.Data.(ard.Map)["uid"].(string)
	self.Level++
	log.Debugf("\nReading %s %s %s, level: %d", uid, self.Name, self.Path.String(), self.Level)
	tags = reflection.GetAllFieldTagsForValue(entity)
	_, ok := tags["Parent"]
	if ok {
		readername := self.Data.(ard.Map)["readername"].(string)
		tags["Parent"] = readername + "," + readername
	}

	// Read tag overrides
	if self.ReadOverrides != nil {
		for fieldName, tag := range self.ReadOverrides {
			if tag != "" {
				tags[fieldName] = tag
			} else {
				// Empty tag means delete
				if _, ok := tags[fieldName]; !ok {
					panic(fmt.Sprintf("unknown field: \"%s\"", fieldName))
				}
				delete(tags, fieldName)
			}
		}
	}

	// Gather all tagged keys
	for _, tag := range tags {
		t := strings.Split(tag, ",")
		keys = append(keys, t[0])
	}

	// Parse tags
	var readFields []*ReadField
	for fieldName, tag := range tags {
		readField := DbNewReadField(fieldName, tag, self, entity)
		if readField.Important {
			// Important fields come first
			readFields = append([]*ReadField{readField}, readFields...)
		} else {
			readFields = append(readFields, readField)
		}
	}

	for _, readField := range readFields {
		if readField.Wildcard {
			// Iterate all keys that aren't tagged
			for key := range self.Data.(ard.Map) {
				tagged := false
				for _, k := range keys {
					if key == k {
						tagged = true
						break
					}
				}
				if !tagged {
					readField.Key = key
					readField.DbRead()
				}
			}
		} else {
			readField.DbRead()
		}
	}

	return keys
}

//
// DbNewReadField
//

func DbNewReadField(fieldName string, tag string, context *Context, entity reflect.Value) *ReadField {
	// TODO: is it worth caching some of this?

	t := strings.Split(tag, ",")

	var self = ReadField{
		FieldName: fieldName,
		Key:       t[0],
		Context:   context,
		Entity:    entity,
	}

	if context.ReadFromDb && len(t) > 1 {
		self.Key = t[1]
	}

	if self.Key == "?" {
		self.Wildcard = true
		self.Mode = ReadFieldModeItem
	} else {
		self.Mode = ReadFieldModeDefault
	}

	var readerName string
	if len(t) > 1 {
		readerName = t[1]
		if len(t) > 2 {
			readerName = t[2]
		}

		if strings.HasPrefix(readerName, "!") {
			// Important
			readerName = readerName[1:]
			self.Important = true
		}

		if strings.HasPrefix(readerName, "[]") {
			// List
			readerName = readerName[2:]
			self.Mode = ReadFieldModeList
		} else if strings.HasPrefix(readerName, "{}") {
			// Sequenced list
			readerName = readerName[2:]
			self.Mode = ReadFieldModeSequencedList
		}
		var ok bool
		self.Reader, ok = context.Grammar[readerName]
		if !ok {
			panic(fmt.Sprintf("reader not found: %s", readerName))
		}
		self.ReaderName = readerName
	}

	return &self
}

func (self *ReadField) DbRead() {
	var key string
	key = strings.ToLower(self.FieldName)
	if key == "" {
		fmt.Printf("\nKey not mapped for %s\n", self.FieldName)
		return
	}
	var childData interface{}
	var item interface{}
	var cmpfield string
	var ok bool
	var uid string
	var cuids []string

	childData, ok = self.Context.Data.(ard.Map)[key]
	if !ok {
		return
	}

	field := self.Entity.FieldByName(self.FieldName)

	uid = (self.Context.Data.(ard.Map)["uid"]).(string)
	cmpfield = uid + "/" + self.FieldName
	log.Debugf("\n===> fieldName = %s", cmpfield)

	if self.Reader != nil {
		fieldType := field.Type()
		if reflection.IsSliceOfPtrToStruct(fieldType) {
			// Field is compatible with []*interface{}
			slice := field
			childData = readFromDb(self, uid, key, false)
			self.Context.Data.(ard.Map)[key] = childData

			if key == "substitutionmappings" {
				self.Mode = ReadFieldModeList
			}

			switch self.Mode {
			case ReadFieldModeList:
				self.Context.FieldChild(key, childData).ReadListItemsDb(self.Reader, func(item interface{}) {
					slice = reflect.Append(slice, reflect.ValueOf(item))
				}, &cuids, self, false)

			case ReadFieldModeSequencedList:
				self.Context.FieldChild(key, childData).ReadSequencedListItemsDb(self.Reader, func(item interface{}) {
					slice = reflect.Append(slice, reflect.ValueOf(item))
				}, &cuids, self, false)

			case ReadFieldModeItem:
				length := slice.Len()
				var cuid string
				var isinprogok bool
				item, cuid, isinprogok = isReadInProgress(childData, self, nil, true)
				if item == nil && !isinprogok {
					item = self.Reader(self.Context.ListChild(length, childData))
					postReadProcess(cuid, item)
				}
				if item != nil {
					slice = reflect.Append(slice, reflect.ValueOf(item))
				}
			default:
				self.Context.FieldChild(key, childData).ReadMapItemsDb(self.Reader, func(item interface{}) {
					slice = reflect.Append(slice, reflect.ValueOf(item))
				}, &cuids, self, false)

			}
			if slice.IsNil() {
				// If we have no items, at least have an empty slice
				// so that "require" will not see a nil here
				slice = reflect.MakeSlice(slice.Type(), 0, 0)
			}
			field.Set(slice)
		} else if reflection.IsMapOfStringToPtrToStruct(fieldType) {

			childData = readFromDb(self, uid, key, true)

			self.Context.Data.(ard.Map)[key] = childData

			// Field is compatible with map[string]*interface{}
			switch self.Mode {
			case ReadFieldModeList:
				context := self.Context.FieldChild(key, childData)
				context.ReadListItemsDb(self.Reader, func(item interface{}) {
					context.setMapItem(field, item)
				}, &cuids, self, true)
			case ReadFieldModeSequencedList:
				context := self.Context.FieldChild(key, childData)
				context.ReadSequencedListItemsDb(self.Reader, func(item interface{}) {
					context.setMapItem(field, item)
				}, &cuids, self, true)
			case ReadFieldModeItem:
				context := self.Context.FieldChild(key, childData)
				var cuid string
				var isinprogok bool
				item, cuid, isinprogok = isReadInProgress(childData, self, context, false)
				if item == nil && !isinprogok {
					item = self.Reader(context)
					context.setMapItem(field, item)
					postReadProcess(cuid, item)
				} else if item != nil {
					context.setMapItem(field, item)
				}

			default:
				context := self.Context.FieldChild(key, childData)
				context.ReadMapItemsDb(self.Reader, func(item interface{}) {
					context.setMapItem(field, item)
				}, &cuids, self, true)

			}
		} else {
			childData = readItemFromDb(self, uid, key, &childData)
			var cuid string
			var inprogok bool
			item, cuid, inprogok = isReadInProgress(childData, self, nil, false)
			if item == nil && !inprogok {
				item = readNode(self, childData, key, cuid)
			}

			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		}
	} else {
		fieldEntityPtr := field.Interface()
		if reflection.IsPtrToString(fieldEntityPtr) {
			// Field is *string
			item := self.Context.FieldChild(key, childData).ReadString()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else if reflection.IsPtrToInt64(fieldEntityPtr) {
			// Field is *int64
			item := self.Context.FieldChild(key, childData).ReadInteger()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else if reflection.IsPtrToFloat64(fieldEntityPtr) {
			// Field is *float64
			item := self.Context.FieldChild(key, childData).ReadFloat()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else if reflection.IsPtrToBool(fieldEntityPtr) {
			// Field is *bool
			item := self.Context.FieldChild(key, childData).ReadBooleanDb()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else if reflection.IsPtrToSliceOfString(fieldEntityPtr) {
			// Field is *[]string
			item := self.Context.FieldChild(key, childData).ReadStringList()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else if reflection.IsPtrToMapOfStringToString(fieldEntityPtr) {
			// Field is *map[string]string
			item := self.Context.FieldChild(key, childData).ReadStringMap()
			if item != nil {
				field.Set(reflect.ValueOf(item))
			}
		} else {
			panic(fmt.Sprintf("\"read\" tag's field type \"%T\" is not supported in struct: %T.%s", fieldEntityPtr, self.Entity.Interface(), self.FieldName))
		}
	}
}

//
// Read helpers
//
func (self *Context) ReadBooleanDb() *bool {
	if self.ValidateType("string") {
		value, err := strconv.ParseBool(self.Data.(string))
		common.FailOnError(err)
		return &value
	}

	return nil
}

func (self *Context) ReadMapItemsDb(read Reader, process Processor, uids *[]string, rfield *ReadField, postProcess bool) bool {
	if self.ValidateType("map") {
		for itemName, data := range self.Data.(ard.Map) {
			var ctxt *Context
			if postProcess {
				ctxt = self
			}
			item, uid, isinprogok := isReadInProgress(data, rfield, ctxt, !postProcess)
			*uids = append(*uids, uid)
			if item == nil && !isinprogok {
				item = read(self.MapChild(itemName, data))
				process(item)
				postReadProcess(uid, item)
			} else if item != nil {
				process(item)
			}
		}
		return true
	}
	return false
}

// ReadListItemsDb - reads list items from db
func (self *Context) ReadListItemsDb(read Reader, process Processor, uids *[]string, rfield *ReadField, postProcess bool) bool {
	if self.ValidateType("list") {
		for index, data := range self.Data.(ard.List) {
			item, uid, isinprogok := isReadInProgress(data, rfield, nil, !postProcess)
			*uids = append(*uids, uid)
			if item == nil && !isinprogok {
				item = read(self.ListChild(index, data))
				process(item)
				postReadProcess(uid, item)
			} else if item != nil {
				process(item)
			}
		}
		return true
	}
	return false
}

func (self *Context) ReadSequencedListItemsDb(read Reader, process Processor, uids *[]string, rfield *ReadField, postProcess bool) bool {
	if self.ValidateType("list") {
		for index, data := range self.Data.(ard.List) {
			if !reflection.IsMap(data) {
				self.ReportFieldMalformedSequencedList()
				return false
			}
			item := data.(ard.Map)
			if len(item) != 1 {
				self.ReportFieldMalformedSequencedList()
				return false
			}
			for itemName, data := range item {
				ritem, uid, isinprogok := isReadInProgress(data, rfield, nil, !postProcess)
				*uids = append(*uids, uid)
				if ritem == nil && !isinprogok {
					ritem = read(self.SequencedListChild(index, itemName, data))
					process(ritem)
					postReadProcess(uid, ritem)
				} else if ritem != nil {
					process(ritem)
				}

			}
		}
		return true
	}
	return false
}

func isReadInProgress(childData interface{}, self *ReadField, ctxt *Context, isList bool) (interface{}, string, bool) {
	var item interface{}
	var cuid string
	inprogok := false
	cd, ok := childData.(ard.Map)
	if ok {
		cuid, ok = cd["uid"].(string)
		if ok {
			item, ok = visitedNodes[cuid]
			if ok {
				log.Debugf("\n *** %s/%s read previously!!!", cuid, self.FieldName)

			} else {
				_, inprogok = inprogressCompFields[cuid]
				if inprogok {
					if self.FieldName == "TargetNodeType" {
						ind := 0
						ind++
					}
					lst, inwaitok := inwaitCompFields[cuid]
					if !inwaitok {
						lst = make(ard.List, 0)
					}
					//field := self.Entity.FieldByName(self.FieldName)
					shldr := SliceHolder{
						Sfield:   self,
						Scontext: ctxt,
						IsList:   isList,
					}
					lst = append(lst, shldr)
					inwaitCompFields[cuid] = lst
				} else {
					inprogressCompFields[cuid] = true
				}
			}
		}
	}

	return item, cuid, inprogok
}

func readNode(self *ReadField, childData interface{}, key string, cuid string) interface{} {
	var item interface{}

	self.Context.Data.(ard.Map)[key] = childData
	if cuid != "" {
		inprogressCompFields[cuid] = true
	}
	if self.ReaderName == "DataType" ||
		self.ReaderName == "PolicyType" ||
		self.ReaderName == "CapabilityType" ||
		self.ReaderName == "ArtifactType" ||
		self.ReaderName == "GroupType" ||
		self.ReaderName == "InterfaceType" ||
		self.ReaderName == "RelationshipType" ||
		self.ReaderName == "NodeType" {
		name := childData.(ard.Map)["name"].(string)
		if self.ReaderName == "NodeType" && name == "cci.nodes.Network" {
			ind := 0
			ind++
		}
		item = self.Reader(self.Context.FieldChild(name, childData))
		if name == "cci.nodes.Network" {
			ind := 0
			ind++
		}
	} else {
		item = self.Reader(self.Context.FieldChild(key, childData))
	}
	postReadProcess(cuid, item)

	return item
}

func postReadProcess(cuid string, item interface{}) {
	if cuid != "" {
		visitedNodes[cuid] = item
		inprogressCompFields[cuid] = false
		delete(inprogressCompFields, cuid)
		lst, inwaitok := inwaitCompFields[cuid]
		if inwaitok {
			for _, hldr := range lst {
				shldr, ok := hldr.(SliceHolder)
				if ok {
					rfield := shldr.Sfield
					field := rfield.Entity.FieldByName(rfield.FieldName)
					if shldr.IsList {
						field.Set(reflect.Append(field, reflect.ValueOf(item)))
					} else if shldr.Scontext == nil {
						field.Set((reflect.ValueOf(item)))
					} else {
						shldr.Scontext.setMapItem(field, item)
					}

				}
			}
			delete(inwaitCompFields, cuid)

		}
	}

}
func postReadProcessListItem(cuid string, item interface{}) {
	if cuid != "" {
		visitedNodes[cuid] = item
		inprogressCompFields[cuid] = false
		delete(inprogressCompFields, cuid)
	}

}
func postReadProcessForList(cuids *[]string, slice *reflect.Value) {
	for _, cuid := range *cuids {
		lst, inwaitok := inwaitCompFields[cuid]
		if inwaitok {
			for _, hldr := range lst {
				shldr, ok := hldr.(SliceHolder)
				if ok {
					rfield := shldr.Sfield
					field := rfield.Entity.FieldByName(rfield.FieldName)
					field.Set(*slice)
				}
			}
			delete(inwaitCompFields, cuid)

		}
	}
}

func IsType(data interface{}, typeNames ...string) bool {
	valid := false
	for _, typeName := range typeNames {
		typeValidator, ok := PrimitiveTypeValidators[typeName]
		if !ok {
			panic(fmt.Sprintf("unsupported field type: %s", typeName))
		}
		if typeValidator(data) {
			valid = true
			break
		}
	}
	return valid
}

func convertArrayMapToSequencedList(childData interface{}, readerName string) []interface{} {
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

func parseArguments(args string) (ard.List, error) {
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

func parseMapArguments(args string) (ard.Map, error) {
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
func parseScalarUnitArguments(args string) (ard.List, error) {
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

// transformConditionData - transforms condition data in dgraph to what is expected by Puccini
func transformConditionData(childData interface{}) interface{} {
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
			argList, err := parseArguments(args)
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

// transformValueData - transforms value data in dgraph to what is expected by Puccini
func transformValueData(childData interface{}) interface{} {
	var ok bool
	var fname string
	var vdata ard.Map
	fname, ok = childData.(ard.Map)["functionname"].(string)
	if ok {
		vdata = make(ard.Map)
		vdata[fname] = childData.(ard.Map)["fnarguments"]
		return vdata
	}

	return childData
}

func readFromDb(readField *ReadField, uid string, key string, isMap bool) interface{} {

	var childData interface{}

	if dbObject, ok := dbObjectMap[readField.ReaderName]; ok {
		compMap := dbObject.DbRead(readField.Context.Dgt, nil, uid, key)
		v1 := compMap["comp"].([]interface{})
		v2 := v1[0].(map[string]interface{})
		childData = v2[key]
		switch childData.(type) {
		case ard.Map:
			childData.(ard.Map)["readername"] = readField.ReaderName
		case []interface{}:
			cda := childData.([]interface{})
			if isMap {
				childData = dbObject.ConvertMap(&cda, key, readField.ReaderName)
			} else {
				childData = dbObject.ConvertSlice(&cda, &readField.Context.Data, readField.ReaderName)
			}
		}
	} else {
		common.FailOnError(errors.New("*** DbObject for [" + readField.ReaderName + "] is not defined!"))
	}

	return childData
}
func readItemFromDb(readField *ReadField, uid string, key string, cData *interface{}) interface{} {

	var childData interface{}

	if dbObject, ok := dbObjectMap[readField.ReaderName]; ok {

		childData = readField.Context.Data
		if dbObject.ByPassDbRead(&childData, readField.Context.Name, key) {
			childData = readField.Context.Data.(ard.Map)[key]
		} else {
			compMap := dbObject.DbRead(readField.Context.Dgt, cData, uid, key)
			v1 := compMap["comp"].([]interface{})
			v2 := v1[0].(map[string]interface{})
			childData = v2[key]
			childData = dbObject.Convert(&childData, readField.ReaderName)
		}
	} else {
		common.FailOnError(errors.New("*** DbObject for [" + readField.ReaderName + "] is not defined!"))
	}

	return childData
}
