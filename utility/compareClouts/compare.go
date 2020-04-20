package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/tliron/puccini/ard"
	"github.com/tliron/puccini/common"
	"github.com/tliron/puccini/format"
	"github.com/tliron/puccini/puccini-js/cmd"
)

// Here "-log n" is optional and specifies the log level:
// 0 - ERROR
// 1 - INFO
// 2 - DEBUG
const COMPARE_USAGE = "Usage: go run compare.go logging.go file1.json file2.json [-log n]"

func main() {
	args := os.Args
	var path1 string
	var path2 string
	var errs1 []string
	var errs2 []string
	var logLevel int

	logLevel = INFO
	if len(args) < 3 {
		log.Errorf("Did not specify clout file names to compare!")
		log.Errorf(COMPARE_USAGE)
		os.Exit(1)
	} else if len(args) >= 3 {
		path1 = args[1]
		path2 = args[2]
		if len(args) > 3 {
			logoption := args[3]
			if logoption != "-log" && logoption != "-l" {
				log.Errorf("Invalid log level option specified!")
				log.Errorf(COMPARE_USAGE)
				os.Exit(1)
			}
			if len(args) > 4 {
				level, err := strconv.Atoi(args[4])
				common.FailOnError(err)
				logLevel = level
			} else {
				log.Errorf("Did not provide log level - usually between 0 to 2!")
				log.Errorf(COMPARE_USAGE)
				os.Exit(1)
			}
		}
	}

	//logLevel := INFO
	//logLevel := DEBUG
	logFileName := "compare-clout.log"
	// remove existing log file
	if fileExists(logFileName) {
		os.Remove(logFileName)
	}

	ConfigureLogging(logLevel, &logFileName, true)

	// added this to compare clout files in yaml format
	path1 = getJSONPath(path1)
	path2 = getJSONPath(path2)

	clout1 := getJSONObject(path1)
	clout2 := getJSONObject(path2)

	if clout1 == nil {
		common.Failf("Invalid file format for path : %v", path1)
	}

	if clout2 == nil {
		common.Failf("Invalid file format for path : %v", path2)
	}

	//get names of clout files from their path
	fileName1 := getNameOfCloutFile(path1)
	fileName2 := getNameOfCloutFile(path2)

	//compare metadata, properties and version of clout and vertex's edgesOut
	compareVersionMetadataAndProperties(clout1, clout2, fileName1, fileName2, &errs1)

	//compare count of vertexes based on their types
	compareCountOfVertexTypes(clout1, clout2, fileName1, fileName2, &errs1)

	//compare vertexes of clout1 with clout2
	compareVertexes(clout1, clout2, fileName1, fileName2, &errs1)

	//compare vertexes of clout2 with clout1
	compareVertexes(clout2, clout1, fileName2, fileName1, &errs2)

	//print errors on console
	if len(errs1) > 0 || len(errs2) > 0 {
		log.Errorf("")
		log.Errorf("Comparing [%s] with [%s] :", fileName1, fileName2)
		log.Errorf("------------------------------------------------")
		for _, err := range errs1 {
			log.Errorf(err)
		}
		log.Errorf("")
		log.Errorf("****************************************************")
		log.Errorf("")
		log.Errorf("Comparing [%s] with [%s] :", fileName2, fileName1)
		log.Errorf("------------------------------------------------")
		for _, err := range errs2 {
			log.Errorf(err)
		}
		log.Errorf("")
		log.Errorf("****************************************************")
		log.Errorf("")
		log.Errorf("Files [%s] and [%s] do not match!", fileName1, fileName2)
		//common.Failf("")
	} else {
		log.Infof("")
		log.Infof("Files [%s] and [%s] match!", fileName1, fileName2)
	}
}

