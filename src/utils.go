package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/jszwec/csvutil"
)

var NAN = math.NaN()

func structToMap[T comparable](data T) (map[string]*string, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	mapData := make(map[string]*string)
	err = json.Unmarshal(dataBytes, &mapData)
	if err != nil {
		return nil, err
	}
	return mapData, nil
}

func loadVerifyParamsFromCSV(path string) []VerifyParams {
	content, _ := os.ReadFile(path)
	var paramsList []VerifyParams
	csvutil.Unmarshal(content, &paramsList)
	return paramsList
}

func unmarshalJson(body []byte, result *VerificationResponse) {
	json.Unmarshal(body, result)
	fmt.Println(string(body))
}
