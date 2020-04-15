package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"sort"
	"strings"
	"sync"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/clout"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/tosca/compiler"
	"github.com/tliron/puccini/tosca/dbread"
	"github.com/tliron/puccini/tosca/dbread/dgraph"
	"github.com/tliron/puccini/tosca/normal"

	"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/database"
	"github.com/tliron/puccini/tosca/parser"
	"github.com/tliron/puccini/tosca/reflection"
	"github.com/tliron/puccini/url"
)

var quirks []string
var cache sync.Map // entityPtr or Promise

// DgContext structure to hold the parser context
type DgContext struct {
	Pcontext parser.Context
}

//var dgraphTypes *dgraph.DgraphTypes
var inputs []string
var inputsUrl string
var stopAtPhase uint32
var dumpPhases []uint
var filter string
var ardFormat string
var pretty bool
var inputValues = make(map[string]interface{})

var grammarVersion string

var dbObjectMap dgraph.DbObjectMap

func init() {
	dbObjectMap = dbread.DbGrammar["dgraph.tosca_v1_3"]
}

// ReadServiceTemplateFromDgraph reads servicetemplate from dgraph
func (dbc *DgContext) ReadServiceTemplateFromDgraph(sturl url.URL, inps []string, inpsUrl string) (*normal.ServiceTemplate, bool) {

	var err error
	inputs = inps
	inputsUrl = inpsUrl
	ParseInputs()

	//var dbc parser.Context
	dbc.Pcontext = parser.NewContext(quirks)
	toscaContext := tosca.NewContext(&dbc.Pcontext.Problems, dbc.Pcontext.Quirks)
	toscaContext.URL = sturl
	toscaContext.ReadFromDb = true

	dburl := fmt.Sprintf("%s:%d", common.SoConfig.Dgraph.Host, common.SoConfig.Dgraph.Port)

	dgraphClient, conn, err := database.GetDgraphClient1(dburl)
	common.FailOnError(err)
	defer conn.Close()
	ctx := context.Background()
	dgt := new(dgraph.DgraphTemplate)
	dgt.Ctxt = ctx
	dgt.Client = dgraphClient

	toscaContext.Dgt = dgt

	stname := database.ExtractTopologyName(sturl.String())

	serviceTemplate, ok := dbc.read(nil, toscaContext, "", "service_template", stname, dgt)
	if serviceTemplate == nil {
		common.FailOnError(errors.New("Service template [" + stname + "] not found!"))
	}

	dbc.Pcontext.ServiceTemplate = serviceTemplate
	sort.Sort(dbc.Pcontext.Units)

	for key := range parser.GrammerVersions {
		dbc.Pcontext.GrammerVersions = append(dbc.Pcontext.GrammerVersions, key)
	}
	stopAtPhase = 10
	filter = ""
	ardFormat = "yaml"
	pretty = true

	// Phase 2: Namespaces
	if stopAtPhase >= 2 {
		dbc.Pcontext.AddNamespaces()
		dbc.Pcontext.LookupNames()
		if !common.Quiet && ToPrintPhase(2) {
			if len(dumpPhases) > 1 {
				fmt.Fprintf(format.Stdout, "%s\n", format.ColorHeading("Namespaces"))
			}
			dbc.Pcontext.PrintNamespaces(1)
		}
	}
	mergeToscaScriptNamespace(toscaContext)

	if stopAtPhase >= 3 {
		dbc.Pcontext.AddHierarchies()
		if !common.Quiet && ToPrintPhase(3) {
			if len(dumpPhases) > 1 {
				fmt.Fprintf(format.Stdout, "%s\n", format.ColorHeading("Hierarchies"))
			}
			dbc.Pcontext.PrintHierarchies(1)
		}
	}

	// Phase 4: Inheritance
	if stopAtPhase >= 4 {
		tasks := dbc.Pcontext.GetInheritTasks()
		if !common.Quiet && ToPrintPhase(4) {
			if len(dumpPhases) > 1 {
				fmt.Fprintf(format.Stdout, "%s\n", format.ColorHeading("Inheritance Tasks"))
			}
			tasks.Print(1)
		}
		tasks.Drain()
	}

	parser.SetInputs(dbc.Pcontext.ServiceTemplate.EntityPtr, inputValues)

	// Phase 5: Rendering
	if stopAtPhase >= 5 {
		entityPtrs := dbc.Pcontext.Render()
		if !common.Quiet && ToPrintPhase(5) {
			sort.Sort(entityPtrs)
			if len(dumpPhases) > 1 {
				fmt.Fprintf(format.Stdout, "%s\n", format.ColorHeading("Rendering"))
			}
			for _, entityPtr := range entityPtrs {
				fmt.Fprintf(format.Stdout, "%s:\n", format.ColorPath(tosca.GetContext(entityPtr).Path.String()))
				err = format.Print(entityPtr, ardFormat, pretty)
				common.FailOnError(err)
			}
		}
	}

	if filter != "" {
		entityPtrs := dbc.Pcontext.Gather(filter)
		if len(entityPtrs) == 0 {
			common.Failf("No paths found matching filter: \"%s\"\n", filter)
		} else if !common.Quiet {
			for _, entityPtr := range entityPtrs {
				fmt.Fprintf(format.Stdout, "%s\n", format.ColorPath(tosca.GetContext(entityPtr).Path.String()))
				err = format.Print(entityPtr, ardFormat, pretty)
				common.FailOnError(err)
			}
		}
	}

	if !common.Quiet {
		dbc.Pcontext.Problems.Print()
	}

	if !dbc.Pcontext.Problems.Empty() {
		os.Exit(1)
	}

	// Normalize
	s, ok := parser.Normalize(dbc.Pcontext.ServiceTemplate.EntityPtr)
	if !ok {
		common.Fail("grammar does not support normalization")
	}

	return s, ok
}

