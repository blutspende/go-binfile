# go-binfile
Libarary for scanning binary transmissions

### Install
```go get github.com/DRK-Blutspende-BaWueHe/go-binfile```

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
    data := "D 03116506 044760722905768    E61     6.40  62      935  \r"
	
    var r DataMessage
	err := binfile.Unmarshal([]byte(data), &r, binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")
    ...
}
```

You can turn a filled annotated struct into a byte array.

```
    ...

	var r TestBinaryStructure1
	r.RecordType = "D "
	r.UnitNo = 3
	r.RackNumber = 1165
	r.CupPosition = 6
	r.SampleType = ""
	r.SampleNo = "0447"
	r.SampleId = "60722905768"
	r.BlockIdentification = "E"
	r.TestResults = []TestResultsStructure{
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
	}

	data, err := Marshal(r, ' ', binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")

    // data is: "D 03116506 044760722905768    E61     6.40  62      935  \r"
    ...
```


## Primitive types

Annotations for the fields lie in the 'bin' tag and are separated by comma characters. All fields - except for nested structs - must be annotated. The converter only processes the exported fields.

`` `bin:"<absolute_position>:<relative_length>"` ``

This annotation is required for every non-struct type. It describes the size and position of the data in the byte array.

The absolute position is counted from the start of the current structure. It is best used for leaving gaps, which will be filled with the provided filler byte. And is optional to have.

Providing the relative length is a must. It tells the converter exactly how many bytes long the value is in the byte array. Every primitive type has its own rules for formatting. 

### Integer

The sign is always the first and takes up 1 byte of space from the specified amount. Currently, only the negative sign is explicitly added.

Then a "0" padded integer number's digits take up the rest. It must fully fit in the specified space.

### Float32 / Float64

A float type is handled similarly to an integer, but the padding occurs after the last digit. The decimal point also takes up a byte.

There is no support for truncation, so the converted value must fit into the provided space. Also, scientific notation is currently not supported.

### String

Currently, there is no support for string truncation but will be padded with spaces before the value if needed.

To accompany this, there is a convenient "trim" annotation that can be added to the field. It will remove trailing spaces from the read value.

## Arrays

If an array contains a primitive type, it also must have the generic absolute position and relative length annotation. Which will be applied to all elements as described above.

### Arrays with variable length and terminator

`` `bin:"array:terminator"` ``

Arrays can be encoded by indicating the end with a symbol of some kind, we call that "terminator". If the string-field that is annotated with "terminator" equals the terminator from the call, here \r, then the field is read and the byte is used and the loop ends. 

### Arrays with fixed length

`` `bin:"array:<array_size>"` ``

Arrays can have a fixed size which needs to be noted in the annotation.

If your data set is smaller than the specified size, the rest will be padded with zero values. When reading back a fixed size array data encounters a zero byte padded space of the required size, it will not create the extra empty entries in the array.

If your data set is bigger than the specified array size, the array will be truncated and a "terminator" is attached at the end.

### Arrays with dynamic size

`` `bin:"array:<field_name_with_size>"` ``

There is an option to provide the field name which is in the same struct and contains a valid array size integer. In this case, the array will be handled like the fixed size one but the size will be read from the provided field.

Currently, this is only supported in the Unmarshal feature. And has a limitation, that the provided field should come before the array. 