//this method compares following:
//	. version, metadata and properties of clout
//	. metadata and properties of vertex's edgesOuts
func compareVersionMetadataAndProperties(clout1 ard.Map, clout2 ard.Map,
	fileName1 string, fileName2 string, errs *[]string) bool {
	arr := [2]string{"metadata", "properties"}
	var match bool = true

	//compare version of clout
	versionOfClout1, _ := clout1["version"].(interface{})
	versionOfClout2, _ := clout2["version"].(interface{})
	path := ""
	if !equal(versionOfClout1, versionOfClout2, nil, nil, path) {
		if errs != nil {
			*errs = append(*errs, "Clout version in "+fileName1+" and "+fileName2+" did not match")
		}
		match = false
	}

	//compare metadata and properties of clout
	for _, data := range arr {
		dataOfClout1, _ := clout1[data].(ard.Map)
		dataOfClout2, _ := clout2[data].(ard.Map)
		path := ""
		if !equal(dataOfClout1, dataOfClout2, nil, nil, path) {
			if errs != nil {
				*errs = append(*errs, "Clout "+data+" in "+fileName1+" and "+fileName2+" did not match")
			}
			match = false
		}
	}

	return match
}

//compare count of vertexes and their types in both clout files
func compareCountOfVertexTypes(clout1 ard.Map, clout2 ard.Map, fileName1 string, fileName2 string,
	errs *[]string) {
	arr := [8]string{"nodeTemplate", "workflowStep", "workflowActivity", "workflow", "policy", "operation", "substitution", "group"}
	vertexesOfClout1, _ := clout1["vertexes"].(ard.Map)
	vertexesOfClout2, _ := clout2["vertexes"].(ard.Map)

	//check for equal number of vertexes in both clout files
	countOfVertexesFromClout1 := countTotalNumberOfVertexes(vertexesOfClout1)
	countOfVertexesFromClout2 := countTotalNumberOfVertexes(vertexesOfClout2)

	if countOfVertexesFromClout1 != countOfVertexesFromClout2 {
		countFromClout1 := strconv.Itoa(countOfVertexesFromClout1)
		countFromClout2 := strconv.Itoa(countOfVertexesFromClout2)
		*errs = append(*errs, "Mismatch in number of vertexes. "+fileName1+": "+countFromClout1+" vertexes, "+fileName2+": "+countFromClout2+" vertexes")
	}

	//check for equal number of vertexes based on vertex types in both clout files
	for _, vertexTypeName := range arr {
		countVertexTypeFromClout1 := countVertexesOfSpecificType(vertexTypeName, vertexesOfClout1)
		countVertexTypeFromClout2 := countVertexesOfSpecificType(vertexTypeName, vertexesOfClout2)

		if countVertexTypeFromClout1 != countVertexTypeFromClout2 {
			countFromClout1 := strconv.Itoa(countVertexTypeFromClout1)
			countFromClout2 := strconv.Itoa(countVertexTypeFromClout2)
			*errs = append(*errs, "Mismatch in number of vertexes of type "+vertexTypeName+". "+fileName1+": "+countFromClout1+" vertexes, "+fileName2+": "+countFromClout2+" vertexes")
		}
	}
}

