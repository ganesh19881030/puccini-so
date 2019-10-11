package clout

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
)

func Compare(file1 string, file2 string) {
	jsonObject1 := getJSONObject(file1)
	jsonObject2 := getJSONObject(file2)

	result := Equal(jsonObject1, jsonObject2)

	if result {
		fmt.Println("files matched")
	} else {
		fmt.Println("files did not match")
	}

	//fmt.Println(Equal(jsonObject1, jsonObject2))
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

// Equal checks equality between 2 Body-encoded data.
func Equal(vx, vy interface{}) bool {
	if reflect.TypeOf(vx) != reflect.TypeOf(vy) {
		//fmt.Println("here1")
		return false
	}
	switch x := vx.(type) {
	case map[string]interface{}:

		y := vy.(map[string]interface{})

		if len(x) != len(y) {
			//fmt.Println("here2")
			return false
		}

		for k, v := range x {
			val2 := y[k]

			if (v == nil) != (val2 == nil) {
				//fmt.Println("here3")
				return false
			}

			if k != "history" && !Equal(v, val2) {
				//fmt.Println("here4")
				return false
			}
		}

		return true
	case []interface{}:
		y := vy.([]interface{})

		if len(x) != len(y) {
			//fmt.Println("here5")
			return false
		}

		var matches int
		flagged := make([]bool, len(y))
		for _, v := range x {
			for i, v2 := range y {
				if Equal(v, v2) && !flagged[i] {
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
