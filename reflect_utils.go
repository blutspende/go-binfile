package binfile

import "reflect"

// Iterates through a struct in 'structValue' and returns the field with the provided 'name' and a bool accordingly. (', ok' idiom)
func getFieldFromStruct(structValue reflect.Value, name string) (reflect.Value, bool) {
	for fieldNo := 0; fieldNo < structValue.NumField(); fieldNo++ {
		if structValue.Type().Field(fieldNo).Name == name {
			return structValue.Field(fieldNo), true
		}
	}
	return reflect.Value{}, false
}
