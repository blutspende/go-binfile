package binfile

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type LineBreak int

const (
	NO_LINEBREAKS LineBreak = -1
	CR            LineBreak = 0x0D
	LF            LineBreak = 0x0A
	CRLF          LineBreak = 0x0D0A
)

func (l LineBreak) ToString() string {
	return string(byte(l))
}

const (
	ANNOTATION_TRIM       = "trim"
	ANNOTATION_TERMINATOR = "terminator"
	ANNOTATION_ARRAY      = "array"
)

var errAbortArrayTerminator = fmt.Errorf("Aborting due to array-terminator found")

func Unmarshal(inputStringLines []byte, v interface{}, linebreak LineBreak, arrayTerminator string) error {

	/*if linebreak != NO_LINEBREAKS {
		inputString := string(d)
		inputStringLines := strings.Split(inputString, linebreak.ToString())
	}*/

	_, _, err := reflectInputToStruct(inputStringLines, 1, 0 /*currentbyte*/, v, arrayTerminator)

	if err != nil {
		return err
	}

	return nil
}

func marshal(v interface{}) []byte {

	return []byte{}
}

type RETV int

const (
	OK         RETV = 1
	UNEXPECTED RETV = 2 // an exit that wont abort processing. used for skipping optional records
	ERROR      RETV = 3 // a definite error that stops the process
)

func reflectInputToStruct(bufferedBytes []byte, depth int, currentByte int, targetStruct interface{}, arrayTerminator string) (int, RETV, error) {
	if depth > 32 {
		return currentByte, ERROR, fmt.Errorf("maximum recursion depth reached (%d). Too many nested structures ? - aborting", depth)
	}

	/*
		if bufferedInputLines[currentInputLine] == "" {
			// Caution : +1 might skip one; .. without could stick in loop
			return currentInputLine + 1, UNEXPECTED, errors.New(fmt.Sprintf("Empty Input"))
		}
	*/

	var targetStructType reflect.Type
	var targetStructValue reflect.Value
	if reflect.TypeOf(targetStruct).Kind() == reflect.Struct {
		targetStructType = reflect.TypeOf(targetStruct)
		targetStructValue = reflect.ValueOf(targetStruct)
	} else {
		targetStructType = reflect.TypeOf(targetStruct).Elem()
		targetStructValue = reflect.ValueOf(targetStruct).Elem()
	}

	for i := 0; i < targetStructType.NumField(); i++ {
		currentRecord := targetStructValue.Field(i)

		fmt.Printf("Processing Record %s\n", targetStructValue.Type().Field(i).Name)

		// ftype := targetStructType.Field(i)
		/*
			if targetStructType.Field(i).Type.Kind() == reflect.Slice {
				// Array of Structs
				if reflect.TypeOf(targetStructValue.Interface()).Kind() == reflect.Struct {
					innerStructureType := targetStructType.Field(i).Type.Elem()
					sliceForNestedStructure := reflect.MakeSlice(targetStructType.Field(i).Type, 0, 0)
					for currentInputLine < len(bufferedInputLines) {
						allocatedElement := reflect.New(innerStructureType)
						var err error
						var retv RETV
						currentInputLine, retv, err = reflectInputToStruct(bufferedInputLines, depth+1,
							currentInputLine, allocatedElement.Interface())
						if err != nil {
							if retv == UNEXPECTED {
								break
							}
							if retv == ERROR { // a serious error ends the processing
								return currentInputLine, ERROR, err
							}
							sliceForNestedStructure = reflect.Append(sliceForNestedStructure, allocatedElement.Elem())
							reflect.ValueOf(targetStruct).Elem().Field(i).Set(sliceForNestedStructure)
						}
						continue
					}
				} else if targetStructType.Field(i).Type.Kind() == reflect.Struct { // struct without annotation - descending
					var err error
					var retv RETV

					dood := currentRecord.Addr().Interface()
					currentInputLine, retv, err = reflectInputToStruct(bufferedInputLines, depth+1, currentInputLine, dood)
					if err != nil {
						if retv == UNEXPECTED {
							if depth > 0 {
								// if nested structures abort due to unexpected records that does not create an error
								// as the parse will be continued one level higher
								return currentInputLine, UNEXPECTED, err
							} else {
								return currentInputLine, ERROR, err
							}
						}
						if retv == ERROR { // a serious error ends the processing
							return currentInputLine, ERROR, err
						}
					}
					continue
				} else {
					return currentInputLine, ERROR, errors.New(fmt.Sprintf("Invalid Datatype '%s' - abort unmarshal.", targetStructType.Field(i).Type.Kind()))
				}
			}
		*/

		if currentRecord.Kind() == reflect.Slice {

			innerStructureType := targetStructType.Field(i).Type.Elem()
			sliceForNestedStructure := reflect.MakeSlice(targetStructType.Field(i).Type, 0, 0)
			for { // iterate for as long as the same type repeats
				allocatedElement := reflect.New(innerStructureType)

				if currentByte, err := reflectAnnotatedFields(bufferedBytes, allocatedElement.Elem(), 0, arrayTerminator); err != nil {
					return currentByte, ERROR, fmt.Errorf("failed to process input line '%d' err:%w ", bufferedBytes[currentByte], err)
				}

				sliceForNestedStructure = reflect.Append(sliceForNestedStructure, allocatedElement.Elem())
				reflect.ValueOf(targetStruct).Elem().Field(i).Set(sliceForNestedStructure)

				// keep reading while same elements are up
				currentByte = currentByte + 1
				if currentByte >= len(bufferedBytes) {
					break
				}
			}

		} else { // The "normal" case: scanning a string into a structure :

			var err error
			currentByte, err = reflectAnnotatedFields(bufferedBytes, targetStructValue, currentByte, arrayTerminator)
			if err != nil {
				return currentByte, ERROR, fmt.Errorf("failed to process input line '%d' err:%w", bufferedBytes[currentByte], err)
			}

			break
		}

	}
	return currentByte, OK, nil
}

