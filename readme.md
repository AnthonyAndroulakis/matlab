# Matlab

Apparently a parser for matlab 5.0 doesn't exist in go. So, here we are.

The matlab format can be found in this [pdf](https://www.mathworks.com/help/pdf_doc/matlab/matfile_format.pdf).

This is still very much a work in progress!

[![Go Report Card](https://goreportcard.com/badge/github.com/daniellowtw/matlab)](https://goreportcard.com/report/github.com/daniellowtw/matlab)
[![Travis CI](https://travis-ci.org/daniellowtw/matlab.svg?branch=master)](https://travis-ci.org/daniellowtw/matlab.svg?branch=master)

# Example usage

```go
package main

import (
	"fmt"
	"github.com/daniellowtw/matlab"
	"github.com/davecgh/go-spew/spew"
	"os"
)

func main() {
	f, err := os.Open("example.mat")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	file, err := matlab.NewFileFromReader(f)
	if err != nil {
		panic(err)
	}
	elements, err := file.ReadAllElements()
	if err != nil {
		fmt.Println("Err reading elements ")
		panic(err)
	}
	fmt.Printf("Number of elements read: %+v\n", len(elements))
	spew.Dump(elements)
}
```

# TODO

- Support sparse array class within miMatrix parser
- Support structure class within miMatrix parser
- Support cell class within miMatrix parser
- Support object class within miMatrix parser
- Support writing to mat file
