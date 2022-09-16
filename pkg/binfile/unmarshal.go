package binfile

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

//var ErrAbortArrayTerminator = fmt.Errorf("aborting due to array-terminator found")

// var ErrInvalidAddressAnnotation = fmt.Errorf("invalid address annotation")
var ErrAnnotatedFieldNotWritable = fmt.Errorf("annotated Field is not writable")

var ErrFoundZeroValueBytes = errors.New("specified range is all zero value bytes")

func Unmarshal(inputBytes []byte, target interface{}, enc Encoding, tz Timezone, arrayTerminator string) error {

	// only pointers allowed
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("unable to unmarshal %s of type %s", reflect.TypeOf(target).Name(), reflect.TypeOf(target))
	}

	var targetValue = reflect.ValueOf(target).Elem()
	var targetKind = targetValue.Kind()
	switch targetKind {
	case reflect.Struct:
		_, err := internalUnmarshal(inputBytes, 0, targetValue, arrayTerminator, 1, enc, tz)
		return err

	case reflect.Slice:
		var targetInnerKind = reflect.ValueOf(targetValue).Kind()

		var currentByte = 0
		for {
			var outputTarget = reflect.New(targetValue.Type().Elem())

			switch targetInnerKind {
			case reflect.Slice:
				// TODO: slice of slices?

			case reflect.Struct:

				var processedBytes, err = internalUnmarshal(inputBytes[currentByte:], 0, outputTarget.Elem(), arrayTerminator, 1, enc, tz)
				if err != nil {
					return err
				}

				currentByte += processedBytes

				targetValue = reflect.Append(targetValue, outputTarget.Elem())
				reflect.ValueOf(target).Elem().Set(targetValue)

			default:
				return fmt.Errorf("invalid target - should be struct or slice of structs")
			}

			// top-level arrays are always terminator types - advance through
			currentByte, _ = advanceThroughTerminator(inputBytes, currentByte, arrayTerminator)

			if currentByte >= len(inputBytes) {
				return nil // the end (do not move this lower in code, as the boundary check has to be first)
			}
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

		// Really useful debugging:
		for k := 0; k < depth; k++ {
			fmt.Print(" ")
		}
		fmt.Printf("Field %s (%d:%d) with at %d \n",
			record.Type().Field(fieldNo).Name,
			absoluteAnnotatedPos, relativeAnnotatedLength, currentByte)

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

					currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, outputTarget.Elem(), relativeAnnotatedLength, annotationList, depth+1, enc, tz)
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
					var isFound bool
					if currentByte, isFound = advanceThroughTerminator(inputBytes, currentByte, arrayTerminator); isFound {
						break
					}
				}

			}

			continue
		}

		if !hasAnnotatedAddress {
			return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, fmt.Errorf("non-struct field must have address annotation"))
		}

		currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, recordField, relativeAnnotatedLength, annotationList, depth+1, enc, tz)
		if err != nil {
			return currentByte, fmt.Errorf("error processing field '%s' `%s`: %w", record.Type().Field(fieldNo).Name, binTag, err)
		}
	}

	return currentByte, nil
}

func unmarshalSimpleTypes(inputBytes []byte, currentByte int, recordField reflect.Value, relativeAnnotatedLength int, annotationList []string, depth int, enc Encoding, tz Timezone) (int, error) {

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

		if hasAnnotationPadspace(annotationList) {
			strvalue = strings.Replace(strvalue, " ", "", -1) // ex.: "-  3"
		}

		num, err := strconv.Atoi(strvalue)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(num))

	case reflect.Float32:

		strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength

		if hasAnnotationPadspace(annotationList) {
			strvalue = strings.Replace(strvalue, " ", "", -1) // ex.: "-  3.1"
		}

		num, err := strconv.ParseFloat(strvalue, 32)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(float32(num)))

	case reflect.Float64:

		strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
		currentByte += relativeAnnotatedLength

		if hasAnnotationPadspace(annotationList) {
			strvalue = strings.Replace(strvalue, " ", "", -1) // ex.: "-  3.1"
		}

		num, err := strconv.ParseFloat(strvalue, 64)
		if err != nil {
			return currentByte, err
		}

		reflect.ValueOf(recordField.Addr().Interface()).Elem().Set(reflect.ValueOf(float64(num)))

	default:

		return currentByte, fmt.Errorf("unsupported field type '%s'", valueKind)

	}

	return currentByte, nil
}
