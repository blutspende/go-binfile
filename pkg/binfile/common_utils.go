package binfile

import (
	"fmt"
	"reflect"
)

func resolveDynamicArraySize(structValue reflect.Value, name string) (int, error) {

	var arraySize = -1

	// TOOD: this will only work if referenced field is already processed but won't give error otherwise
	if fieldVal, isFieldFound := getFieldFromStruct(structValue, name); isFieldFound {
		var fieldKind = reflect.TypeOf(fieldVal.Interface()).Kind()
		if fieldKind != reflect.Int {
			return arraySize, fmt.Errorf("invalid type for array size field '%s' - should be int", name)
		}
		arraySize = int(fieldVal.Int())
		if int64(arraySize) != fieldVal.Int() {
			return arraySize, fmt.Errorf("int conversion overflow 32 vs 64 bit system")
		}
		if arraySize < 0 {
			return arraySize, fmt.Errorf("invalid size for array size field '%s'", name)
		}
	} else {
		return arraySize, fmt.Errorf("unknown field name for array size '%s'", name)
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

func advanceThroughTerminator(byteArray []byte, currentPos int, arrayTerminator string) (int, bool) {
	if len(byteArray) >= currentPos+len(arrayTerminator) {
		var strvalue = string(byteArray[currentPos : currentPos+len(arrayTerminator)])
		if strvalue == arrayTerminator {
			return currentPos + len(arrayTerminator), true
		}
	}
	return currentPos, false
}

func appendPaddingBytes(original []byte, length int, byteToUse byte) ([]byte, int) {
	var paddingBytes = make([]byte, length)
	for i := range paddingBytes {
		paddingBytes[i] = byteToUse
	}
	var temp = append(original, paddingBytes...)
	return temp, len(temp)
}
