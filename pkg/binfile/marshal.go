package binfile

import (
	"fmt"
	"reflect"
	"strconv"
)

func Marshal(target interface{}, padding byte, enc Encoding, tz Timezone, arrayTerminator string) ([]byte, error) {

	// TODO: accepting a Ptr here is confusing as the func will not change the contents
	if reflect.TypeOf(target).Kind() == reflect.Ptr {
		return Marshal(reflect.ValueOf(target).Elem(), padding, enc, tz, arrayTerminator)
	}

	var outBytes []byte
	var err error
	var depth = 0

	var targetValue = reflect.ValueOf(target)
	var targetKind = targetValue.Kind()

	switch targetKind {
	case reflect.Slice:
		var innerValueKind = reflect.TypeOf(targetValue.Interface()).Elem().Kind()

		for i := 0; i < targetValue.Len(); i++ {
			var tempBytes []byte

			switch innerValueKind {
			case reflect.Slice:
				// TODO: slice of slices?

			case reflect.Struct:
				tempBytes, _, err = internalMarshal(targetValue.Index(i), false, padding, arrayTerminator, 0, depth+1)
				if err != nil {
					return []byte{}, err
				}
				outBytes = append(outBytes, tempBytes...)

			default:
				return []byte{}, fmt.Errorf("invalid target - should be struct or slice of structs")
			}
		}

		// TODO: end of the array terminator - might not needed
		outBytes = append(outBytes, []byte(arrayTerminator)...)

		return outBytes, err

	case reflect.Struct:
		// TODO: should we check if all bytes are consumed or at least return the current position for further processing of data?
		outBytes, _, err = internalMarshal(targetValue, false, padding, arrayTerminator, 0, depth)
		return outBytes, err

	}

	return []byte{}, fmt.Errorf("invalid target - should be struct or slice of structs")
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
				return []byte{}, currentByte, fmt.Errorf("absoulute position points backwards '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}
		}
		/*
			for k := 0; k < depth; k++ {
				fmt.Print(" ")
			}
			fmt.Printf("Field %s (%d:%d) with at %d \n",
				record.Type().Field(fieldNo).Name,
				absoluteAnnotatedPos, relativeAnnotatedLength, currentByte)*/

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
					arraySize, err = resolveDynamicArraySize(record, fieldName)
					if err != nil {
						return []byte{}, currentByte, err
					}
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

					tempOutByte, currentByte, err = marshalSimpleTypes(currentElement, onlyPaddWithZeros, relativeAnnotatedLength, padding, currentByte, depth)
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
		tempOutByte, currentByte, err = marshalSimpleTypes(recordField, onlyPaddWithZeros, relativeAnnotatedLength, padding, currentByte, depth)
		if err != nil {
			return []byte{}, currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
		}
		outBytes = append(outBytes, tempOutByte...)

	}

	return outBytes, currentByte, nil
}

func marshalSimpleTypes(recordField reflect.Value, onlyPaddWithZeros bool, relativeAnnotatedLength int, padding byte, currentByte int, depth int) ([]byte, int, error) {

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
