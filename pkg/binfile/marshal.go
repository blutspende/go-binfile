package binfile

import (
	"fmt"
	"reflect"
	"strconv"
)

func Marshal(target interface{}, padding byte, enc Encoding, tz Timezone, arrayTerminator string) ([]byte, error) {

	if reflect.TypeOf(target).Kind() == reflect.Ptr {
		return Marshal(reflect.ValueOf(target).Elem(), padding, enc, tz, arrayTerminator)
	}

	// todo check if target is struct :
	targetStruct := reflect.ValueOf(target)
	outBytes, _, err := internalMarshal(targetStruct, false, padding, arrayTerminator, 0, 1 /*depth*/)

	return outBytes, err
}

func internalMarshal(record reflect.Value, onlyPaddWithZeros bool, padding byte, arrayTerminator string, currentByte int, depth int) ([]byte, int, error) {

	outBytes := []byte{}

	for fieldNo := 0; fieldNo < record.NumField(); fieldNo++ {

		var recordField = record.Field(fieldNo)

		var binTag = record.Type().Field(fieldNo).Tag.Get("bin")
		if !recordField.CanInterface() {
			if binTag != "" {
				return []byte{}, currentByte, fmt.Errorf("field '%s' is not exported but annotated", record.Type().Field(fieldNo).Name)
			} else {
				continue // TODO: this won't notify you about accidentally not exported nested structs
			}
		}

		var annotationList, hasAnnotations = getAnnotationList(binTag)

		// skip processing terminator here - it's handled at the end of the array
		// TODO: get rid of terminator field 'cause it will never be red and explicitly written by the array processor hence not needed
		if sliceContainsString(annotationList, ANNOTATION_TERMINATOR) {
			continue
		}

		absoluteAnnotatedPos, relativeAnnotatedLength, hasAnnotatedAddress, err := getAddressAnnotation(annotationList)
		if err != nil {
			return []byte{}, currentByte, fmt.Errorf("invalid address annotation field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
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

		for k := 0; k < depth; k++ {
			fmt.Print(" ")
		}
		fmt.Printf("Field %s (%d:%d) with at %d \n",
			record.Type().Field(fieldNo).Name,
			absoluteAnnotatedPos, relativeAnnotatedLength, currentByte)

		var valueKind = reflect.TypeOf(recordField.Interface()).Kind()
		if valueKind == reflect.Struct {

			var tempOutByte []byte
			var err error
			tempOutByte, currentByte, err = internalMarshal(recordField, onlyPaddWithZeros, padding, arrayTerminator, currentByte, depth+1)
			if err != nil { // If the nested structure did fail, then bail out
				return []byte{}, currentByte, err
			}

			outBytes = append(outBytes, tempOutByte...)

			continue
		}

		if !hasAnnotations {
			continue // Do not process unannotated fields
		}

		if valueKind == reflect.Slice {

			var arrayAnnotation, hasArrayAnnotation = getArrayAnnotation(annotationList)
			if !hasArrayAnnotation {
				return []byte{}, currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("array fields must have an array annotation"))
			}

			var sliceValue = reflect.ValueOf(recordField.Interface())
			var innerValueKind = reflect.TypeOf(recordField.Interface()).Elem().Kind()

			if innerValueKind != reflect.Struct && !hasAnnotatedAddress {
				return []byte{}, currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("non-struct field must have address annotation"))
			}

			var arraySize = sliceValue.Len()
			var isTerminatorType = isArrayTypeTerminator(arrayAnnotation)
			if !isTerminatorType {
				if size, isFixedSize := getArrayFixedSize(arrayAnnotation); isFixedSize {
					arraySize = size
				} else if fieldName, isDynamic := getArraySizeFieldName(arrayAnnotation); isDynamic {
					_ = fieldName // TODO: find size by field name or error
				}
			}

			var tempOutByte []byte
			var err error
			for i := 0; i < arraySize; i++ {

				var currentElement reflect.Value
				if i < sliceValue.Len() {
					currentElement = sliceValue.Index(i)
				} else {
					currentElement = reflect.New(reflect.TypeOf(recordField.Interface()).Elem()).Elem()
					onlyPaddWithZeros = true
				}

				switch innerValueKind {
				case reflect.Struct:

					tempOutByte, currentByte, err = internalMarshal(currentElement, onlyPaddWithZeros, padding, arrayTerminator, currentByte, depth+1)
					if err != nil {
						return []byte{}, currentByte, err
					}
					outBytes = append(outBytes, tempOutByte...)

				default:

					tempOutByte, currentByte, err = processSimpleTypes(currentElement, onlyPaddWithZeros, relativeAnnotatedLength, padding, currentByte, depth)
					if err != nil {
						return []byte{}, currentByte, fmt.Errorf("error processing field %s: %w", record.Type().Field(fieldNo).Name, err)
					}
					outBytes = append(outBytes, tempOutByte...)
				}

			}
			onlyPaddWithZeros = false

			// TODO: why do need the terminator in the 2nd case here?
			if isTerminatorType || sliceValue.Len() > arraySize {
				// TODO: do we always need this at the end or only if it's not the last element in a struct?
				outBytes = append(outBytes, arrayTerminator...)
				currentByte += len(arrayTerminator)
			}

			continue
		}

		if !hasAnnotatedAddress {
			return []byte{}, currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("non-struct field must have address annotation"))
		}

		var tempOutByte []byte
		tempOutByte, currentByte, err = processSimpleTypes(recordField, onlyPaddWithZeros, relativeAnnotatedLength, padding, currentByte, depth)
		if err != nil {
			return []byte{}, currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
		}
		outBytes = append(outBytes, tempOutByte...)

	}

	return outBytes, currentByte, nil
}

func processSimpleTypes(recordField reflect.Value, onlyPaddWithZeros bool, relativeAnnotatedLength int, padding byte, currentByte int, depth int) ([]byte, int, error) {

	if onlyPaddWithZeros {
		return make([]byte, relativeAnnotatedLength), currentByte + relativeAnnotatedLength, nil
	}

	var outBytes = []byte{}

	var valueKind = reflect.TypeOf(recordField.Interface()).Kind()
	switch valueKind {
	case reflect.String:

		var tempBytes = []byte(recordField.String())
		if len(tempBytes) > relativeAnnotatedLength {
			return []byte{}, currentByte, fmt.Errorf("invalid value length '%d'", len(tempBytes))
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
			return []byte{}, currentByte, fmt.Errorf("int conversion overflow 32 vs 64 bit system")
		}

		var isNegative = tempInt < 0
		if isNegative {
			outBytes = append(outBytes, '-')
		}

		var tempBytes = []byte(strconv.Itoa(tempInt))
		if len(tempBytes) > relativeAnnotatedLength {
			return []byte{}, currentByte, fmt.Errorf("invalid value length '%d'", len(tempBytes))
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
			return []byte{}, currentByte, fmt.Errorf("invalid value length '%d'", len(tempBytes))
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
		return []byte{}, currentByte, fmt.Errorf("unsupported field type '%s'", valueKind)

	}

	return outBytes, currentByte, nil
}
