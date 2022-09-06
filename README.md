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
type DataMessage struct {
	RecordType          string `bin:":2"`      
	UnitNo              int    `bin:":2"`      
	RackNumber          int    `bin:":4"`      
	CupPosition         int    `bin:":2"`      
	SampleType          string `bin:":1,trim"` 
	SampleNo            string `bin:":4"`      
	SampleId            string `bin:":11"`     
	Dummy               string `bin:":4,trim"` 
	BlockIdentification string `bin:":1,trim"` 

	TestResults []struct {
		TestCode   string `bin:":2"`            
		TestResult string `bin:":9,trim"`       
		Flags      string `bin:":2,trim"`       
		Terminator string `bin:":1,terminator"` 
	}
}

func main() {
    ...
    data := "D 03116506 044760722905768    E61     6.40  62      935  \r"
	
    var r DataMessage
	err := binfile.Unmarshal([]byte(data), &r, binfile.EncodingUTF8, binfile.TimezoneUTC, "\r")
    ...
}
```

## Arrays with variable length and terminator
Arrays can be encoded by indicating the end with a symbol of some kind, we call that "terminator". If the string-field that is anotated with "termiantor" equals the terminator from the call, here \r, then the field is read and the byte is used and the loop ends. 

## Arrays with fixed length
Not implemented yet

    