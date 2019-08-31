package main

import "fmt"

type badThreshold string

var _ error = badThreshold("")

func (b badThreshold) Error() string {
	return fmt.Sprintf("bad threshold: %q", string(b))
}

type badAssertion struct {
	raw    string
	reason error
}

var _ error = badAssertion{}

func (b badAssertion) Error() string {
	return fmt.Sprintf("bad assertion %q: %s", b.raw, b.reason.Error())
}
