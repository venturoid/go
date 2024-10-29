package logger

import (
	"fmt"
	"reflect"
)

var (
	textError  = "Error function through Map :: key: %s :: value: %v :: message: failed to convert interface to form %s"
	credFields = map[string]bool{
		"password":   true,
		"pin":        true,
		"email":      true,
		"ktp":        true,
		"npwp":       true,
		"birthDate":  true,
		"currentPin": true,
		"newPin":     true,
	}
)

type Document struct{}

type (
	MapType   map[string]interface{}
	ArrayType []interface{}
)

func (doc *Document) ThroughMap(docMap MapType) MapType {
	for k, v := range docMap {
		if v == nil {
			continue
		}
		vt := reflect.TypeOf(v)
		switch vt.Kind() {
		case reflect.Map:
			if mv, ok := v.(map[string]interface{}); ok {
				docMap[k] = doc.ThroughMap(mv)
			} else {
				panic(fmt.Sprintf(textError, k, v, "map[string]interface{}"))
			}
		case reflect.Array, reflect.Slice:
			if mv, ok := v.([]interface{}); ok {
				if len(mv) > 10 {
					docMap[k] = fmt.Sprintf(`{"count" : "%d"}`, len(mv))
				} else {
					docMap[k] = doc.throughArray(mv)
				}
			} else {
				panic(fmt.Sprintf(textError, k, v, "[]interface{}"))
			}
		default:
			if credFields[k] {
				docMap[k] = "*******"
			} else {
				docMap[k] = v
			}
		}
	}
	return docMap
}

func (doc *Document) throughArray(arrayType ArrayType) ArrayType {
	for k, v := range arrayType {
		vt := reflect.TypeOf(v)
		switch vt.Kind() {
		case reflect.Map:
			if mv, ok := v.(map[string]interface{}); ok {
				arrayType[k] = doc.ThroughMap(mv)
			} else {
				panic(fmt.Sprintf(textError, k, v, "map[string]interface{}"))
			}
		case reflect.Array, reflect.Slice:
			if mv, ok := v.([]interface{}); ok {
				arrayType[k] = doc.throughArray(mv)
			} else {
				panic(fmt.Sprintf(textError, k, v, "[]interface{}"))
			}
		default:
			arrayType[k] = v
		}
	}
	return arrayType
}
