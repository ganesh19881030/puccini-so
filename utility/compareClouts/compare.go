package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/tliron/puccini/common"
)

func main() {
	args := os.Args
	var path1 string
	var path2 string
	var errs []string

	if len(args) < 3 {
		fmt.Println("Usage: go run compare.go file1.json file2.json")
		os.Exit(1)
	} else if len(args) == 3 {
		path1 = args[1]
		path2 = args[2]
	}

	clout1 := getJSONObject(path1)
	clout2 := getJSONObject(path2)

	if clout1 == nil || clout2 == nil {
		common.Failf("Invalid file path. path1 = %v, path2 = %v", path1, path2)
	}

	//get names of clout files from their path
	fileName1 := getNameOfCloutFile(path1)
	fileName2 := getNameOfCloutFile(path2)

	//compare metadata, properties and version of clout and vertex's edgesOut
	compareVersionMetadataAndProperties(clout1, clout2, fileName1, fileName2, &errs)

	//compare count of vertexes based on their types
	compareCountOfVertexTypes(clout1, clout2, fileName1, fileName2, &errs)

	//compare vertexes of clout1 with clout2
	compareVertexes(clout1, clout2, fileName1, fileName2, &errs)

	//compare vertexes of clout2 with clout1
	compareVertexes(clout2, clout1, fileName2, fileName1, &errs)

	//print errors on console
	if len(errs) != 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		common.Failf("Files " + fileName1 + " and " + fileName2 + " did not match")
	}
	fmt.Println("Files " + fileName1 + " and " + fileName2 + " are matched")
}

//this method compares following:
//	. version, metadata and properties of clout
//	. metadata and properties of vertex's edgesOuts
func compareVersionMetadataAndProperties(clout1 map[string]interface{}, clout2 map[string]interface{},
	fileName1 string, fileName2 string, errs *[]string) bool {
	arr := [2]string{"metadata", "properties"}
	var match bool = true

	//compare version of clout
	versionOfClout1, _ := clout1["version"].(interface{})
	versionOfClout2, _ := clout2["version"].(interface{})
	if !equal(versionOfClout1, versionOfClout2, nil, nil) {
		if errs != nil {
			*errs = append(*errs, "Clout version in "+fileName1+" and "+fileName2+" did not match")
		}
		match = false
	}

	//compare metadata and properties of clout
	for _, data := range arr {
		dataOfClout1, _ := clout1[data].(map[string]interface{})
		dataOfClout2, _ := clout2[data].(map[string]interface{})
		if !equal(dataOfClout1, dataOfClout2, nil, nil) {
			if errs != nil {
				*errs = append(*errs, "Clout "+data+" in "+fileName1+" and "+fileName2+" did not match")
			}
			match = false
		}
	}

	return match
}

