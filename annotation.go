package binfile

import (
	"regexp"
	"strconv"
	"strings"
)

// Processes the annotations aquired from the 'bin' tag.
// Removes optional indentation spaces, splits them by comma and removes empty entries.
//
// Returns a string array of the annotations and a bool with true if there are actual entries in it.
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

// Checks the annotation array if the 'trim' annotation is in it and returns a bool accordingly.
func hasAnnotationTrim(annotationList []string) bool {
	return sliceContainsString(annotationList, "trim")
}

// Checks the annotation array if the 'padspace' annotation is in it and returns a bool accordingly.
func hasAnnotationPadspace(annotationList []string) bool {
	return sliceContainsString(annotationList, "padspace")
}

// Checks the annotation array if the 'forcesign' annotation is in it and returns a bool accordingly.
func hasAnnotationForceSign(annotationList []string) bool {
	return sliceContainsString(annotationList, "forcesign")
}

// Finds and returns the 'array' annotation in the annotation list along with a bool which value is true if found.
func getArrayAnnotation(annotationList []string) (string, bool) {

	for i, val := range annotationList {
		if strings.HasPrefix(annotationList[i], "array") {
			return val, true
		}
	}

	return "", false
}

// Checks the provided 'array' annotation if it's a terminated type and returns a bool accordingly.
//
// NOTE: Will also return false on mistyped values.
func isArrayTypeTerminator(arrayAnnotation string) bool {

	var vals = strings.Split(arrayAnnotation, ":")

	if len(vals) == 2 && vals[1] == "terminator" {
		return true
	}

	return false
}

// Checks the provided 'array' annotation it it has a valid integer value and returns it along with a true boolean value.
//
// Note: Will return a false on missing or non-integer values in which case the value should not be used. (', ok' idiom)
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

// Returns the 'array' annotation's value if found along with a bool accordingly. (', ok' idiom)
//
// Note: This will return true even for a terminated type. Although this library won't restrict you
// to use a field named 'terminator' for an array size, you should refrain from it and check for a terminated type first.
func getArraySizeFieldName(arrayAnnotation string) (string, bool) {

	var vals = strings.Split(arrayAnnotation, ":")

	if len(vals) != 2 {
		return "", false
	}

	return vals[1], true
}

// Returns the absolute position and the relative length from the annotation list along with a bool which is true if found.
// The default value for both is '-1' - this doesn't mean they are valid numbers in your context.
// An error is given back if there was an error while parsing the integers.
//
// NOTE: You should only consider the values valid if the returned bool is true. (', ok' idiom)
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

// Read an address annotation with format "absolute:length" or ":length".
// Gives an error if the values aren't valid integers.
func readAddressAnnotation(str string) (int, int, error) {
	var abs = -1
	var length = -1
	var err = error(nil)

	parts := strings.Split(str, ":")

	if len(parts) >= 1 { // the 1st part is the absolute address
		if parts[0] != "" {
			if abs, err = strconv.Atoi(parts[0]); err != nil {
				return -1, -1, newInvalidAbsolutePositionError(err)
			}
		}
	}

	if len(parts) >= 2 { // the 2nd part is the length of the field
		if parts[1] != "" {
			if length, err = strconv.Atoi(parts[1]); err != nil {
				return -1, -1, newInvalidRelativeLengthError(err)
			}
		}
	}

	return abs, length, err
}

// Finds and returns the precision from the annotation list. The default value is '-1' which is a valid value for strconv.FormatFloat.
// Gives an error if the value is not a valid integer that's bigger than -1.
func getPrecisionFromAnnotation(annotationList []string) (int, error) {

	for i, precisionAnnotation := range annotationList {
		if strings.HasPrefix(annotationList[i], "precision") {

			var vals = strings.Split(precisionAnnotation, ":")

			if len(vals) != 2 {
				return -1, nil
			}

			// -1 is a valid precision - means all digits needed
			if precision, err := strconv.Atoi(vals[1]); err == nil && precision >= -1 {
				return precision, nil
			}

			return -1, newInvalidPrecisionError(vals[1])
		}
	}

	return -1, nil
}
