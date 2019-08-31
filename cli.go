package main

import (
	"flag"
	"fmt"
	. "github.com/Al2Klimov/go-monplug-utils"
	js "github.com/robertkrimen/otto"
	"regexp"
	"strings"
)

type thresholds map[string]OptionalThreshold

var _ flag.Value = thresholds(nil)

func (r thresholds) String() string {
	var out []string = nil

	for k, v := range r {
		out = append(out, fmt.Sprintf("%s(%s)", k, v.String()))
	}

	return strings.Join(out, " ")
}

var cliThreshold = regexp.MustCompile(`\A(\w+)\((.+)\)\z`)

func (r thresholds) Set(raw string) error {
	match := cliThreshold.FindStringSubmatch(raw)
	if match == nil {
		return badThreshold(raw)
	}

	var actualThreshold OptionalThreshold
	if actualThreshold.Set(match[2]) != nil {
		return badThreshold(raw)
	}

	r[match[1]] = actualThreshold

	return nil
}

type assertion struct {
	raw      string
	compiled *js.Script
}

type assertions []assertion

var _ flag.Value = (*assertions)(nil)

func (a *assertions) String() string {
	var out []string = nil

	for _, assertion := range *a {
		out = append(out, assertion.raw)
	}

	return strings.Join(out, " ")
}

var vm = js.New()

func (a *assertions) Set(raw string) error {
	compiled, errJs := vm.Compile("<cli>", raw)
	if errJs != nil {
		return badAssertion{raw, errJs}
	}

	*a = append(*a, assertion{raw, compiled})

	return nil
}
