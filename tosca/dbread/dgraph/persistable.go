package dgraph

import (
	"reflect"

	"github.com/op/go-logging"
	"github.com/tliron/puccini/ard"
)

var Log = logging.MustGetLogger("dgraph")

type SearchFields struct {
	ObjectKey    string
	ObjectDGType string // Dgraph Object Type
	ObjectNSuid  string // namespace uid
	SubjectUid   string
	Predicate    string
}

type SaveFields struct {
	EntityPtr  interface{}
	Name       string
	DgType     string
	Nurl       string
	Nsuid      string
	SubjectUid string
	Predicate  string
}

// Persistable interface for objects that can be saved to and retrieved from the Dgraph database
type Persistable interface {
	Convert(responseData *interface{}, readerName string) interface{}
	ConvertMap(responseData *[]interface{}, key string, readername string) interface{}
	ConvertSlice(responseData *[]interface{}, contextData *interface{}, readername string) interface{}
	DbRead(dgt *DgraphTemplate, fieldData *interface{}, uid string, key string) ard.Map
	ByPassDbRead(contextData *interface{}, name string, key string) bool
	//RequiresConversion(key string)bool
	DbFind(dgt *DgraphTemplate, searchObject interface{}) (bool, string, error)
	DbBuildInsertQuery(dataObject interface{}) (string, error)
	DbInsert(dgt *DgraphTemplate, mutateQuery string) (string, error)
}

type DbObjectMap map[string]Persistable
type DbGrammarMap map[string]DbObjectMap

// interfaces and registry for methods referencing tosca package in dbread
// and underlying packages
type ToscaMethod interface {
	Process(entityPtr *interface{}, nam string) string
}

var QueryAttributeDefaultKey string = "attribute.default"
var QueryValueKey string = "value.key"
var ToscaMethodRegistry map[string]ToscaMethod = make(map[string]ToscaMethod)

// interfaces and registry for methods to build query for a field
type FieldQuery interface {
	Query(field reflect.Value) string
}

type FieldRegistryHolder struct {

	// Registry for queries specific to a particular field
	QueryRegistry map[string]FieldQuery

	// registry for methods to execute a tosca method
	// key is the field name
	// value is the key to the ToscaMethodRegistry
	ToscaMethodRegistry map[string]string

	// registry for methods to merge fields in embedded structs
	// key is a composite of dgtype and field name
	// value is true or false
	MergeRegistry map[string]bool

	// registry for fields that may contain array values
	// in the form "[val1 val2 val3]"
	// key is a composite of field name
	// value is true or false
	ArrayRegistry map[string]bool
}

var FieldRegistries FieldRegistryHolder = FieldRegistryHolder{
	QueryRegistry:       make(map[string]FieldQuery),
	ToscaMethodRegistry: make(map[string]string),
	MergeRegistry:       make(map[string]bool),
	ArrayRegistry:       make(map[string]bool),
}

//var FieldQueryRegistry map[string]FieldQuery = make(map[string]FieldQuery)

// registry for methods to execute a tosca method
// key is a composite of dgtype and field name
// value is the key into the ToscaMethodRegistry
//var FieldToscaMethodRegistry map[string]string = make(map[string]string)

// registry for methods to merge fields in embedded structs
// key is a composite of dgtype and field name
// value is true or false
//var FieldMergeRegistry map[string]bool = make(map[string]bool)

// registry for fields that may contain array values
// in the form "[val1 val2 val3]"
//var FieldArrayRegistry map[string]bool = make(map[string]bool)
