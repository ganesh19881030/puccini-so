package reflection

import (
	"reflect"
)

// Ignore fields tagged with "traverse:ignore" or "lookup"
func DbTraverse(entityPtr interface{}, hentityMap map[interface{}]bool, traverse Traverser) {
	if !traverse(entityPtr) {
		return
	}

	if !IsPtrToStruct(reflect.TypeOf(entityPtr)) {
		return
	}

	value := reflect.ValueOf(entityPtr).Elem()

	for _, structField := range GetStructFields(value.Type()) {
		// Has traverse:"ignore" tag?
		traverseTag, ok := structField.Tag.Lookup("traverse")
		if ok && (traverseTag == "ignore") {
			continue
		}

		// Has "lookup" tag?
		//if _, ok = structField.Tag.Lookup("lookup"); ok {
		//	continue
		//}

		field := value.FieldByName(structField.Name)
		fieldType := field.Type()
		if IsPtrToStruct(fieldType) && !field.IsNil() {
			// Compatible with *interface{}
			DbTraverse(field.Interface(), hentityMap, traverse)
		} else if IsSliceOfPtrToStruct(fieldType) {
			// Compatible with []*interface{}
			length := field.Len()
			for i := 0; i < length; i++ {
				element := field.Index(i)
				DbTraverse(element.Interface(), hentityMap, traverse)
			}
		} else if IsMapOfStringToPtrToStruct(fieldType) {
			// Compatible with map[string]*interface{}
			for _, mapKey := range field.MapKeys() {
				element := field.MapIndex(mapKey)
				DbTraverse(element.Interface(), hentityMap, traverse)
			}
		}
	}
}
