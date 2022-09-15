package binfile

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var ErrAbortArrayTerminator = fmt.Errorf("aborting due to array-terminator found")

// var ErrInvalidAddressAnnotation = fmt.Errorf("invalid address annotation")
var ErrAnnotatedFieldNotWritable = fmt.Errorf("annotated Field is not writable")

var ErrFoundZeroValueBytes = errors.New("specified range is all zero value bytes")

func Unmarshal(inputBytes []byte, target interface{}, enc Encoding, tz Timezone, arrayTerminator string) error {

	switch reflect.ValueOf(target).Kind() {
	case reflect.Ptr:
		nested := reflect.ValueOf(target).Elem()
		switch nested.Kind() {
		case reflect.Struct:
			targetStruct := reflect.ValueOf(target).Elem()
			_, err := internalUnmarshal(inputBytes, 0, targetStruct, arrayTerminator, 1, enc, tz)
			return err // its not a desaster if we accidentaly descended into a non-struct;
		case reflect.Slice:
			targetSlice := reflect.ValueOf(target).Elem()
			targetType := targetSlice.Type()

			currentByte := 0
			for {
				outputTarget := reflect.New(targetType.Elem())

				var err error
				lastByte := currentByte
				currentByte, err = internalUnmarshal(inputBytes, currentByte, outputTarget.Elem(), arrayTerminator, 1, enc, tz)

				if lastByte == currentByte {
					return nil // no further progress
				}

				if err == nil || err == ErrAbortArrayTerminator {
					// helpflul debugging
					// fmt.Printf("Adding slice %#v\n", outputTarget.Elem())
					targetSlice = reflect.Append(targetSlice, outputTarget.Elem())
					reflect.ValueOf(target).Elem().Set(targetSlice)
				}

				if currentByte >= len(inputBytes) {
					return nil // the end (do not move this lower in code, as the boundary check has to be first)
				}

				if err == ErrAbortArrayTerminator {
					continue // just the end of an array
				}

				if err != nil && err != ErrAbortArrayTerminator {
					return err // failed
				}

			}
		default:
			return fmt.Errorf("unable to unmarshal %s of type %s", reflect.TypeOf(target).Name(), reflect.TypeOf(target))
		}
	default:
		return fmt.Errorf("unable to unmarshal %s of type %s", reflect.TypeOf(target).Name(), reflect.TypeOf(target))
	}

}

