package main

import (
	"os"
)

// os.IsExist is buggy for LinkError in go1; replace
// myIsExist calls with os.IsExist once this fix is well
// released
// https://code.google.com/p/go/source/detail?r=32eb6dac3ff4
func myIsExist(err error) bool {
	if os.IsExist(err) {
		return true
	}
	switch err2 := err.(type) {
	case *os.LinkError:
		return os.IsExist(err2.Err)
	}
	return false
}
