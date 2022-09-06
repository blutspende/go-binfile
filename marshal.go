package binfile

import (
	"fmt"
	"reflect"
	"strings"
)

func Marshal(target interface{}, padding byte, enc Encoding, tz Timezone, arrayTerminator string) ([]byte, error) {

	if reflect.TypeOf(target).Kind() == reflect.Ptr {
		return Marshal(reflect.ValueOf(target).Elem(), padding, enc, tz, arrayTerminator)
	}

	// todo check if target is struct :
	targetStruct := reflect.ValueOf(target)
	outBytes, _, err := internalMarshal(targetStruct, 0, 1 /*depth*/)

	return outBytes, err
}

func internalMarshal(record reflect.Value, currentByte int, depth int) ([]byte, int, error) {

	outBytes := []byte{}

	for fieldNo := 0; fieldNo < record.NumField(); fieldNo++ {

		recordField := record.Field(fieldNo)

		binTag := record.Type().Field(fieldNo).Tag.Get("bin")
		if binTag != "" && !recordField.CanInterface() {
			return []byte{}, currentByte, fmt.Errorf("field '%s' is not exported but annotated", record.Type().Field(fieldNo).Name)
		}

		binTagsList := strings.Split(binTag, ",")
		for i := 0; i < len(binTagsList); i++ {
			binTagsList[i] = strings.Trim(binTagsList[i], " ")
		}

		absoluteAnnotatedPos := -1
		relativeAnnotatedLength := -1
		if len(binTagsList) >= 1 {
			if !isValidAddressAnnotation(binTagsList[0]) && binTagsList[0] != "" && binTag != "" {
				return []byte{}, currentByte, fmt.Errorf("invalid address annotation field '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}
			absoluteAnnotatedPos, relativeAnnotatedLength = readAddressAnnotation(binTagsList[0])
		}

		if !recordField.CanInterface() {
			continue // field is not accessible
		}

		for k := 0; k < depth; k++ {
			fmt.Print(" ")
		}
		fmt.Printf("Field %s (%d:%d) with at %d \n",
			record.Type().Field(fieldNo).Name,
			absoluteAnnotatedPos, relativeAnnotatedLength, currentByte)

	}

	return outBytes, currentByte, nil
}
