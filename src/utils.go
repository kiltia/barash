package main

import (
	"encoding/json"
	"math"
)

var NAN = math.NaN()

func structToMap[T any](data T) (map[string]*string, error) {
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