//compare vertexes based on their types
func compareVertexes(clout1 ard.Map, clout2 ard.Map, fileName1 string, fileName2 string,
	errs *[]string) {
	vertexesOfClout1, _ := clout1["vertexes"].(ard.Map)
	vertexesOfClout2, _ := clout2["vertexes"].(ard.Map)
	arr := [...]string{"nodeTemplate", "workflowStep", "policy", "group", "workflow", "policyTrigger"}
	var match bool
	var matchedVertex interface{}

	//compare vertexes of type nodeTemplate, workflowStep, policy, group, workflow and policyTrigger
	for _, vertexType := range arr {
		for _, vertex := range vertexesOfClout1 {
			if !checkVertexType(vertexType, vertex) {
				continue
			}
			//get vertex name from clout1 and search vertex with that name in clout2 and then compare that two vertexes
			vertexMap, _ := vertex.(ard.Map)
			clout1VertexProperties, _ := vertexMap["properties"].(ard.Map)
			vertexName, _ := clout1VertexProperties["name"].(interface{}).(string)

			//find vertex from their name in clout2
			for _, vertex := range vertexesOfClout2 {
				vertexPropertiesMap, _ := vertex.(ard.Map)
				vertexProperties, _ := vertexPropertiesMap["properties"].(ard.Map)
				name, _ := vertexProperties["name"]
				if (name != nil) && (name == vertexName) {
					matchedVertex = vertex
				}
			}

			if matchedVertex == nil {
				*errs = append(*errs, "Vertex of type '"+vertexType+"' with name '"+vertexName+"' in "+fileName1+" not found in "+fileName2+"")
				continue
			}
			log.Debugf("")
			log.Debugf("Comparing vertex : %s", vertexName)
			log.Debugf("----------------------------------")
			path := vertexType + "/" + vertexName
			//compare vertexes from both clouts
			if !equal(vertex, matchedVertex, vertexesOfClout1, vertexesOfClout2, path) {
				*errs = append(*errs, "Vertex of type '"+vertexType+"' with name '"+vertexName+"' in "+fileName1+" did not match with "+fileName2+"")
			} else {
				log.Debugf("!!!!------  Vertex %s Matched  -------!!!!", vertexName)
			}
		}
	}

	//compare vertex of type substitution from both clout files
	for _, vertexOfClout1 := range vertexesOfClout1 {
		match = false
		fnd := false
		if !checkVertexType("substitution", vertexOfClout1) {
			continue
		}
		vprops1 := vertexOfClout1.(ard.Map)["properties"].(ard.Map)
		subtype1 := vprops1["type"].(string)
		for _, vertexOfClout2 := range vertexesOfClout2 {
			if !checkVertexType("substitution", vertexOfClout2) {
				continue
			}
			vprops2 := vertexOfClout2.(ard.Map)["properties"].(ard.Map)
			subtype2 := vprops2["type"].(string)
			//if subtype1 == subtype2 && subtype1 == "cci.nodes.PacketSink" {
			//	log.Debugf("Comparing packet sink substitution")
			//}
			path := ""
			if subtype1 == subtype2 {
				fnd = true
				log.Debugf("")
				log.Debugf("Comparing substitution mapping type %s", subtype1)
				match = equal(vertexOfClout1, vertexOfClout2, vertexesOfClout1, vertexesOfClout2, path)
				if match {
					log.Debugf("!!!!------  Substitution mapping type %s Matched  -------!!!!", subtype1)
					break
				}
			}
		}
		if !match {
			vertexPropertiesMap, _ := vertexOfClout1.(ard.Map)
			vertexProperties, _ := vertexPropertiesMap["properties"].(ard.Map)
			stype, _ := vertexProperties["type"].(string)
			if !fnd {
				log.Debugf("         substitution mapping type %s not found in other clout!")
			} else {
				log.Debugf("         substitution mapping type %s failed to match!", subtype1)
			}
			log.Debugf("")

			*errs = append(*errs, "Vertex of type 'substitution' of type ["+stype+"] in "+fileName1+" did not match with "+fileName2+"")
		}
	}
}

//check for equality between two objects
func equal(vx, vy interface{}, vertexesOfClout1 ard.Map, vertexesOfClout2 ard.Map, path string) bool {
	path1 := path
	if reflect.TypeOf(vx) != reflect.TypeOf(vy) {
		log.Debugf("     failed because data types %s & %s do not match", reflect.TypeOf(vx), reflect.TypeOf(vy))
		log.Debugf("     path -> %s", path)
		return false
	}
	switch x := vx.(type) {
	case ard.Map:

		y := vy.(ard.Map)

		if len(x) != len(y) {
			log.Debugf("     failed while comparing map length:  %d vs %d", len(x), len(y))
			log.Debugf("     map x keys: %s", getMapKeys(x))
			log.Debugf("     map y keys: %s", getMapKeys(y))
			log.Debugf("     path -> %s", path)
			return false
		}

		for k, v := range x {
			path1 = path + "/" + k
			val2 := y[k]

			//ignore location, path, url and description in clout
			if ignoreFields(k) {
				continue
			}

			//compare edgesOut of vertexes
			if k == "edgesOut" {
				log.Debugf("comparing edgesOut...")
				if !compareEdgesOutOfVertexes(v, val2, vertexesOfClout1, vertexesOfClout2, path1) {
					log.Debugf("     failed while comparing edgesOut.")
					log.Debugf("     path -> %s", path)
					return false
				}
				continue
			}

			if k == "directives" {
				log.Debugf("comparing directives...")
				if !compareDirectivesOfVertexes(v, val2, vertexesOfClout1, vertexesOfClout2, path1) {
					log.Debugf("     failed while comparing directives.")
					log.Debugf("     path -> %s", path)
					return false
				}
				continue
			}

			if (v == nil) != (val2 == nil) {
				log.Debugf("     failed while comparing map values - one of them is nil.")
				log.Debugf("     path -> %s", path)
				return false
			}

			if k != "history" {
				log.Debugf("Comparing %s values...", k)
				if !equal(v, val2, vertexesOfClout1, vertexesOfClout2, path1) {
					return false
				}
			}
		}
		return true

	case ard.List:
		y := vy.(ard.List)

		if len(x) != len(y) {
			log.Debugf("     failed while comparing array length:  %d vs %d", len(x), len(y))
			log.Debugf("     path -> %s", path)
			return false
		}

		var matches int
		flagged := make([]bool, len(y))
		for _, v := range x {
			for i, v2 := range y {
				if equal(v, v2, vertexesOfClout1, vertexesOfClout2, path) && !flagged[i] {
					matches++
					flagged[i] = true
					break
				}
			}
		}
		if matches != len(x) {
			log.Debugf("     array comparison failed. Only matched [%d] of [%d] elements of array:", matches, len(x))
			log.Debugf("     %s", path)
		}
		return matches == len(x)
	default:
		if vx != vy {
			log.Debugf("     not equal-> [%s] vs [%s]", vx, vy)
			log.Debugf("     path -> %s", path)
		}
		return vx == vy
	}
}

