# go-binfile
Libarary for scanning binary transmissions

### Install
```go get github.com/DRK-Blutspende-BaWueHe/go-binfile```

## Features
  * Unmarshalling byte-arrays with annotated structs
  * Datatypes: string, float32, float64, int

## Usage
Annotate your structure and then unmarshal using the library to map the values
```
type TestBinaryStructure1 struct {
	RecordType          string `bin:":2"`      // 0
	UnitNo              int    `bin:":2"`      // 2
	RackNumber          int    `bin:":4"`      // 4
	CupPosition         int    `bin:":2"`      // 8
	SampleType          string `bin:":1,trim"` // 10
	SampleNo            string `bin:":4"`      // 11
	SampleId            string `bin:":11"`     // 15
	Dummy               string `bin:":4,trim"` // 26
	BlockIdentification string `bin:":1,trim"` // 30

	TestResults []struct {
		TestCode   string `bin:":2"`            // 1. 30  2. 43
		TestResult string `bin:":9,trim"`       // 1. 32  2. 45
		Flags      string `bin:":2,trim"`       // 1. 41  2. 54
		Terminator string `bin:":1,terminator"` // 1.     2. 56
	}
}

func main() {
    ...
    data := "D 03116506 044760722905768    E61     6.40  62      935  "
	
    var r TestBinaryStructure1
	err := binfile.Unmarshal([]byte(data), &r, binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")
    ...
}
```

## Arrays with variable length and terminator
Arrays can be encoded by indicating the end with a symbol of some kind, we call that "terminator". If the string-field that is anotated with "termiantor" equals the terminator from the call, here \r, then the field is read and the byte is used and the loop ends. 

## Arrays with fixed length
Not implemented yet

    