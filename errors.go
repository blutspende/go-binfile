package binfile

import (
	"fmt"
	"reflect"
)

// An ErrorUnsupportedType is returned when the processed field type is not supported by the implementation.
type ErrorUnsupportedType struct {
	InvalidType reflect.Type
}

func (e *ErrorUnsupportedType) Error() string {
	return fmt.Sprintf("unsupported type '%s'", e.InvalidType.Name())
}

func (e *ErrorUnsupportedType) Is(target error) bool {
	_, ok := target.(*ErrorUnsupportedType)
	return ok
}

func newUnsupportedTypeError(invalidType reflect.Type) error {
	return &ErrorUnsupportedType{InvalidType: invalidType}
}

// An ErrorAnnotatedFieldNotWritable is returned when the processed field is annotated but not exported.
var ErrorAnnotatedFieldNotWritable = fmt.Errorf("annotated field is not writable")

// An ErrorExportedFieldNotAnnotated is returned when the processed field is exported but not unnotated.
var ErrorExportedFieldNotAnnotated = fmt.Errorf("field is not exported but annotated")

// An ErrorFoundZeroValueBytes is returned when the sum of the processed bytes is zero.
var ErrorFoundZeroValueBytes = fmt.Errorf("specified range is all zero value bytes")

// An ErrorInvalidPrecision is returned when the provided precision value is
// an invalid integer or is lower than -1.
type ErrorInvalidPrecision struct {
	Precision string
}

func (e *ErrorInvalidPrecision) Error() string {
	return fmt.Sprintf("invalid precision given '%s'", e.Precision)
}

func (e *ErrorInvalidPrecision) Is(target error) bool {
	_, ok := target.(*ErrorInvalidPrecision)
	return ok
}

func newInvalidPrecisionError(precision string) error {
	return &ErrorInvalidPrecision{Precision: precision}
}

// An ErrorInvalidAddressAnnotation is returned when an error happens
// while processing the address annotation.
type ErrorInvalidAddressAnnotation struct {
	Err error
}

func (e *ErrorInvalidAddressAnnotation) Error() string {
	return fmt.Sprintf("invalid address annotation: %s", e.Err.Error())
}

func (e *ErrorInvalidAddressAnnotation) Is(target error) bool {
	_, ok := target.(*ErrorInvalidAddressAnnotation)
	return ok
}

func (e *ErrorInvalidAddressAnnotation) Unwrap() error {
	return e.Err
}

func newInvalidAddressAnnotationError(err error) error {
	return &ErrorInvalidAddressAnnotation{Err: err}
}

// An ErrorInvalidAbsolutePosition is returned when the
// address annotation's absolute position value is an invalid integer.
type ErrorInvalidAbsolutePosition struct {
	Err error
}

func (e *ErrorInvalidAbsolutePosition) Error() string {
	return fmt.Sprintf("invalid absolute position value: %s", e.Err.Error())
}

func (e *ErrorInvalidAbsolutePosition) Is(target error) bool {
	_, ok := target.(*ErrorInvalidAbsolutePosition)
	return ok
}

func (e *ErrorInvalidAbsolutePosition) Unwrap() error {
	return e.Err
}

func newInvalidAbsolutePositionError(err error) error {
	return &ErrorInvalidAbsolutePosition{Err: err}
}

// An ErrorInvalidRelativeLength is returned when the
// address annotation's relative length value is an invalid integer.
type ErrorInvalidRelativeLength struct {
	Err error
}

func (e *ErrorInvalidRelativeLength) Error() string {
	return fmt.Sprintf("invalid relative length value: %s", e.Err.Error())
}

func (e *ErrorInvalidRelativeLength) Is(target error) bool {
	_, ok := target.(*ErrorInvalidRelativeLength)
	return ok
}

func (e *ErrorInvalidRelativeLength) Unwrap() error {
	return e.Err
}

func newInvalidRelativeLengthError(err error) error {
	return &ErrorInvalidRelativeLength{Err: err}
}

// An ErrorIntConversionOverflow is returned when you try to convert a 64 bit value on a 32 bit system.
var ErrorIntConversionOverflow = fmt.Errorf("int conversion overflow 32 vs 64 bit system")

// An ErrorInvalidSizeForArray is returned when an invalid value found for array size - probably a negative value.
type ErrorInvalidSizeForArray struct {
	ArraySize int
}

func (e *ErrorInvalidSizeForArray) Error() string {
	return fmt.Sprintf("invalid size for array '%d'", e.ArraySize)
}