func (dbc *DgContext) read(promise parser.Promise, toscaContext *tosca.Context, uid string, key string, name string, dgt *dgraph.DgraphTemplate) (*parser.Unit, bool) {
	//defer dbc.WG.Done()
	if promise != nil {
		// For the goroutines waiting for our cached entityPtr
		defer promise.Release()
	}

	//readerName := dgraphTypes.TypeMap[key]
	readerName := "ServiceTemplate"

	//log.Infof("{read} %s:t%s", readerName, toscaContext.URL.Key())

	// Read ARD
	var err error

	urlstring := strings.Replace(toscaContext.URL.String(), "\\", "/", -1)
	if stObject, ok := dbObjectMap[readerName]; ok {
		toscaContext.Data = stObject.DbRead(dgt, nil, name, urlstring)
		if toscaContext.Data == nil {
			return nil, false
		} else {
			grammarVersion = (toscaContext.Data).(ard.Map)["tosca_definitions_version"].(string)
		}
	} else {
		common.FailOnError(errors.New("No object defined for " + readerName))
	}

	if err != nil {
		toscaContext.ReportError(err)
		return nil, false
	}

	// Detect grammar
	if !parser.DetectGrammar(toscaContext) {
		return nil, false
	}

	// Read entityPtr
	read, ok := toscaContext.Grammar[readerName]
	if !ok {
		panic(fmt.Sprintf("grammar does not support reader \"%s\"", readerName))
	}
	entityPtr := read(toscaContext)
	if entityPtr == nil {
		// Even if there are problems, the reader should return an entityPtr
		panic(fmt.Sprintf("reader \"%s\" returned a non-entity: %T", reflection.GetFunctionName(read), entityPtr))
	}

	// Validate required fields
	reflection.Traverse(entityPtr, tosca.ValidateRequiredFields)

	unit := parser.NewUnit(entityPtr, nil, nil)
	dbc.Pcontext.AddUnit(unit)

	return unit, true
}

func ToPrintPhase(phase uint) bool {
	for _, p := range dumpPhases {
		if p == phase {
			return true
		}
	}
	return false
}

func mergeToscaScriptNamespace(toscaContext *tosca.Context) {
	var grammar tosca.Grammar
	toscaDef1 := parser.Grammars["tosca_definitions_version"]
	grammar = toscaDef1[grammarVersion]

	paths := make([]string, 0)

	toscaDef := parser.InternalProfilePaths["tosca_definitions_version"]
	toscaPath := toscaDef[grammarVersion]
	paths = append(paths, toscaPath)

	for _, path := range paths {
		if profileURL, err := url.NewValidInternalURL(path); err == nil {
			data, _, _ := ard.ReadURL(profileURL, true)
			toscaContext.URL = profileURL
			toscaContext.Data = data["metadata"]
			reader := grammar["Metadata"]
			reader(toscaContext)
		}

	}
}

// Compile - Compiles ServiceTemplate into Clout structure
func (dbc *DgContext) Compile(st *normal.ServiceTemplate, sturl url.URL, resolve bool, coerce bool, output string) (*clout.Clout, error) {

	clout, err := compiler.Compile(st)
	common.FailOnError(err)

	//resolve = false

	// Resolve
	if resolve {
		compiler.Resolve(clout, &dbc.Pcontext.Problems, ardFormat, pretty)
		if !dbc.Pcontext.Problems.Empty() {
			if !common.Quiet {
				dbc.Pcontext.Problems.Print()
			}
			os.Exit(1)
		}
	}

	// Coerce
	if coerce {
		compiler.Coerce(clout, &dbc.Pcontext.Problems, ardFormat, pretty)
		if !dbc.Pcontext.Problems.Empty() {
			if !common.Quiet {
				dbc.Pcontext.Problems.Print()
			}
			os.Exit(1)
		}
	}

	// turned it off for now as the clout is not being saved properly
	persist := false
	if persist {
		internalImport := common.InternalImport
		urlString := strings.Replace(sturl.String(), "\\", "/", -1)
		database.Persist(clout, st, urlString, dbc.Pcontext.GrammerVersions, internalImport)
	}

	if !common.Quiet || (output != "") {
		err = format.WriteOrPrint(clout, ardFormat, pretty, output)
		common.FailOnError(err)
	}

	return clout, err
}
func ParseInputs() {
	if inputsUrl != "" {
		//log.Infof("load inputs from %s", inputsUrl)
		url_, err := url.NewValidURL(inputsUrl, nil)
		common.FailOnError(err)
		reader, err := url_.Open()
		common.FailOnError(err)
		if readerCloser, ok := reader.(io.ReadCloser); ok {
			defer readerCloser.Close()
		}
		data, err := format.Read(reader, "yaml")
		common.FailOnError(err)
		if map_, ok := data.(ard.Map); ok {
			for key, value := range map_ {
				inputValues[key] = value
			}
		} else {
			common.Failf("malformed inputs in %s", inputsUrl)
		}
	}

	for _, input := range inputs {
		s := strings.SplitN(input, "=", 2)
		if len(s) != 2 {
			common.Failf("malformed input: %s", input)
		}
		value, err := format.Decode(s[1], "yaml")
		common.FailOnError(err)
		inputValues[s[0]] = value
	}
}