//compare edgesOuts of vertexes from both clout files with following:
//   . get match edges from both vertexes
//   . compare metadata, properties and name of target's vertex based on 'targetID' of edge
//   . if target vertex is of type operation or workflowActivity then compare that whole vertex
func compareEdgesOutOfVertexes(vertex1EdgeData interface{}, vertex2EdgeData interface{}, vertexesOfClout1 ard.Map,
	vertexesOfClout2 ard.Map, path string) bool {
	edgesOfVertex1, _ := vertex1EdgeData.([]interface{})
	edgesOfVertex2, _ := vertex2EdgeData.([]interface{})
	var match bool

	if (len(edgesOfVertex1) == 0) && (len(edgesOfVertex2) == 0) {
		return true
	}

	//compare all edges from edgesOut of vertex from clout1
	for _, edgeOfVertex1 := range edgesOfVertex1 {
		match = false

		//based on 'targetID' from edge, get name of target vertex from clout1
		mapOfEdgeFromVertex1, _ := edgeOfVertex1.(ard.Map)
		dataOfTargetVertexFromEdgeOfVertex1 := findVertexBasedOnID(mapOfEdgeFromVertex1["targetID"].(string), vertexesOfClout1)
		targetVertexOfEdgeFromVertex1, _ := dataOfTargetVertexFromEdgeOfVertex1.(ard.Map)
		propertiesOfTargetVertex, _ := targetVertexOfEdgeFromVertex1["properties"].(ard.Map)
		nameOfTargetVertexFromClout1, _ := propertiesOfTargetVertex["name"]

		//get matching egde from clout2 and compare it with edge of clout1
		for _, edge2 := range edgesOfVertex2 {

			//compare only metadata and properties of edges
			if !compareVersionMetadataAndProperties(edgeOfVertex1.(ard.Map), edge2.(ard.Map), "", "", nil) {
				continue
			}

			//based on 'targetID' from edge get name of target vertex from clout2
			mapOfEdgeFromVertex2, _ := edge2.(ard.Map)
			dataOfTargetVertexFromEdgeOfVertex2 := findVertexBasedOnID(mapOfEdgeFromVertex2["targetID"].(string), vertexesOfClout2)
			targetVertexOfEdgeFromVertex2, _ := dataOfTargetVertexFromEdgeOfVertex2.(ard.Map)
			propertiesOfTargetVertex, _ := targetVertexOfEdgeFromVertex2["properties"].(ard.Map)
			nameOfTargetVertexFromClout2, _ := propertiesOfTargetVertex["name"]

			//compare name of vertex that is obtained from 'targetID' from both edges
			if (nameOfTargetVertexFromClout1 == nameOfTargetVertexFromClout2) &&
				(nameOfTargetVertexFromClout1 != nil && nameOfTargetVertexFromClout2 != nil) {
				match = true
				break
			}
			log.Debugf("Target names: [%s] and [%s] do not match", nameOfTargetVertexFromClout1, nameOfTargetVertexFromClout2)

			//if target vertex is of type workflowActivity or operation then compare that whole vertexes
			if (checkVertexType("workflowActivity", targetVertexOfEdgeFromVertex2) && checkVertexType("workflowActivity", targetVertexOfEdgeFromVertex1)) ||
				(checkVertexType("operation", targetVertexOfEdgeFromVertex2) && checkVertexType("operation", targetVertexOfEdgeFromVertex1)) {

				if (equal(targetVertexOfEdgeFromVertex1, targetVertexOfEdgeFromVertex2, vertexesOfClout1, vertexesOfClout2, path)) &&
					(compareVersionMetadataAndProperties(edgeOfVertex1.(ard.Map), edge2.(ard.Map), "", "", nil)) {
					match = true
					break
				}
			}
		}
		if !match {
			return false
		}
	}
	return true
}