func (e *ErrorInvalidSizeForArray) Is(target error) bool {
	_, ok := target.(*ErrorInvalidSizeForArray)
	return ok
}

func newInvalidSizeForArrayError(size int) error {
	return &ErrorInvalidSizeForArray{ArraySize: size}
}

// An ErrorUnknownFieldName is returned when a struct doesn't have the searched field name.
var ErrorUnknownFieldName = fmt.Errorf("unknown field name")

// An ErrorInvalidDynamicArraySize is returned when something went wrong while resolving the array size.
// Check the underlying error for more information!
type ErrorInvalidDynamicArraySize struct {
	StructName string
	FieldName  string
	Err        error
}

func (e *ErrorInvalidDynamicArraySize) Error() string {
	return fmt.Sprintf("error getting dynamic array size: %s", e.Err.Error())
}

func (e *ErrorInvalidDynamicArraySize) Is(target error) bool {
	_, ok := target.(*ErrorInvalidDynamicArraySize)
	return ok
}

func (e *ErrorInvalidDynamicArraySize) Unwrap() error {
	return e.Err
}

func newInvalidDynamicArraySizeError(structName string, fieldName string, err error) error {
	return &ErrorInvalidDynamicArraySize{StructName: structName, FieldName: fieldName, Err: err}
}

// An ErrorInvalidValueLength is returned when the converted value is larger than the place it needs to fit.
type ErrorInvalidValueLength struct {
	Length int
	Value  string
}

func (e *ErrorInvalidValueLength) Error() string {
	return fmt.Sprintf("invalid value length '%s', (%d)", e.Value, e.Length)
}

func (e *ErrorInvalidValueLength) Is(target error) bool {
	_, ok := target.(*ErrorInvalidValueLength)
	return ok
}

func newInvalidValueLengthError(value string, length int) error {
	return &ErrorInvalidValueLength{Value: value, Length: length}
}

// An ErrorInvalidOffset is returned when the annotated absolute position...
// is lower than the current position in the byte array.
type ErrorInvalidOffset struct {
	CurrentPos int
	PointedPos int
}

func (e *ErrorInvalidOffset) Error() string {
	return fmt.Sprintf("absolute position points backwards to '%d' - current position is '%d'", e.PointedPos, e.CurrentPos)
}

func (e *ErrorInvalidOffset) Is(target error) bool {
	_, ok := target.(*ErrorInvalidOffset)
	return ok
}

func newInvalidInvalidOffsetError(currentPos, pointedPos int) error {
	return &ErrorInvalidOffset{CurrentPos: currentPos, PointedPos: pointedPos}
}

// An ErrorReadingOutOfBounds is returned when the unmarshal tries to read more than the remaining data length.
type ErrorReadingOutOfBounds struct {
	FromPos   int
	ToPos     int
	MaxLength int
}

func (e *ErrorReadingOutOfBounds) Error() string {
	return fmt.Sprintf("reading out of bounds from position '%d' to '%d' (%d bytes) in input data of '%d' bytes", e.FromPos, e.ToPos, e.ToPos-e.FromPos, e.MaxLength)
}

func (e *ErrorReadingOutOfBounds) Is(target error) bool {
	_, ok := target.(*ErrorReadingOutOfBounds)
	return ok
}

func newReadingOutOfBoundsError(fromPos, toPos, maxLength int) error {
	return &ErrorReadingOutOfBounds{FromPos: fromPos, ToPos: toPos, MaxLength: maxLength}
}

// An ErrorProcessingField is returned when something went wrong while processing the field.
// Check the underlying error for more information!
type ErrorProcessingField struct {
	FieldName   string
	Annotations string
	Err         error
}

func (e *ErrorProcessingField) Error() string {
	return fmt.Sprintf("error processing field '%s' `%s`: %s", e.FieldName, e.Annotations, e.Err.Error())
}

func (e *ErrorProcessingField) Is(target error) bool {
	_, ok := target.(*ErrorProcessingField)
	return ok
}

func (e *ErrorProcessingField) Unwrap() error {
	return e.Err
}

func newProcessingFieldError(fieldName string, annotations string, err error) error {
	return &ErrorProcessingField{FieldName: fieldName, Annotations: annotations, Err: err}
}

// An ErrorMissingAddressAnnotation is returned when a non-struct field is missing the address annotation.
var ErrorMissingAddressAnnotation = fmt.Errorf("non-struct field must have address annotation")

// An ErrorMissingArrayAnnotation is returned when an array field is missing the 'array' annotation.
var ErrorMissingArrayAnnotation = fmt.Errorf("array fields must have an 'array' annotation")
