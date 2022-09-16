package binfile

import (
	"reflect"
)

// Searches for a field in 'structValue' with the provided 'name' and returns the valid integer value from it or an error.
func resolveDynamicArraySize(structValue reflect.Value, name string) (int, error) {

	var arraySize = -1

	// TODO: this will only work if referenced field is already processed but won't give error otherwise
	if fieldVal, isFieldFound := getFieldFromStruct(structValue, name); isFieldFound {
		var fieldKind = reflect.TypeOf(fieldVal.Interface()).Kind()
		if fieldKind != reflect.Int {
			return arraySize, newUnsupportedTypeError(reflect.TypeOf(fieldVal.Interface()))
		}
		arraySize = int(fieldVal.Int())
		if int64(arraySize) != fieldVal.Int() {
			return arraySize, ErrorIntConversionOverflow
		}
		if arraySize < 0 {
			return arraySize, newInvalidSizeForArrayError(arraySize)
		}
	} else {
		return arraySize, ErrorUnknownFieldName
	}

	return arraySize, nil
}

// find a searchstring within an array of strings. only matches full
// returns
//   - true if the string is present
//   - false otherwise
func sliceContainsString(list []string, search string) bool {
	for _, x := range list {
		if x == search {
			return true
		}
	}
	return false
}

// Checks if the next bytes from 'currentPos' are the provided 'arrayTerminator'.
// Returns the position advanced by the length of the terminator if found and a true boolean value.
// Otherwise it returns the same position and a false boolean value.
//
// NOTE: Will check for array size.
func advanceThroughTerminator(byteArray []byte, currentPos int, arrayTerminator string) (int, bool) {
	if len(byteArray) >= currentPos+len(arrayTerminator) {
		var strvalue = string(byteArray[currentPos : currentPos+len(arrayTerminator)])
		if strvalue == arrayTerminator {
			return currentPos + len(arrayTerminator), true
		}
	}
	return currentPos, false
}

// Creates and adds the padding bytes of provided 'byteToUse' and of requested 'length' to the 'original' byte array.
// Returns the padded byte array and it's new size.
func appendPaddingBytes(original []byte, length int, byteToUse byte) ([]byte, int) {
	var paddingBytes = make([]byte, length)
	for i := range paddingBytes {
		paddingBytes[i] = byteToUse
	}
	var temp = append(original, paddingBytes...)
	return temp, len(temp)
}
