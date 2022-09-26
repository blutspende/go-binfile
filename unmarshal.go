package binfile

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

//var ErrAbortArrayTerminator = fmt.Errorf("aborting due to array-terminator found")

// var ErrInvalidAddressAnnotation = fmt.Errorf("invalid address annotation")

// Accepts a byte array that needs to be parsed and a reference to an annotated struct or array of sturcts to parse into.
//
// Returns an error if a problem found or nil. The parsed contents will be in the provided 'target'.
//
// Check the README.md for usage.
func Unmarshal(inputBytes []byte, target interface{}, enc Encoding, tz Timezone, arrayTerminator string) (int, error) {

	// only pointers allowed
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return 0, newUnsupportedTypeError(reflect.TypeOf(target))
	}

	var targetValue = reflect.ValueOf(target).Elem()
	var targetKind = targetValue.Kind()
	switch targetKind {
	case reflect.Struct:
		return internalUnmarshal(inputBytes, 0, targetValue, arrayTerminator, 1, enc, tz)

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
					return currentByte + processedBytes, err
				}

				currentByte += processedBytes

				targetValue = reflect.Append(targetValue, outputTarget.Elem())
				reflect.ValueOf(target).Elem().Set(targetValue)

			default:
				return 0, newUnsupportedTypeError(targetValue.Type())
			}

			// top-level arrays are always terminator types - advance through
			currentByte, _ = advanceThroughTerminator(inputBytes, currentByte, arrayTerminator)

			if currentByte >= len(inputBytes) {
				return currentByte, nil // the end (do not move this lower in code, as the boundary check has to be first)
			}
		}

	default:
		return 0, newUnsupportedTypeError(reflect.TypeOf(target))
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
				return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, ErrorExportedFieldNotAnnotated)
			} else {
				continue // TODO: this won't notify you about accidentally not exported nested structs
			}
		}

		var annotationList, hasAnnotations = getAnnotationList(binTag)

		absoluteAnnotatedPos, relativeAnnotatedLength, hasAnnotatedAddress, err := getAddressAnnotation(annotationList)
		if err != nil {
			return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, newInvalidAddressAnnotationError(err))
		}

		if hasAnnotatedAddress && absoluteAnnotatedPos > 0 {
			// The current field has an absolute Address. This causes the cursor to be forwarded
			var newPos = initialStartByte + absoluteAnnotatedPos
			if len(inputBytes)-initialStartByte < newPos {
				return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, newReadingOutOfBoundsError(newPos, newPos+relativeAnnotatedLength, len(inputBytes)-initialStartByte))
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
				return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, err)
			}

			continue
		}

		if !hasAnnotations {
			continue // Do not process unannotated fields
		}

		if valueKind == reflect.Slice {

			var arrayAnnotation, hasArrayAnnotation = getArrayAnnotation(annotationList)
			if !hasArrayAnnotation {
				return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, ErrorMissingArrayAnnotation)
			}

			var targetKind = reflect.TypeOf(recordField.Interface()).Elem().Kind()
			if targetKind != reflect.Struct && !hasAnnotatedAddress {
				return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, ErrorMissingAddressAnnotation)
			}

			var arraySize = -1
			var isTerminatorType = isArrayTypeTerminator(arrayAnnotation)
			if !isTerminatorType {
				if size, isFixedSize := getArrayFixedSize(arrayAnnotation); isFixedSize {
					arraySize = size
				} else if fieldName, isDynamic := getArraySizeFieldName(arrayAnnotation); isDynamic {
					arraySize, err = resolveDynamicArraySize(record, fieldName)
					if err != nil {
						return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, newInvalidDynamicArraySizeError(record.Type().Name(), fieldName, err))
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
						if !isTerminatorType && errors.Is(err, ErrorFoundZeroValueBytes) {
							continue
						}
						return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, err)
					}

				default:

					currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, outputTarget.Elem(), relativeAnnotatedLength, annotationList, depth+1, enc, tz)
					if err != nil {
						if !isTerminatorType && errors.Is(err, ErrorFoundZeroValueBytes) {
							continue
						}
						return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, err)
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
			return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, ErrorMissingAddressAnnotation)
		}

		currentByte, err = unmarshalSimpleTypes(inputBytes, currentByte, recordField, relativeAnnotatedLength, annotationList, depth+1, enc, tz)
		if err != nil {
			// the last item should actually return the error but itmes before should process to advance the current byte
			if fieldNo < record.NumField()-1 && errors.Is(err, ErrorFoundZeroValueBytes) {
				continue
			}
			return currentByte, newProcessingFieldError(record.Type().Field(fieldNo).Name, binTag, err)
		}
	}

	return currentByte, nil
}

// use this for processing end nodes
func unmarshalSimpleTypes(inputBytes []byte, currentByte int, recordField reflect.Value, relativeAnnotatedLength int, annotationList []string, depth int, enc Encoding, tz Timezone) (int, error) {

	if relativeAnnotatedLength > 0 {
		// Having a length, the total length is not supposed to exceed the boundaries of the input
		if currentByte+relativeAnnotatedLength-1 > len(inputBytes) {
			return currentByte, newReadingOutOfBoundsError(currentByte, currentByte+relativeAnnotatedLength, len(inputBytes))
		}
	}

	var byteSum = 0
	for _, val := range inputBytes[currentByte : currentByte+relativeAnnotatedLength] {
		byteSum += int(val)
	}
	if byteSum == 0 {
		return currentByte + relativeAnnotatedLength, ErrorFoundZeroValueBytes
	}

	if !recordField.CanSet() {
		return currentByte, ErrorAnnotatedFieldNotWritable
	}

	var valueKind = reflect.TypeOf(recordField.Interface()).Kind()
	switch valueKind {
	case reflect.String:

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

		return currentByte, newUnsupportedTypeError(reflect.TypeOf(recordField.Interface()))
	}

	return currentByte, nil
}
