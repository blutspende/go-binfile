package binfile

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Marshal(target interface{}, padding byte, enc Encoding, tz Timezone, arrayTerminator string) ([]byte, error) {

	if reflect.TypeOf(target).Kind() == reflect.Ptr {
		return Marshal(reflect.ValueOf(target).Elem(), padding, enc, tz, arrayTerminator)
	}

	// todo check if target is struct :
	targetStruct := reflect.ValueOf(target)
	outBytes, _, err := internalMarshal(targetStruct, padding, arrayTerminator, 0, 1 /*depth*/)

	return outBytes, err
}

func internalMarshal(record reflect.Value, padding byte, arrayTerminator string, currentByte int, depth int) ([]byte, int, error) {

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

		if !recordField.CanInterface() {
			continue // field is not accessible
		}

		absoluteAnnotatedPos := -1
		relativeAnnotatedLength := -1
		if len(binTagsList) >= 1 {
			if !isValidAddressAnnotation(binTagsList[0]) && binTagsList[0] != "" && binTag != "" {
				return []byte{}, currentByte, fmt.Errorf("invalid address annotation field '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}
			absoluteAnnotatedPos, relativeAnnotatedLength = readAddressAnnotation(binTagsList[0])
		}

		// skip processing terminator here - it's handled at the end of the array
		if sliceContainsString(binTagsList, ANNOTATION_TERMINATOR) {
			continue
		}

		if absoluteAnnotatedPos != -1 {
			if currentByte < absoluteAnnotatedPos {
				var paddingBytes = make([]byte, absoluteAnnotatedPos-currentByte)
				for i := range paddingBytes {
					paddingBytes[i] = padding
				}
				outBytes = append(outBytes, paddingBytes...)
				currentByte += len(paddingBytes)
			} else if currentByte > absoluteAnnotatedPos {
				return []byte{}, currentByte, fmt.Errorf("absoulute position is bigger then the current '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}
		}

		var valueKind = reflect.TypeOf(recordField.Interface()).Kind()
		switch valueKind {
		case reflect.Slice:

			// TODO: only slice of structs for now
			var sliceValue = reflect.ValueOf(recordField.Interface())
			var innerValueKind = reflect.TypeOf(recordField.Interface()).Elem().Kind()
			switch innerValueKind {
			case reflect.Struct:

				var tempOutByte []byte
				var err error

				for i := 0; i < sliceValue.Len(); i++ {
					var currentElement = sliceValue.Index(i)
					tempOutByte, currentByte, err = internalMarshal(currentElement, padding, arrayTerminator, currentByte, depth+1)
					if err != nil {
						return []byte{}, currentByte, err
					}

					outBytes = append(outBytes, tempOutByte...)
				}

				// append the terminator
				// TODO: do we always need this at the end or only if it's not the last element in a struct?
				outBytes = append(outBytes, arrayTerminator...)
				currentByte += len(arrayTerminator)

			}

		case reflect.Struct:

			var tempOutByte []byte
			var err error
			tempOutByte, currentByte, err = internalMarshal(recordField, padding, arrayTerminator, currentByte, depth+1)
			if err != nil { // If the nested structure did fail, then bail out
				return []byte{}, currentByte, err
			}

			outBytes = append(outBytes, tempOutByte...)
		}

		if binTag == "" {
			continue // Do not process unannotated fields
		}

		switch valueKind {
		case reflect.String:

			var tempBytes = []byte(recordField.String())
			if len(tempBytes) > relativeAnnotatedLength {
				return []byte{}, currentByte, fmt.Errorf("invalid value length '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			} else if len(tempBytes) < relativeAnnotatedLength {
				var paddingBytes = make([]byte, relativeAnnotatedLength-len(tempBytes))
				for i := range paddingBytes {
					paddingBytes[i] = byte(' ')
				}
				outBytes = append(outBytes, paddingBytes...)
			}

			outBytes = append(outBytes, tempBytes...)
			currentByte += relativeAnnotatedLength

		case reflect.Int:

			// checks overflow - if system uses int32 as default
			var tempInt = int(recordField.Int())
			if int64(tempInt) != recordField.Int() {
				return []byte{}, currentByte, fmt.Errorf("int conversion overflow '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}

			var isNegative = tempInt < 0
			if isNegative {
				outBytes = append(outBytes, '-')
			}

			var tempBytes = []byte(strconv.Itoa(tempInt))
			if len(tempBytes) > relativeAnnotatedLength {
				return []byte{}, currentByte, fmt.Errorf("invalid value length '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			} else if len(tempBytes) < relativeAnnotatedLength {
				var paddingLength = relativeAnnotatedLength - len(tempBytes)
				var paddingBytes = make([]byte, paddingLength)
				for i := range paddingBytes {
					paddingBytes[i] = byte('0')
				}
				outBytes = append(outBytes, paddingBytes...)
			}

			if isNegative {
				outBytes = append(outBytes, tempBytes[1:]...)
			} else {
				outBytes = append(outBytes, tempBytes...)
			}
			currentByte += relativeAnnotatedLength

		case reflect.Float32, reflect.Float64:

			// TODO: is there a minimal length for a float?

			var tempFloat = recordField.Float()
			var tempStr string
			if reflect.TypeOf(recordField.Interface()).Kind() == reflect.Float32 {
				tempStr = strconv.FormatFloat(tempFloat, 'f', -1, 32)
			} else {
				tempStr = strconv.FormatFloat(tempFloat, 'f', -1, 64)
			}
			if tempFloat == float64(int(tempFloat)) { // is truly an int?
				if relativeAnnotatedLength > 1 {
					tempStr += "."
				}
			}

			var tempBytes = []byte(tempStr)

			if len(tempBytes) > relativeAnnotatedLength {
				return []byte{}, currentByte, fmt.Errorf("invalid value length '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}

			outBytes = append(outBytes, tempBytes...)

			if len(tempBytes) < relativeAnnotatedLength {
				var paddingLength = relativeAnnotatedLength - len(tempBytes)
				var paddingBytes = make([]byte, paddingLength)
				for i := range paddingBytes {
					paddingBytes[i] = byte('0')
				}
				outBytes = append(outBytes, paddingBytes...)
			}

			currentByte += relativeAnnotatedLength

		default:

			if binTag != "" { // only if annotated this wil create an error
				return []byte{}, currentByte, fmt.Errorf("invalid type for field %s", record.Type().Field(fieldNo).Name)
			}
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
