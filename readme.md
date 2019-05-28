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
	"os"

	"github.com/daniellowtw/matlab"
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
	fmt.Println(file.GetVarsNames()) // prints the variables in the mat file
	matrix, _ := file.GetVar("a")
	// convenient method to marshal into go types
	var _ []int64 = matrix.IntArray()
	floatMatrix, _ := file.GetVar("b")
	// convenient method to marshal into go types
	var _ []float64 = floatMatrix.DoubleArray()
}
```

# Matlab with cells

A CellMatrix is a matrix where the values are matrices. The `GetAtLocation` method allows indexing into the values array. A convenience method `String()` on a Matrix is available to convert the CharArray matrix into a string.

Example:
```go
cellMatrix, _ := file.GetVar("a")
firstElement := cellMatrix.GetAtLocation(0).(*Matrix).String()
SecondElement := cellMatrix.GetAtLocation(1).(*Matrix).DoubleArray()
```

# Matrix with struct

Suppose we create a struct with the following:
```matlab
X.w = [1];
# or X.w = 1

X.y = [2];
X.z = ["abc"];
# or X.z = "abc"
```

We can read it as follows:
```go
structMatrix, _ := file.GetVar("X")
w := cellMatrix.Struct()["w"].GetAtLocation(0) // float64(1)
z := cellMatrix.Struct()["w"].String() // "abc"
```

# TODO

- Support sparse array class within miMatrix parser
- Support object class within miMatrix parser
- Support writing to mat file
