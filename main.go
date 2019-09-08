//go:generate go run vendor/github.com/Al2Klimov/go-gen-source-repos/main.go github.com/Al2Klimov/check_systemd_unit

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	_ "github.com/Al2Klimov/go-gen-source-repos"
	. "github.com/Al2Klimov/go-monplug-utils"
	"github.com/coreos/go-systemd/dbus"
	js "github.com/robertkrimen/otto"
	"html"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func main() {
	os.Exit(ExecuteCheck(onTerminal, checkSystemdUnit))
}

func onTerminal() (output string) {
	return fmt.Sprintf(
		"For the terms of use, the source code and the authors\n"+
			"see the projects this program is assembled from:\n\n  %s\n",
		strings.Join(GithubcomAl2klimovGo_gen_source_repos, "\n  "),
	)
}

func checkSystemdUnit() (output string, perfdata PerfdataCollection, errs map[string]error) {
	cli := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	unit := cli.String("unit", "", "e.g. icinga2.service")

	thresholdsWarn := thresholds{}
	thresholdsCrit := thresholds{}

	var jsWarn, jsCrit assertions

	cli.Var(thresholdsWarn, "warn", "e.g. NRestarts(@~:42)")
	cli.Var(thresholdsCrit, "crit", "e.g. NRestarts(@~:42)")
	cli.Var(&jsWarn, "js-warn", `e.g. ActiveState==="active"`)
	cli.Var(&jsCrit, "js-crit", `e.g. ActiveState==="active"`)

	if errCli := cli.Parse(os.Args[1:]); errCli != nil {
		os.Exit(3)
	}

	if *unit == "" {
		fmt.Fprintln(os.Stderr, "-unit missing")
		cli.Usage()
		os.Exit(3)
	}

	dBus, errConn := dbus.New()
	if errConn != nil {
		errs = map[string]error{"D-Bus": errConn}
		return
	}

	defer dBus.Close()

	props, errProps := dBus.GetAllProperties(*unit)
	if errProps != nil {
		errs = map[string]error{"D-Bus": errProps}
		return
	}

	vm := js.New()

	for k, v := range props {
		if num, signed, isNum := number2float(v); isNum {
			var uom string
			if strings.HasSuffix(k, "USec") {
				uom = "us"
			}

			perfdata = append(perfdata, Perfdata{
				Label: k,
				UOM:   uom,
				Value: num,
				Warn:  thresholdsWarn[k],
				Crit:  thresholdsCrit[k],
				Min:   OptionalNumber{!signed, 0},
			})

			delete(thresholdsWarn, k)
			delete(thresholdsCrit, k)
		}

		vm.Set(k, v)
	}

	if len(thresholdsWarn) > 0 || len(thresholdsCrit) > 0 {
		perfdata = nil

		nosuch := map[string]struct{}{}
		nonum := map[string]struct{}{}

		for _, t := range [2]thresholds{thresholdsWarn, thresholdsCrit} {
			for k := range t {
				if _, hasProp := props[k]; hasProp {
					nonum[k] = struct{}{}
				} else {
					nosuch[k] = struct{}{}
				}
			}
		}

		errs = make(map[string]error, len(nosuch)+len(nonum))

		if len(nosuch) > 0 {
			err := errors.New("no such unit property")

			for k := range nosuch {
				errs[fmt.Sprintf("threshold %q", k)] = err
			}
		}

		for k := range nonum {
			v := props[k]
			typ := reflect.TypeOf(v)
			var typeName string

			if typ == nil {
				typeName = "nil"
			} else {
				typeName = typ.Name()
			}

			errs[fmt.Sprintf("threshold %q", k)] = fmt.Errorf("not a number: %s(%#v)", typeName, v)
		}

		return
	}

	var failedJs []string = nil
	var jsWarnFailed uint64 = 0
	var jsCritFailed uint64 = 0

	for _, jw := range jsWarn {
		if res, errJs := vm.Eval(jw.compiled); errJs != nil {
			if errs == nil {
				errs = map[string]error{}
			}

			errs[fmt.Sprintf("script %q", jw.raw)] = errJs
		} else if boool, errBool := res.ToBoolean(); errBool != nil || !boool {
			failedJs = append(failedJs, jw.raw)
			jsWarnFailed++
		}
	}

	for _, jc := range jsCrit {
		if res, errJs := vm.Eval(jc.compiled); errJs != nil {
			if errs == nil {
				errs = map[string]error{}
			}

			errs[fmt.Sprintf("script %q", jc.raw)] = errJs
		} else if boool, errBool := res.ToBoolean(); errBool != nil || !boool {
			failedJs = append(failedJs, jc.raw)
			jsCritFailed++
		}
	}

	if errs != nil {
		perfdata = nil
		return
	}

	var out bytes.Buffer

	if len(failedJs) > 0 {
		out.Write([]byte(`<p><b>Some JS assertions failed:</b></p><ul>`))

		for _, fj := range failedJs {
			out.Write([]byte(`<li style="font-family: monospace; white-space: pre;">`))
			out.Write([]byte(html.EscapeString(fj)))
			out.Write([]byte(`</li>`))
		}

		out.Write([]byte(`</ul>`))
	}

	var failedThresholds []*Perfdata = nil

	for i := range perfdata {
		if pd := &perfdata[i]; pd.GetStatus() != Ok {
			failedThresholds = append(failedThresholds, pd)
		}
	}

	perfdata = append(perfdata, Perfdata{
		Label: "js_warn",
		Value: float64(jsWarnFailed),
		Warn:  OptionalThreshold{true, false, 0, 0},
		Min:   OptionalNumber{true, 0},
		Max:   OptionalNumber{true, float64(len(jsWarn))},
	})

	perfdata = append(perfdata, Perfdata{
		Label: "js_crit",
		Value: float64(jsCritFailed),
		Crit:  OptionalThreshold{true, false, 0, 0},
		Min:   OptionalNumber{true, 0},
		Max:   OptionalNumber{true, float64(len(jsCrit))},
	})

	if failedThresholds != nil {
		sort.Slice(failedThresholds, func(i, j int) bool {
			lhs := failedThresholds[i]
			rhs := failedThresholds[j]

			lhss := lhs.GetStatus()
			rhss := rhs.GetStatus()

			if lhss == rhss {
				return lhs.Label < rhs.Label
			} else {
				return lhss > rhss
			}
		})

		out.Write([]byte(`<p><b>Some threshold assertions failed:</b></p><ul>`))

		colors := map[PerfdataStatus]string{Warning: "770", Critical: "700"}

		for _, ft := range failedThresholds {
			fmt.Fprintf(
				&out,
				`<li style="color: #%s;">%s = %s</li>`,
				colors[ft.GetStatus()],
				ft.Label,
				strconv.FormatFloat(ft.Value, 'f', -1, 64),
			)
		}

		out.Write([]byte(`</ul>`))
	}

	if out.Len() < 1 {
		out.Write([]byte(`<p style="color: #070;">OK</p>`))
	}

	output = out.String()
	return
}

func number2float(in interface{}) (out float64, signed, ok bool) {
	switch num := in.(type) {
	case uint:
		return float64(num), false, true
	case uint8:
		return float64(num), false, true
	case uint16:
		return float64(num), false, true
	case uint32:
		return float64(num), false, true
	case uint64:
		return float64(num), false, true
	case int:
		return float64(num), true, true
	case int8:
		return float64(num), true, true
	case int16:
		return float64(num), true, true
	case int32:
		return float64(num), true, true
	case int64:
		return float64(num), true, true
	case float32:
		return float64(num), true, true
	case float64:
		return num, true, true
	default:
		return 0, false, false
	}
}
