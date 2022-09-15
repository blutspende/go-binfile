package binfile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func getAnnotationList(tag string) ([]string, bool) {

	tag = strings.Replace(tag, " ", "", -1)
	var annotations = strings.Split(tag, ",")
	var annotationsFiltered []string
	for i := range annotations {
		if len(annotations[i]) > 0 {
			annotationsFiltered = append(annotationsFiltered, annotations[i])
		}
	}

	return annotationsFiltered, len(annotationsFiltered) > 0
}

func hasAnnotationTrim(annotationList []string) bool {
	return sliceContainsString(annotationList, "trim")
}

func getArrayAnnotation(annotationList []string) (string, bool) {

	for i, val := range annotationList {
		if strings.HasPrefix(annotationList[i], "array") {
			return val, true
		}
	}

	return "", false
}

func isArrayTypeTerminator(arrayAnnotation string) bool {

	var vals = strings.Split(arrayAnnotation, ":")

	if len(vals) == 2 && vals[1] == "terminator" {
		return true
	}

	return false
}

func getArrayFixedSize(arrayAnnotation string) (int, bool) {

	var vals = strings.Split(arrayAnnotation, ":")

	if len(vals) != 2 {
		return 0, false
	}

	if num, err := strconv.Atoi(vals[1]); err == nil && num > 0 {
		return num, true
	}

	return 0, false
}

func getArraySizeFieldName(arrayAnnotation string) (string, bool) {

	var vals = strings.Split(arrayAnnotation, ":")

	if len(vals) != 2 {
		return "", false
	}

	return vals[1], true
}

func getAddressAnnotation(annotationList []string) (int, int, bool, error) {

	for _, val := range annotationList {
		if isValidAddressAnnotation(val) {
			var abs, rel, err = readAddressAnnotation(val)
			if err != nil {
				return -1, -1, false, err
			}
			return abs, rel, true, nil
		}
	}

	return -1, -1, false, nil
}

// isValidAddressAnnotation - verify the valdity of an address annotation
// returns
//   - true when the string matches /^\d*:\d+$/
//   - false otherwise
func isValidAddressAnnotation(str string) bool {

	expr, err := regexp.Compile(`^\d*:\d+$`)
	if err != nil {
		panic("Invalid Expression in regexp (this error should never occur)")
	}
	matched := expr.Match([]byte(str))

	return matched
}

// read an address annotation with format "absolute:length" or ":length"
func readAddressAnnotation(str string) (int, int, error) {
	var abs = -1
	var length = -1
	var err = error(nil)

	parts := strings.Split(str, ":")

	if len(parts) >= 1 { // the 1st part is the absolute address
		if parts[0] != "" {
			if abs, err = strconv.Atoi(parts[0]); err != nil {
				return -1, -1, fmt.Errorf("invalid absolute position value '%v': %w", parts[0], err)
			}
		}
	}

	if len(parts) >= 2 { // the 2nd part is the length of the field
		if parts[1] != "" {
			if length, err = strconv.Atoi(parts[1]); err != nil {
				return -1, -1, fmt.Errorf("invalid relative length value '%v': %w", parts[1], err)
			}
		}
	}

	return abs, length, err
}
