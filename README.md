# go-binfile
Libarary for scanning binary transmissions

### Install
```go get github.com/blutspende/go-binfile```

## Features
  - Unmarshalling byte-arrays with annotated structs
  - Marshaling annotated structs to byte-arrays
  - Datatypes: string, float32, float64, int

## Usage
Annotate your structure and then unmarshal using the library to map the values
```
type DataMessage struct {
	RecordType          string `bin:":2"`      
	UnitNo              int    `bin:":2"`      
	RackNumber          int    `bin:":4"`      
	CupPosition         int    `bin:":2"`      
	SampleType          string `bin:":1,trim"` 
	SampleNo            string `bin:":4"`      
	SampleId            string `bin:":11"`
	BlockIdentification string `bin:":1,trim"` 

	TestResults []TestResultsStructure `bin:"array:terminator"`
}

type TestResultsStructure struct {
	TestCode   string `bin:":2"`            
	TestResult string `bin:":9,trim"`       
	Flags      string `bin:":2,trim"`       
}

func main() {
	...

	data := []byte("D 03116506 044760722905768E61     6.40  62      935  \r")
	
	var result DataMessage
	position, err := binfile.Unmarshal(data, &result, binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")

	// 'data' is parsed into 'result'

	...
}
```

You can turn a filled annotated struct into a byte array.

```
	...

	var inputData = DataMessage{
		RecordType:          "D ",
		UnitNo:              3,
		RackNumber:          1165,
		CupPosition:         6,
		SampleType:          "",
		SampleNo:            "0447",
		SampleId:            "60722905768",
		BlockIdentification: "E",
		TestResults: []TestResultsStructure{
			{
				TestCode:   "61",
				TestResult: "6.40",
				Flags:      "",
			},
			{
				TestCode:   "62",
				TestResult: "935",
				Flags:      "",
			},
		},
	}

	data, err := Marshal(inputData, ' ', binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")

	// 'data' is []byte("D 03116506 044760722905768E61     6.40  62      935  \r")

	...
```

## Annotation

Annotations for the fields lie in the 'bin' tag and are separated by comma characters. All fields - except for nested structs - must be annotated. The converter only processes the exported fields.

There are different required and optional formatting annotations available for the supported types. They are listed in the formatting summary below.

## Primitive types

`` `bin:"<absolute_position>:<relative_length>"` ``

This annotation is required for every non-struct type. It describes the size and position of the data in the byte array.

The absolute position is counted from the start of the current structure. It is best used for leaving gaps, which will be filled with the provided filler byte. And is optional to have.

Providing the relative length is a must. It tells the converter exactly how many bytes long the value is in the byte array. Every primitive type has its own rules for formatting. 

### Integer

The sign is always the first and takes up 1 byte of space from the specified amount. By default, only the negative sign is explicitly added. If you want to specifically add the '+' sign, then use the ``forcesign`` annotation. This doesn't have an effect on unmarshaling.

Then a '0' padded integer number's digits take up the rest. It must fully fit in the specified space. The default zero padding can be changed to spaces by using the ``padspace`` annotation.

### Float32 / Float64

A float type is handled similarly to an integer and the ``forcesign`` and ``padspace`` annotations also work. The decimal point takes up a byte.

A **Float64** differs from a Float32 as it only uses scientific notation. *'-d.ddddE±dd'* The exponent also needs to fit into the specified space.

There is no automatic truncation, but you can optionally have the below annotation.

``precision:<num_decimal_digits>``

The above annotation accepts an integer above -1 to round the floating point number on conversion expressly. This doesn't affect unmarshaling, as it would cause accidental data loss.

### String

Currently, there is no special support for other character encodings than UTF-8.

There is no support for string truncation, but will be padded with spaces before the value if needed.

To accompany this, there is a convenient ``trim`` annotation that can be added to the field. It will remove trailing spaces from the read value.

## Arrays

If an array contains a primitive type, it also must have the generic absolute position and relative length annotation. Which will be applied to all elements as described above.

There are multiple kinds of supported array formats, as they are listed below.

### Arrays with variable length and terminator

`` `bin:"array:terminator"` ``

Arrays can be encoded by only indicating the end with a symbol of some kind, we call that *"terminator"*. It is optional to have a terminator at the end of the byte array in unmarshaling.

### Arrays with fixed length

`` `bin:"array:<array_size>"` ``

Arrays can have a fixed size which needs to be provided in the annotation as an integer value.

If your data set is smaller than the specified size, the rest will be padded with zero values. When reading back a fixed size array data encounters a zero byte padded space of the required size, it will not create the extra empty entries in the array.

If your data set is bigger than the specified array size, the array will be truncated and a "terminator" is attached at the end.

### Arrays with dynamic size

`` `bin:"array:<field_name_with_size>"` ``

There is an option to provide the field name which is in the same struct and contains a valid array size integer. In this case, the array will be handled like the fixed size one but the size will be read from the provided field.

This has a limitation on unmarshaling, that the provided field should come before the array. 

## Top-level arrays

Besides structs, this implementation supports top-level arrays for processing multiple messages of the same kind in the same byte array. The messages need to be separated by a *"terminator"*.

Currently there is no support for nested arrays. The arrays must contain annotated structs.
