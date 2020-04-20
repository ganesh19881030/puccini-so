package reflection

import (
	"reflect"
)

func GetTaggedFields(entityPtr interface{}, name string) []reflect.Value {
	var fields []reflect.Value
	entity := reflect.ValueOf(entityPtr).Elem()
	for fieldName := range GetFieldTagsForValue(entity, name) {
		field := entity.FieldByName(fieldName)
		fields = append(fields, field)
	}
	return fields
}

func GetFieldTagsForValue(value reflect.Value, name string) map[string]string {
	return GetFieldTagsForType(value.Type(), name)
}

func GetFieldTagsForType(type_ reflect.Type, name string) map[string]string {
	tags := make(map[string]string)
	for _, structField := range GetStructFields(type_) {
		if value, ok := structField.Tag.Lookup(name); ok {
			tags[structField.Name] = value
		}
	}
	return tags
}

func GetAllFieldTagsForValue(value reflect.Value) map[string]string {
	return GetAllFieldTagsForType(value.Type())
}

func GetAllFieldTagsForType(type_ reflect.Type) map[string]string {
	tags := make(map[string]string)
	for _, structField := range GetStructFields(type_) {
		value, ok := structField.Tag.Lookup("read")
		if !ok {
			value, ok = structField.Tag.Lookup("lookup")
		}
		//if !ok {
		//	value, ok = structField.Tag.Lookup("namespace")
		//}
		if ok {
			tags[structField.Name] = value
		}
	}
	return tags
}
