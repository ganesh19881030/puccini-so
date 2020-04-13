package database

import (
	"reflect"
	"strings"

	"github.com/tliron/puccini/tosca/parser"
	"github.com/tliron/puccini/tosca/reflection"
)

// Traverse2 - traverse model entity tree
func Traverse2(entityPtr interface{}, traverse DbTraverser, bag TravelBag, pst *parser.Unit) {
	if !traverse(entityPtr, &bag) {
		return
	}

	if !reflection.IsPtrToStruct(reflect.TypeOf(entityPtr)) {
		return
	}

	bag.Level++
	value := reflect.ValueOf(entityPtr).Elem()

	num := value.NumField()
	unitInd := -1
	var inds []int
	var name string
	var adeptr interface{}
	for ind := 0; ind < num; ind++ {
		name = value.Type().Field(ind).Name
		if name == "Unit" {
		} else if name == "Entity" {
		} else if (name == "AttributeDefinition" && bag.ChildDgraphType == "PropertyDefinition") ||
			(name == "PropertyDefinition" && bag.ChildDgraphType == "ParameterDefinition") ||
			(name == "ArtifactDefinition" && bag.ChildDgraphType == "Artifact") {
			field := value.Field(ind)
			adeptr = field.Interface()
		} else if name == "Type" {
		} else {
			inds = append(inds, ind)
		}
	}
	if unitInd > -1 {
		inds = append(inds, unitInd)
	}

	for _, ind := range inds {
		field := value.Field(ind)
		name = value.Type().Field(ind).Name
		processField(field, name, traverse, bag, pst)
	}
	// fetch all fields from AttributeDefinition when handling PropertyDefinition
	if adeptr != nil {
		mergeFields(adeptr, traverse, bag, pst)
	}

}

func processField(field reflect.Value, name string, traverse DbTraverser, bag TravelBag, pst *parser.Unit) {
	fieldType := field.Type()
	bag.Predicate = strings.ToLower(name)
	if reflection.IsPtrToStruct(fieldType) && !field.IsNil() {
		Log.Debugf("\nLevel: %d, Traversing from %s/%s to %s\n", bag.Level, bag.ChildDgraphType, bag.ChildName, name)
		Traverse2(field.Interface(), traverse, bag, pst)
	} else if reflection.IsSliceOfPtrToStruct(fieldType) {
		// Compatible with []*interface{}
		length := field.Len()
		if length > 0 {
			Log.Debugf("Level: %d, Traversing slice %s length=%d\n", bag.Level, name, length)
		}
		for i := 0; i < length; i++ {
			element := field.Index(i)
			Traverse2(element.Interface(), traverse, bag, pst)
		}
	} else if reflection.IsMapOfStringToPtrToStruct(fieldType) {
		// Compatible with map[string]*interface{}
		if len(field.MapKeys()) > 0 {
			Log.Debugf("Level: %d, Traversing map %s length=%d\n", bag.Level, name, len(field.MapKeys()))
		}
		for _, mapKey := range field.MapKeys() {
			element := field.MapIndex(mapKey)
			bag.Mapkey = mapKey.String()
			Traverse2(element.Interface(), traverse, bag, pst)
		}
	}

}

func mergeFields(adeptr interface{}, traverse DbTraverser, bag TravelBag, pst *parser.Unit) {
	value := reflect.ValueOf(adeptr).Elem()
	num := value.NumField()
	for ind := 0; ind < num; ind++ {
		name := value.Type().Field(ind).Name
		field := value.Field(ind)
		if name == "AttributeDefinition" {
			mergeFields(field.Interface(), traverse, bag, pst)
		} else if name != "Unit" && name != "Entity" {
			processField(field, name, traverse, bag, pst)
		}
	}

}
