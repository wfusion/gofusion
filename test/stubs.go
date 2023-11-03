package test

import (
	_ "unsafe"
)

//go:linkname MethodFilter github.com/stretchr/testify/suite.methodFilter
func MethodFilter(name string) (bool, error)