//compare directives from both clout files
func compareDirectivesOfVertexes(vertex1DirectivesData interface{}, vertex2DirectivesData interface{}, vertexesOfClout1 ard.Map,
	vertexesOfClout2 ard.Map, path string) bool {
	vertex1Directives, _ := vertex1DirectivesData.([]interface{})
	vertex2Directives, _ := vertex2DirectivesData.([]interface{})
	var vertex1 interface{}
	var vertex2 interface{}
	var match bool
	var matchName bool
	var targetName1 string

	//if both directives are nil then return true
	if len(vertex1Directives) == 0 && len(vertex2Directives) == 0 {
		return true
	}

	//compare directives of vertex1 with vertex2.
	for _, directiveOfVertex1 := range vertex1Directives {
		directivesOfVertex1 := strings.Split(directiveOfVertex1.(string), ":")
		directiveNameOfVertex1 := directivesOfVertex1[0]
		matchName = false

		for ind1, vertexIDFromDirectives := range directivesOfVertex1 {
			match = false
			if ind1 == 0 {
				continue
			}
			matchTargetName := false
			vertex1 = findVertexBasedOnID(vertexIDFromDirectives, vertexesOfClout1)
			targetName1, _ = getNameFromVertexProperties(vertex1)

			for _, directiveOfVertex2 := range vertex2Directives {
				directivesOfVertex2 := strings.Split(directiveOfVertex2.(string), ":")
				directiveNameOfVertex2 := directivesOfVertex2[0]

				if directiveNameOfVertex1 == directiveNameOfVertex2 {
					matchName = true
					log.Debugf("Comparing directives with name [%s]", directiveNameOfVertex1)
					for ind2, vertexIDFromDirectives2 := range directivesOfVertex2 {
						if ind2 == 0 {
							continue
						}
						vertex2 = findVertexBasedOnID(vertexIDFromDirectives2, vertexesOfClout2)
						if vertex2 != nil {
							var targetName2 string
							targetName2, _ = getNameFromVertexProperties(vertex2)
							if targetName1 != targetName2 {
								log.Debugf("Directives with target property names/types [%s] & [%s] do not match, skipping further comparison.", targetName1, targetName2)
							} else if targetName1 != "" {
								matchTargetName = true
								log.Debugf("Comparing directives with target property name/type [%s]", targetName1)
							}
						}

						if matchTargetName {
							log.Debugf("Comparing directives wtih vertex ids [%s] & [%s]", vertexIDFromDirectives, vertexIDFromDirectives2)
							//compare both name of directives and target vertexes for directive
							if equal(vertex1, vertex2, vertexesOfClout1, vertexesOfClout2, path) {
								// &&
								//	(directiveNameOfVertex1 == directiveNameOfVertex2) {
								match = true
								break
							} else {
								log.Debugf("     failed comparing directives wtih target name/type [%s] and vertex ids [%s] & [%s]", targetName1, vertexIDFromDirectives, vertexIDFromDirectives2)
								break
							}
						}

					}
				}
			}

			if !match {
				if !matchName {
					log.Debugf("     directive [%s] is not present in the other clout file", directiveNameOfVertex1)
					log.Debugf("     path -> %s", path)
				} else if !matchTargetName {
					log.Debugf("     directive with target name/type [%s] is not present in the other clout file", targetName1)
					log.Debugf("     path -> %s", path)
				}
				return false
			}
		}
	}
	return true
}

