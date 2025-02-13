package httpx

import "github.com/mitchellh/mapstructure"

func struct2Map(item interface{}) (map[string]interface{}, error) {
	var fields map[string]interface{}
	err := mapstructure.Decode(item, &fields)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func map2Struct[T any](item map[string]interface{}) (T, error) {
	// var fields map[string]interface{}
	var result T
	err := mapstructure.Decode(item, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}