// use this for recursion
func internalUnmarshal(inputBytes []byte, currentByte int, record reflect.Value, arrayTerminator string, depth int, enc Encoding, tz Timezone) (int, error) {

	var initialStartByte = currentByte

	for fieldNo := 0; fieldNo < record.NumField(); fieldNo++ {

		var recordField = record.Field(fieldNo)

		var binTag = record.Type().Field(fieldNo).Tag.Get("bin")
		if !recordField.CanInterface() {
			if binTag != "" {
				return currentByte, fmt.Errorf("field '%s' is not exported but annotated", record.Type().Field(fieldNo).Name)
			} else {
				continue // TODO: this won't notify you about accidentally not exported nested structs
			}
		}

		var annotationList, hasAnnotations = getAnnotationList(binTag)

		absoluteAnnotatedPos, relativeAnnotatedLength, hasAnnotatedAddress, err := getAddressAnnotation(annotationList)
		if err != nil {
			return currentByte, fmt.Errorf("invalid address annotation field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
		}

		if hasAnnotatedAddress && absoluteAnnotatedPos > 0 {
			// The current field has an absolute Address. This causes the cursor to be forwarded
			var newPos = initialStartByte + absoluteAnnotatedPos
			if len(inputBytes)-initialStartByte < newPos {
				return currentByte, fmt.Errorf("absolute position points out-of-bounds on field '%s' `%s`", record.Type().Field(fieldNo).Name, binTag)
			}
			currentByte = newPos
		}
		/*
			// Really useful debugging:
			for k := 0; k < depth; k++ {
				fmt.Print(" ")
			}
			fmt.Printf("Field %s (%d:%d) with at %d \n",
				record.Type().Field(fieldNo).Name,
				absoluteAnnotatedPos, relativeAnnotatedLength, currentByte)
		*/
		var valueKind = reflect.TypeOf(recordField.Interface()).Kind()

		if valueKind == reflect.Struct {

			var err error
			currentByte, err = internalUnmarshal(inputBytes, currentByte, recordField, arrayTerminator, depth+1, enc, tz)
			if err != nil { // If the nested structure did fail, then bail out
				return currentByte, err
			}

			continue
		}

		if !hasAnnotations {
			continue // Do not process unannotated fields
		}

		if valueKind == reflect.Slice {

			var arrayAnnotation, hasArrayAnnotation = getArrayAnnotation(annotationList)
			if !hasArrayAnnotation {
				return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("array fields must have an array annotation"))
			}

			var targetKind = reflect.TypeOf(recordField.Interface()).Elem().Kind()
			if targetKind != reflect.Struct && !hasAnnotatedAddress {
				return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("non-struct field must have address annotation"))
			}

			var arraySize = -1
			var isTerminatorType = isArrayTypeTerminator(arrayAnnotation)
			if !isTerminatorType {
				if size, isFixedSize := getArrayFixedSize(arrayAnnotation); isFixedSize {
					arraySize = size
				} else if fieldName, isDynamic := getArraySizeFieldName(arrayAnnotation); isDynamic {
					arraySize, err = resolveDynamicArraySize(record, fieldName)
					if err != nil {
						return currentByte, err
					}
				}
			}

			var targetType = recordField.Type()
			var outputSlice = reflect.MakeSlice(targetType, 0, 0)
			recordField.Set(outputSlice)

			var arrayIdx = -1
			var err error
			for {
				arrayIdx++
				if !isTerminatorType {
					if arrayIdx == arraySize {
						break
					}
				}

				var outputTarget = reflect.New(targetType.Elem())
				var lastByte = currentByte

				switch targetKind { // Nested: all here is an array of something
				case reflect.Struct:

					currentByte, err = internalUnmarshal(inputBytes, currentByte, outputTarget.Elem(), arrayTerminator, depth+1, enc, tz)
					if err != nil {
						if !isTerminatorType && errors.Is(err, ErrFoundZeroValueBytes) {
							continue
						}
						return currentByte, err
					}

				default:

					currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, outputTarget.Elem(), relativeAnnotatedLength, annotationList, arrayTerminator, depth+1, enc, tz)
					if err != nil {
						if !isTerminatorType && errors.Is(err, ErrFoundZeroValueBytes) {
							continue
						}
						return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
					}
				}

				if lastByte == currentByte { // we didnt progess a single byte
					break
				}

				outputSlice = reflect.Append(outputSlice, outputTarget.Elem())
				recordField.Set(outputSlice)

				if currentByte >= len(inputBytes) { // read to an end = peaceful exit
					break // read further than the end
				}

				// TODO: are we sure we need to check for a terminator in a fixed sized array's end? ref.: TestMarshalArrayWithFixedLength
				if isTerminatorType || (!isTerminatorType && arrayIdx == arraySize-1) {
					// check if next bytes are a terminator
					var strvalue = string(inputBytes[currentByte : currentByte+len(arrayTerminator)])
					if strvalue == arrayTerminator {
						currentByte = currentByte + len(arrayTerminator)
						break
					}
				}

			}

			continue
		}

		if !hasAnnotatedAddress {
			return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("non-struct field must have address annotation"))
		}

		currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, recordField, relativeAnnotatedLength, annotationList, arrayTerminator, depth+1, enc, tz)
		if err != nil {
			return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
		}
	}

	return currentByte, nil
}

func unmarshalSimpleTypes(inputBytes []byte, currentByte int, recordField reflect.Value, relativeAnnotatedLength int, annotationList []string, arrayTerminator string, depth int, enc Encoding, tz Timezone) (int, error) {

	if relativeAnnotatedLength > 0 {
		// Having a length, the total length is not supposed to exceed the boundaries of the input
		if currentByte+relativeAnnotatedLength-1 > len(inputBytes) {
			return currentByte, fmt.Errorf("reading out of bounds position %d in input data of %d bytes", currentByte+relativeAnnotatedLength, len(inputBytes))
		}
	}

	var byteSum = 0
	for _, val := range inputBytes[currentByte : currentByte+relativeAnnotatedLength] {
		byteSum += int(val)
	}
	if byteSum == 0 {
		return currentByte + relativeAnnotatedLength, ErrFoundZeroValueBytes
	}

	var valueKind = reflect.TypeOf(recordField.Interface()).Kind()
	switch valueKind {
	case reflect.String:

		if !recordField.CanSet() {
			return currentByte, ErrAnnotatedFieldNotWritable
		}

		strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength

		if hasAnnotationTrim(annotationList) {
			strvalue = strings.TrimSpace(strvalue)
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().SetString(reflect.ValueOf(strvalue).String())

	case reflect.Int:

		strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength

		strvalue = strings.TrimSpace(strvalue)

		num, err := strconv.Atoi(strvalue)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(num))

	case reflect.Float32:

		value := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength
		strvalue := strings.TrimSpace(value)
		num, err := strconv.ParseFloat(strvalue, 32)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(float32(num)))

	case reflect.Float64:

		value := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength
		strvalue := strings.TrimSpace(value)
		num, err := strconv.ParseFloat(strvalue, 64)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(float32(num)))

	default:

		return currentByte, fmt.Errorf("unsupported field type '%s'", valueKind)

	}

	return currentByte, nil
}