//compare count of vertexes and their types in both clout files
func compareCountOfVertexTypes(clout1 map[string]interface{}, clout2 map[string]interface{}, fileName1 string, fileName2 string,
	errs *[]string) {
	arr := [8]string{"nodeTemplate", "workflowStep", "workflowActivity", "workflow", "policy", "operation", "substitution", "group"}
	vertexesOfClout1, _ := clout1["vertexes"].(map[string]interface{})
	vertexesOfClout2, _ := clout2["vertexes"].(map[string]interface{})

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
func compareVertexes(clout1 map[string]interface{}, clout2 map[string]interface{}, fileName1 string, fileName2 string,
	errs *[]string) {
	vertexesOfClout1, _ := clout1["vertexes"].(map[string]interface{})
	vertexesOfClout2, _ := clout2["vertexes"].(map[string]interface{})
	arr := [5]string{"nodeTemplate", "workflowStep", "policy", "group", "workflow"}
	var match bool
	var matchedVertex interface{}

	//compare vertexes of type nodeTemplate, workflowStep, policy, group and workflow
	for _, vertexType := range arr {
		for _, vertex := range vertexesOfClout1 {
			if !checkVertexType(vertexType, vertex) {
				continue
			}
			//get vertex name from clout1 and search vertex with that name in clout2 and then compare that two vertexes
			vertexMap, _ := vertex.(map[string]interface{})
			clout1VertexProperties, _ := vertexMap["properties"].(map[string]interface{})
			vertexName, _ := clout1VertexProperties["name"].(interface{}).(string)

			//find vertex from their name in clout2
			for _, vertex := range vertexesOfClout2 {
				vertexPropertiesMap, _ := vertex.(map[string]interface{})
				vertexProperties, _ := vertexPropertiesMap["properties"].(map[string]interface{})
				name, _ := vertexProperties["name"]
				if (name != nil) && (name == vertexName) {
					matchedVertex = vertex
				}
			}

			if matchedVertex == nil {
				*errs = append(*errs, "Vertex of type '"+vertexType+"' with name '"+vertexName+"' in "+fileName1+" not found in "+fileName2+"")
				continue
			}

			//compare vertexes from both clouts
			if !equal(vertex, matchedVertex, vertexesOfClout1, vertexesOfClout2) {
				*errs = append(*errs, "Vertex of type '"+vertexType+"' with name '"+vertexName+"' in "+fileName1+" did not match with "+fileName2+"")
			}
		}
	}

	//compare vertex of type substitution from both clout files
	for _, vertexOfClout1 := range vertexesOfClout1 {
		match = false
		if !checkVertexType("substitution", vertexOfClout1) {
			continue
		}
		for _, vertexOfClout2 := range vertexesOfClout2 {
			if !checkVertexType("substitution", vertexOfClout2) {
				continue
			}
			if match = equal(vertexOfClout1, vertexOfClout2, vertexesOfClout1, vertexesOfClout2); match {
				break
			}
		}
		if !match {
			*errs = append(*errs, "Vertex of type 'substitution' in "+fileName1+" did not match with "+fileName2+"")
		}
	}
}

//check for equality between two objects
func equal(vx, vy interface{}, vertexesOfClout1 map[string]interface{}, vertexesOfClout2 map[string]interface{}) bool {
	if reflect.TypeOf(vx) != reflect.TypeOf(vy) {
		return false
	}
	switch x := vx.(type) {
	case map[string]interface{}:

		y := vy.(map[string]interface{})

		if len(x) != len(y) {
			return false
		}

		for k, v := range x {
			val2 := y[k]

			//ignore location, path, url and description in clout
			if ignoreFields(k) {
				continue
			}

			//compare edgesOut of vertexes
			if k == "edgesOut" {
				if !compareEdgesOutOfVertexes(v, val2, vertexesOfClout1, vertexesOfClout2) {
					return false
				}
				continue
			}

			if k == "directives" {
				if !compareDirectivesOfVertexes(v, val2, vertexesOfClout1, vertexesOfClout2) {
					return false
				}
				continue
			}

			if (v == nil) != (val2 == nil) {
				return false
			}

			if k != "history" && !equal(v, val2, vertexesOfClout1, vertexesOfClout2) {
				return false
			}
		}
		return true

	case []interface{}:
		y := vy.([]interface{})

		if len(x) != len(y) {
			return false
		}

		var matches int
		flagged := make([]bool, len(y))
		for _, v := range x {
			for i, v2 := range y {
				if equal(v, v2, vertexesOfClout1, vertexesOfClout2) && !flagged[i] {
					matches++
					flagged[i] = true
					break
				}
			}
		}
		return matches == len(x)
	default:
		return vx == vy
	}
}

//compare edgesOuts of vertexes from both clout files with following:
//   . get match edges from both vertexes
//   . compare metadata, properties and name of target's vertex based on 'targetID' of edge
//   . if target vertex is of type operation or workflowActivity then compare that whole vertex
func compareEdgesOutOfVertexes(vertex1EdgeData interface{}, vertex2EdgeData interface{}, vertexesOfClout1 map[string]interface{},
	vertexesOfClout2 map[string]interface{}) bool {
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
		mapOfEdgeFromVertex1, _ := edgeOfVertex1.(map[string]interface{})
		targetVertexOfEdgeFromVertex1 := findVertexBasedOnID(mapOfEdgeFromVertex1["targetID"].(string), vertexesOfClout1).(map[string]interface{})
		propertiesOfTargetVertex, _ := targetVertexOfEdgeFromVertex1["properties"].(map[string]interface{})
		nameOfTargetVertexFromClout1, _ := propertiesOfTargetVertex["name"]

		//get matching egde from clout2 and compare it with edge of clout1
		for _, edge2 := range edgesOfVertex2 {

			//compare only metadata and properties of edges
			if !compareVersionMetadataAndProperties(edgeOfVertex1.(map[string]interface{}), edge2.(map[string]interface{}), "", "", nil) {
				continue
			}

			//based on 'targetID' from edge get name of target vertex from clout2
			mapOfEdgeFromVertex2, _ := edge2.(map[string]interface{})
			targetVertexOfEdgeFromVertex2 := findVertexBasedOnID(mapOfEdgeFromVertex2["targetID"].(string), vertexesOfClout2).(map[string]interface{})
			propertiesOfTargetVertex, _ := targetVertexOfEdgeFromVertex2["properties"].(map[string]interface{})
			nameOfTargetVertexFromClout2, _ := propertiesOfTargetVertex["name"]

			//compare name of vertex that is obtained from 'targetID' from both edges
			if (nameOfTargetVertexFromClout1 == nameOfTargetVertexFromClout2) &&
				(nameOfTargetVertexFromClout1 != nil && nameOfTargetVertexFromClout2 != nil) {
				match = true
				break
			}

			//if target vertex is of type workflowActivity, operation, action or condition then compare that whole vertexes
			if (checkVertexType("workflowActivity", targetVertexOfEdgeFromVertex2) && checkVertexType("workflowActivity", targetVertexOfEdgeFromVertex1)) ||
				(checkVertexType("operation", targetVertexOfEdgeFromVertex2) && checkVertexType("operation", targetVertexOfEdgeFromVertex1)) ||
				(checkVertexType("action", targetVertexOfEdgeFromVertex2) && checkVertexType("action", targetVertexOfEdgeFromVertex1)) ||
				(checkVertexType("condition", targetVertexOfEdgeFromVertex2) && checkVertexType("condition", targetVertexOfEdgeFromVertex1)) {

				if (equal(targetVertexOfEdgeFromVertex1, targetVertexOfEdgeFromVertex2, vertexesOfClout1, vertexesOfClout2)) &&
					(compareVersionMetadataAndProperties(edgeOfVertex1.(map[string]interface{}), edge2.(map[string]interface{}), "", "", nil)) {
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
func compareDirectivesOfVertexes(vertex1DirectivesData interface{}, vertex2DirectivesData interface{}, vertexesOfClout1 map[string]interface{},
	vertexesOfClout2 map[string]interface{}) bool {
	vertex1Directives, _ := vertex1DirectivesData.([]interface{})
	vertex2Directives, _ := vertex2DirectivesData.([]interface{})
	var vertex1 interface{}
	var vertex2 interface{}
	var match bool

	//if both directives are nil then return true
	if len(vertex1Directives) == 0 && len(vertex2Directives) == 0 {
		return true
	}

	//compare directives of vertex1 with vertex2.
	for _, directiveOfVertex1 := range vertex1Directives {
		directivesOfVertex1 := strings.Split(directiveOfVertex1.(string), ":")
		directiveNameOfVertex1 := directivesOfVertex1[0]

		for _, vertexIDFromDirectives := range directivesOfVertex1 {
			match = false
			vertex1 = findVertexBasedOnID(vertexIDFromDirectives, vertexesOfClout1)

			for _, directiveOfVertex2 := range vertex2Directives {
				directivesOfVertex2 := strings.Split(directiveOfVertex2.(string), ":")
				directiveNameOfVertex2 := directivesOfVertex2[0]

				for _, vertexIDFromDirectives2 := range directivesOfVertex2 {
					vertex2 = findVertexBasedOnID(vertexIDFromDirectives2, vertexesOfClout2)

					//compare both name of directives and target vertexes for directive
					if equal(vertex1, vertex2, vertexesOfClout1, vertexesOfClout2) &&
						(directiveNameOfVertex1 == directiveNameOfVertex2) {
						match = true
						break
					}
				}
			}

			if !match {
				return false
			}
		}
	}
	return true
}

func getJSONObject(fileName string) map[string]interface{} {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened " + fileName)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	//fmt.Println(string(byteValue))

	var result map[string]interface{}
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
func countTotalNumberOfVertexes(vertexes map[string]interface{}) int {
	var count int = 0
	for _, vertex := range vertexes {
		if vertex != nil {
			count++
		}
	}
	return count
}

//count vertexes from their type
func countVertexesOfSpecificType(vertexType string, vertexes map[string]interface{}) int {
	var count int = 0
	for _, vertex := range vertexes {
		vertexPropertiesMap, _ := vertex.(map[string]interface{})
		if vertexPropertiesMap != nil {
			vertexMetadata, _ := vertexPropertiesMap["metadata"].(map[string]interface{})
			metaDataName, _ := vertexMetadata["puccini-tosca"].(map[string]interface{})
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
	vertexPropertiesMap, _ := vertex.(map[string]interface{})
	vertexMetadata, _ := vertexPropertiesMap["metadata"].(map[string]interface{})
	if vertexMetadata != nil {
		metadata, _ := vertexMetadata["puccini-tosca"].(map[string]interface{})
		kindName, _ := metadata["kind"].(interface{})
		if kindName.(string) == vertexType {
			return true
		}
	}
	return false
}

//find vertex in clout from their ID
func findVertexBasedOnID(vertexid string, vertexes map[string]interface{}) interface{} {
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
