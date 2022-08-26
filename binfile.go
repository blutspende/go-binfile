package binfile

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type LineBreak int

const (
	CR   LineBreak = 0x0D
	LF   LineBreak = 0x0A
	CRLF LineBreak = 0x0D0A
)

func (l LineBreak) ToString() string {
	return string(byte(l))
}

const (
	ANNOTATION_TRIM = "trim"
)

func Unmarshal(d []byte, v interface{}, linebreak LineBreak) error {
	inputString := string(d)

	inputStringLines := strings.Split(inputString, linebreak.ToString())

	_, _, err := reflectInputToStruct(inputStringLines, 1, 0, v)
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

func reflectInputToStruct(bufferedInputLines []string, depth int, currentInputLine int, targetStruct interface{}) (int, RETV, error) {
	if depth > 10 {
		return currentInputLine, ERROR, errors.New(fmt.Sprintf("Maximum recursion depth reached (%d). Too many nested structures ? - aborting", depth))
	}

	if bufferedInputLines[currentInputLine] == "" {
		// Caution : +1 might skip one; .. without could stick in loop
		return currentInputLine + 1, UNEXPECTED, errors.New(fmt.Sprintf("Empty Input"))
	}

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
		ftype := targetStructType.Field(i)
		binTag := ftype.Tag.Get("bin")
		binTagList := strings.Split(binTag, ",")

		if len(binTagList) < 1 {
			continue // not annotated = no processing
		}

		if binTagList[0] != "" {

		}
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
		expectInputRecordType := binTagList[0] // Expected Record type
		trimInputRecord := false
		if sliceContainsString(binTagList, ANNOTATION_TRIM) {
			trimInputRecord = true
		}

		if len(bufferedInputLines[currentInputLine]) == 0 {
			continue // empty lines can only be skipped
		}

		//TODO: here are some bugs!!
		if expectInputRecordType == bufferedInputLines[currentInputLine] {
			if currentRecord.Kind() == reflect.Slice {
				innerStructureType := targetStructType.Field(i).Type.Elem()
				sliceForNestedStructure := reflect.MakeSlice(targetStructType.Field(i).Type, 0, 0)
				for { // iterate for as long as the same type repeats
					allocatedElement := reflect.New(innerStructureType)

					if err := reflectAnnotatedFields(bufferedInputLines[currentInputLine], allocatedElement.Elem(), trimInputRecord); err != nil {
						return currentInputLine, ERROR, errors.New(fmt.Sprintf("Failed to process input line '%s' err:%s", bufferedInputLines[currentInputLine], err))
					}

					sliceForNestedStructure = reflect.Append(sliceForNestedStructure, allocatedElement.Elem())
					reflect.ValueOf(targetStruct).Elem().Field(i).Set(sliceForNestedStructure)

					// keep reading while same elements are up
					currentInputLine = currentInputLine + 1
					if expectInputRecordType != bufferedInputLines[currentInputLine] {
						break
					}
					if currentInputLine >= len(bufferedInputLines) {
						break
					}
				}

			} else { // The "normal" case: scanning a string into a structure :
				if err := reflectAnnotatedFields(bufferedInputLines[currentInputLine], currentRecord, trimInputRecord); err != nil {
					return currentInputLine, ERROR, errors.New(fmt.Sprintf("Failed to process input line '%s' err:%s", bufferedInputLines[currentInputLine], err))
				}
				currentInputLine = currentInputLine + 1
			}

		} else { // The expected input-record did not occur
			return currentInputLine, UNEXPECTED, errors.New(fmt.Sprintf("Expected Record-Type '%s' input was '%c' in depth (%d) (Abort)", expectInputRecordType, bufferedInputLines[currentInputLine][0], depth))
		}
		if currentInputLine >= len(bufferedInputLines) {
			break
		}
	}
	return currentInputLine, OK, nil
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

func reflectAnnotatedFields(inputStr string, record reflect.Value, trimRecord bool) error {

	if reflect.ValueOf(record).Type().Kind() != reflect.Struct {
		return errors.New(fmt.Sprintf("invalid type of target: '%s', expecting 'struct'", reflect.ValueOf(record).Type().Kind()))
	}

	currentPosition := 0
	for j := 0; j < record.NumField(); j++ {
		recordField := record.Field(j)
		if !recordField.CanInterface() {
			return errors.New(fmt.Sprintf("Field %s is not exported - aborting import", recordField.Type().Name()))
		}
		recordFieldInterface := recordField.Addr().Interface()

		hasTrimAnnotation := false
		binTag := record.Type().Field(j).Tag.Get("bin")
		if binTag == "" {
			continue // nothing to process when someone requires astm:
		}
		binTagsList := strings.Split(binTag, ",")
		for i := 0; i < len(binTagsList); i++ {
			binTagsList[i] = strings.Trim(binTagsList[i], " ")
		}
		if sliceContainsString(binTagsList, ANNOTATION_TRIM) {
			// the delimiter is instantly replaced with the delimiters from the file for further parsing. By default that is "\^&"
			hasTrimAnnotation = true
		}

		absolutePos, relativeLength, err := readFieldAddressAnnotation(binTag)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid annotation for field %s. (%s)", record.Type().Field(j).Name, err))
		}
		if absolutePos == -1 && relativeLength == -1 {
			return errors.New(fmt.Sprintf("Invalid annotation for field %s. Can not find any position", record.Type().Field(j).Name))
		}

		if currentPosition >= len(inputStr) || currentPosition < 0 {
			//TODO: user should be able to toggle wether he wants an exact match = error or bestfit = skip silent
			continue // mapped field is beyond the data
		}

		switch reflect.TypeOf(recordField.Interface()).Kind() {
		case reflect.String:
			if absolutePos > 0 {
				currentPosition = absolutePos
			}
			value := inputStr[currentPosition : currentPosition+relativeLength]
			currentPosition += relativeLength

			if hasTrimAnnotation {
				value = strings.TrimSpace(value)
			}

			reflect.ValueOf(recordFieldInterface).Elem().SetString(reflect.ValueOf(value).String())
		case reflect.Int:
			if absolutePos > 0 {
				currentPosition = absolutePos
			}
			value := inputStr[currentPosition : currentPosition+relativeLength]
			currentPosition += relativeLength

			if hasTrimAnnotation {
				value = strings.TrimSpace(value)
			}

			num, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			reflect.ValueOf(recordFieldInterface).Elem().Set(reflect.ValueOf(num))
		case reflect.Float32:
			if absolutePos > 0 {
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

		case reflect.Float64:
			if absolutePos > 0 {
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

		case reflect.Struct:
			switch reflect.TypeOf(recordField.Interface()).Name() {
			default:
				return errors.New(fmt.Sprintf("Invalid type of Field '%s' while trying to unmarshal this string '%s'. This datatype is a structure type which is not implemented.",
					reflect.TypeOf(recordField.Interface()).Name(), inputStr))
			}
		default:
			return errors.New(fmt.Sprintf("Invalid type of Field '%s' while trying to unmarshal this string '%s'. This datatype is not implemented.",
				reflect.TypeOf(recordField.Interface()).Kind(), inputStr))
		}
	}

	return nil
}

func sliceContainsString(list []string, search string) bool {
	for _, x := range list {
		if x == search {
			return true
		}
	}
	return false
}
