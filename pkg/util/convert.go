package util

import (
	"encoding/json"
	"reflect"
)

func ObjectToMap[T any](data T) (map[string]*string, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	mapData := map[string]*string{}

	err = json.Unmarshal(dataBytes, &mapData)
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(data)
	for i := 0; i < v.NumField(); i++ {
		tagValue := v.Type().Field(i).Tag.Get("json")
		if tagValue == "" {
			delete(mapData, v.Type().Field(i).Name)
		}
	}
	return mapData, nil
}
