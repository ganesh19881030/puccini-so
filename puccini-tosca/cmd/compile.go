package cmd

import (
	//"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"

	//"github.com/tliron/puccini/tosca"
	"github.com/tliron/puccini/tosca/compiler"
	"github.com/tliron/puccini/tosca/database"
	//"github.com/tliron/puccini/tosca/parser"
)

var output string
var resolve bool
var coerce bool
var persist bool

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringArrayVarP(&inputs, "input", "i", []string{}, "specify an input (name=YAML)")
	compileCmd.Flags().StringVarP(&inputsUrl, "inputs", "n", "", "load inputs from a PATH or URL")
	compileCmd.Flags().StringVarP(&output, "output", "o", "", "output Clout to file (default is stdout)")
	compileCmd.Flags().BoolVarP(&resolve, "resolve", "r", true, "resolves the topology (attempts to satisfy all requirements with capabilities")
	compileCmd.Flags().BoolVarP(&coerce, "coerce", "c", false, "coerces all values (calls functions and applies constraints)")
	compileCmd.Flags().BoolVarP(&persist, "persist", "g", false, "persists clout output into a dgraph database")
}

var compileCmd = &cobra.Command{
	Use:   "compile [[TOSCA PATH or URL]]",
	Short: "Compile TOSCA to Clout",
	Long:  `Parses a TOSCA service template and compiles the normalized output of the parser to Clout. Supports JavaScript plugins.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var urlString string
		if len(args) == 1 {
			urlString = args[0]
		}

		Compile(urlString)
	},
}

func Compile(urlString string) {

	// read configuration from a file
	common.ReadConfiguration()

	// Parse
	context, s := Parse(urlString)
	internalImport := common.InternalImport
	//fmt.Println(internalImport)
	// Compile
	clout, err := compiler.Compile(s)
	common.FailOnError(err)

	// Resolve
	if resolve {
		compiler.Resolve(clout, &context.Problems, ardFormat, pretty)
		if !context.Problems.Empty() {
			if !common.Quiet {
				context.Problems.Print()
			}
			os.Exit(1)
		}
	}

	// Coerce
	if coerce {
		compiler.Coerce(clout, &context.Problems, ardFormat, pretty)
		if !context.Problems.Empty() {
			if !common.Quiet {
				context.Problems.Print()
			}
			os.Exit(1)
		}
	}

	// Persist
	persist = false
	if persist {
		//grammarVersion := reflect.ValueOf(context.ServiceTemplate.EntityPtr).Type().String()

		//grammarVersion = grammarVersion[1:strings.LastIndex(grammarVersion, ".")]

		database.Persist(clout, s, urlString, context.GrammerVersions, internalImport)
	}

	if !common.Quiet || (output != "") {
		err = format.WriteOrPrint(clout, ardFormat, pretty, output)
		common.FailOnError(err)
	}
}