func getJSONObject(fileName string) ard.Map {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Infof(err.Error())
	}
	log.Infof("Successfully Opened [%s]", fileName)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	//log.Infof(string(byteValue))

	var result ard.Map
	json.Unmarshal([]byte(byteValue), &result)

	return result
}

//ignore location, path, url and description from clout as they may be different in clout
func ignoreFields(ignoreKey string) bool {
	ignoreList := [4]string{"location", "path", "url", "description"}
	for _, key := range ignoreList {
		if key == ignoreKey {
			return true
		}
	}
	return false
}

//count total number of vertexes from clout
func countTotalNumberOfVertexes(vertexes ard.Map) int {
	var count int = 0
	for _, vertex := range vertexes {
		if vertex != nil {
			count++
		}
	}
	return count
}

//count vertexes from their type
func countVertexesOfSpecificType(vertexType string, vertexes ard.Map) int {
	var count int = 0
	for _, vertex := range vertexes {
		vertexPropertiesMap, _ := vertex.(ard.Map)
		if vertexPropertiesMap != nil {
			vertexMetadata, _ := vertexPropertiesMap["metadata"].(ard.Map)
			metaDataName, _ := vertexMetadata["puccini-tosca"].(ard.Map)
			kindName, _ := metaDataName["kind"].(interface{})
			if kindName.(string) == vertexType {
				count++
			}
		}
	}
	return count
}

//check type of given vertex
func checkVertexType(vertexType string, vertex interface{}) bool {
	vertexPropertiesMap, _ := vertex.(ard.Map)
	vertexMetadata, _ := vertexPropertiesMap["metadata"].(ard.Map)
	if vertexMetadata != nil {
		metadata, _ := vertexMetadata["puccini-tosca"].(ard.Map)
		kindName, _ := metadata["kind"].(interface{})
		if kindName.(string) == vertexType {
			return true
		}
	}
	return false
}

//find vertex in clout from their ID
func findVertexBasedOnID(vertexid string, vertexes ard.Map) interface{} {
	for ID, vertex := range vertexes {
		if vertexid == ID {
			return vertex
		}
	}
	return nil
}

//returns name of clout file from their path
func getNameOfCloutFile(fileName string) string {
	fileNames := strings.Split(fileName, "/")
	len := len(fileNames)
	return fileNames[len-1]
}

// getJSONPath - returns path of a clout file in JSON format
//               If the input path is of a clout file in YAML format,
//               it generates the clout file in JSON format and returns its path
//
//               If the input path is of a clout file in JSON format,
//               it does nothing but simply return the input path
func getJSONPath(yamlPath string) string {

	var jsonPath string

	idot := strings.LastIndex(yamlPath, ".")
	if idot >= 0 {
		ext := strings.ToLower(yamlPath[idot+1:])
		if ext == "yaml" {
			clout, err := cmd.ReadClout(yamlPath)
			common.FailOnError(err)
			jsonPath = yamlPath[0 : idot+1]
			jsonPath = jsonPath + "json"

			err = format.WriteOrPrint(clout, "json", true, jsonPath)
			common.FailOnError(err)
		} else if ext == "json" {
			jsonPath = yamlPath
		}
	}
	return jsonPath
}

func getNameFromVertexProperties(vertex interface{}) (string, bool) {
	if vertex != nil {
		nt, ok := vertex.(ard.Map)
		if ok {
			props, ok := nt["properties"]
			if ok {
				name, ok := props.(ard.Map)["name"].(string)
				if ok {
					return name, true
				} else {
					name, ok := props.(ard.Map)["type"].(string)
					if ok {
						return name, true
					}
				}
			}
		}
	}

	return "", false
}
func getMapKeys(mymap ard.Map) []string {
	keys := make([]string, len(mymap))

	i := 0
	for k := range mymap {
		keys[i] = k
		i++
	}

	return keys
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