func readFieldAddressAnnotation(annotation string) (absolute int, relativeLength int, err error) {
	absolute = -1
	relativeLength = -1
	err = nil

	if annotation == "" {
		err = fmt.Errorf("no annotation")
		return
	}

	annotationList := strings.Split(annotation, ",")
	if annotationList[0] != "" {
		absolute, err = strconv.Atoi(annotationList[0])
		if err != nil {
			return
		}
	}

	if annotationList[1] != "" {
		relativeLength, err = strconv.Atoi(annotationList[1])
		if err != nil {
			return
		}
	}

	return
}

type DummyRec struct {
	a string `bin:",2"`
}

// reflectAnnotatedFields - Read the whole struct
//
// Returns:
//   - currentByte Position of the read-index
//   - error
func reflectAnnotatedFields(inputBytes []byte, record reflect.Value, currentByte int, arrayTerminator string) (int, error) {

	if reflect.ValueOf(record).Type().Kind() != reflect.Struct {
		return currentByte, fmt.Errorf("invalid type of target: '%d', expecting 'struct'", reflect.ValueOf(record).Type().Kind())
	}

	for fieldNo := 0; fieldNo < record.NumField(); fieldNo++ {

		recordField := record.Field(fieldNo)
		if !recordField.CanInterface() {
			return currentByte, fmt.Errorf("field %s is not exported - aborting import", recordField.Type().Name())
		}
		recordFieldInterface := recordField.Addr().Interface()

		hasTrimAnnotation := false
		hasTerminatorAnnotation := false
		isArrayAnnotation := false
		binTag := record.Type().Field(fieldNo).Tag.Get("bin")
		if binTag == "" {
			// Nested structs
			if reflect.TypeOf(recordField.Interface()).Kind() == reflect.Struct {
				var err error
				currentByte, _, err = reflectInputToStruct(inputBytes, 1, currentByte, recordFieldInterface, arrayTerminator)
				if err != nil {
					return currentByte, err
				}
			}

			// Not annotated and not a struct or slice -> return current position
			continue
		}
		binTagsList := strings.Split(binTag, ",")
		for i := 0; i < len(binTagsList); i++ {
			binTagsList[i] = strings.Trim(binTagsList[i], " ")
		}
		if sliceContainsString(binTagsList, ANNOTATION_TRIM) {
			// the delimiter is instantly replaced with the delimiters from the file for further parsing. By default that is "\^&"
			hasTrimAnnotation = true
		}
		if sliceContainsString(binTagsList, ANNOTATION_TERMINATOR) {
			hasTerminatorAnnotation = true
		}
		if sliceContainsString(binTagsList, ANNOTATION_ARRAY) {
			isArrayAnnotation = true
		}

		absoluteAnnotatedPos, relativeAnnotatedLength, err := readFieldAddressAnnotation(binTag)
		if err != nil {
			return currentByte, fmt.Errorf("invalid annotation for field %s. (%w)", record.Type().Field(fieldNo).Name, err)
		}

		fmt.Printf("Processing field %3d - %s (%d, %d)\n", currentByte, record.Type().Field(fieldNo).Name, absoluteAnnotatedPos, relativeAnnotatedLength)

		/*
			if absoluteAnnotatedPos == -1 && relativeAnnotatedLength == -1 {
				return currentByte, errors.New(fmt.Sprintf("invalid annotation for field %s. Can not find any position", record.Type().Field(i).Name))
			}*/

		if currentByte >= len(inputBytes) || currentByte < 0 {
			//TODO: user should be able to toggle wether he wants an exact match = error or bestfit = skip silent
			return currentByte, nil // mapped field is beyond the data
		}

		switch reflect.TypeOf(recordField.Interface()).Kind() {
		case reflect.Array, reflect.Slice:

			if isArrayAnnotation { // Only annotated arrays are teaken into account

				if reflect.TypeOf(recordField.Type()).Elem().Kind() == reflect.Struct {

					targetType := recordField.Type()
					output := reflect.MakeSlice(targetType, 0, 0)
					recordField.Set(output)

					for {
						outputTarget := reflect.New(targetType.Elem())
						lastByte := currentByte
						currentByte, _, err = reflectInputToStruct(inputBytes, 1, currentByte, outputTarget.Interface(), arrayTerminator)

						if lastByte == currentByte { // we didnt progess a single byte
							fmt.Println("Stop iterating due to no content")
							break
						}

						output = reflect.Append(output, outputTarget.Elem())
						recordField.Set(output)

						if err == errAbortArrayTerminator {
							fmt.Println("Leave due to terminator")
							break // terminate because the annotated terminator-field was set
						}

						if currentByte >= len(inputBytes) { // read to an end = peaceful exit
							fmt.Println("Leave due to end")
							break // read further than the end
						}
					}
				} else {
					return currentByte, fmt.Errorf("Not implemented yet : Arrays of string,int,float")
				}
			}

			// arrays which are not annotated are simply not processed

		case reflect.String:
			if absoluteAnnotatedPos > 0 {
				currentByte = absoluteAnnotatedPos
			}

			if currentByte+relativeAnnotatedLength > len(inputBytes) {
				return currentByte, fmt.Errorf("reading out of bounds position %d in input data of %d bytes.", currentByte+relativeAnnotatedLength, len(inputBytes))
			}

			strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])

			if hasTerminatorAnnotation {
				fmt.Println("Found the terminator ánnotation ", strvalue)
				if strvalue == arrayTerminator {
					return currentByte, errAbortArrayTerminator
				}
				break
			}

			currentByte += relativeAnnotatedLength

			if hasTrimAnnotation {
				strvalue = strings.TrimSpace(strvalue)
			}

			reflect.ValueOf(recordFieldInterface).Elem().SetString(reflect.ValueOf(strvalue).String())

		case reflect.Int:
			if absoluteAnnotatedPos > 0 {
				currentByte = absoluteAnnotatedPos
			}

			if currentByte+relativeAnnotatedLength > len(inputBytes) {
				return currentByte, fmt.Errorf("reading out of bounds position %d in input data of %d bytes.", currentByte+relativeAnnotatedLength, len(inputBytes))
			}

			strvalue := string(inputBytes[currentByte : currentByte+relativeAnnotatedLength])
			currentByte += relativeAnnotatedLength

			if hasTrimAnnotation {
				strvalue = strings.TrimSpace(strvalue)
			}

			num, err := strconv.Atoi(strvalue)
			if err != nil {
				return currentByte, err
			}

			reflect.ValueOf(recordFieldInterface).Elem().Set(reflect.ValueOf(num))

		case reflect.Float32:
			/*if absolutePos > 0 {
				currentPosition = absolutePos
			}

			value := inputStr[currentPosition : currentPosition+relativeLength]
			currentPosition += relativeLength

			if hasTrimAnnotation {
				value = strings.TrimSpace(value)
			}

			num, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return err
			}
			reflect.ValueOf(recordFieldInterface).Elem().Set(reflect.ValueOf(float32(num)))
			*/
		case reflect.Float64:
			/*if absolutePos > 0 {
				currentPosition = absolutePos
			}

			value := inputStr[currentPosition : currentPosition+relativeLength]
			currentPosition += relativeLength

			if hasTrimAnnotation {
				value = strings.TrimSpace(value)
			}

			num, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			reflect.ValueOf(recordFieldInterface).Elem().Set(reflect.ValueOf(num))
			*/
		case reflect.Struct:
			// here: struct would be anotated (as opposed to the above call)
			// currently: this shouldnt be the case
		default:
			return currentByte, fmt.Errorf("invalid type of Field '%s' while trying to unmarshal", reflect.TypeOf(recordField.Interface()).Kind())
		}
	}

	return currentByte, nil
}

func sliceContainsString(list []string, search string) bool {
	for _, x := range list {
		if x == search {
			return true
		}
	}
	return false
}
