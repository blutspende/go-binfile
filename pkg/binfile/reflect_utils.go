package binfile

import "reflect"

func getFieldFromStruct(structValue reflect.Value, name string) (reflect.Value, bool) {
	for fieldNo := 0; fieldNo < structValue.NumField(); fieldNo++ {
		if structValue.Type().Field(fieldNo).Name == name {
			return structValue.Field(fieldNo), true
		}
	}
	return reflect.Value{}, false
}
