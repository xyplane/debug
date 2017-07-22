package debugger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var mtx sync.Mutex
var includes []*regexp.Regexp
var excludes []*regexp.Regexp
var output io.Writer = os.Stderr
var last = time.Now()

type outputType int

const (
	printType outputType = iota
	printfType
	printlnType
	enabledType
	nameType
)

func init() {
	parse(os.Getenv("DEBUG"))
}

var newlineSuffix = []byte("\n")

var formatRegExp = regexp.MustCompile("%[0-9.+\\-\\[\\]#]*[vTt%bcdoqUespEfFgGxX]")

type Logger func(a ...interface{})

func (log Logger) Ln(a ...interface{}) {
	log.Println(a...)
}

func (log Logger) F(format string, a ...interface{}) {
	log.Printf(format, a...)
}

func (log Logger) Print(a ...interface{}) {
	args := [10]interface{}{printType}
	log(append(args[1:], a...)...)
}

func (log Logger) Println(a ...interface{}) {
	args := [10]interface{}{printlnType}
	log(append(args[1:], a...)...)
}

func (log Logger) Printf(format string, a ...interface{}) {
	args := [10]interface{}{printfType}
	log(append(args[1:], a...)...)
}

func (log Logger) Enabled() bool {
	var enabled bool
	log(enabledType, &enabled)
	return enabled
}

func (log Logger) Name() string {
	var name string
	log(nameType, &name)
	return name
}

func (log Logger) Child(name string) Logger {
	return Debug(log.Name() + ":" + name)
}

func handle(name string, enabled bool, a ...interface{}) {
	if len(a) == 0 {
		writeln(name, enabled)
		return
	}

	switch cmd := a[0].(type) {
	case outputType:
		switch cmd {
		case printType:
			fallthrough
		case printlnType:
			writeln(name, enabled, a[1:]...)
		case printfType:
			if format, ok := a[1].(string); ok {
				writef(name, enabled, format, a[2:]...)
			} else {
				panic("")
			}
		case enabledType:
			if result, ok := a[1].(*bool); ok {
				*result = enabled
			} else {
				panic("")
			}
		case nameType:
			if result, ok := a[1].(*string); ok {
				*result = name
			} else {
				panic("")
			}
		}
	case string:
		if formatRegExp.MatchString(cmd) {
			writef(name, enabled, cmd, a[1:]...)
		} else {
			writeln(name, enabled, a...)
		}
	default:
		writeln(name, enabled, a...)
	}
}

func writeln(name string, enabled bool, a ...interface{}) {
	if !enabled {
		return
	}
	mtx.Lock()
	defer mtx.Unlock()
	buf := &bytes.Buffer{}
	prefix(buf, name)
	fmt.Fprintln(buf, a...)
	output.Write(buf.Bytes())
}

func writef(name string, enabled bool, format string, a ...interface{}) {
	if !enabled {
		return
	}
	mtx.Lock()
	defer mtx.Unlock()
	buf := &bytes.Buffer{}
	prefix(buf, name)
	fmt.Fprintf(buf, format, a...)
	if !bytes.HasSuffix(buf.Bytes(), newlineSuffix) {
		buf.Write(newlineSuffix)
	}
	output.Write(buf.Bytes())
}

func prefix(w io.Writer, name string) {
	now := time.Now()
	delta, unit := int64(now.Sub(last)/time.Millisecond), "ms"
	if delta > 1000 {
		delta, unit = delta/1000, "s"
	}
	last = now
	fmt.Fprintf(w, "  %s +%d%s: ", name, delta, unit)
}

func parse(spec string) {
	mtx.Lock()
	defer mtx.Unlock()

	for _, pat := range strings.Split(spec, ",") {
		exclude := false
		if strings.HasPrefix(pat, "-") {
			pat = strings.TrimPrefix(pat, "-")
			exclude = true
		}

		pat = regexp.QuoteMeta(pat)
		pat = strings.Replace(pat, "\\*", ".*", -1)
		re := regexp.MustCompile("^" + pat + "$")

		if exclude {
			excludes = append(excludes, re)
		} else {
			includes = append(includes, re)
		}
	}
}

func Debug(name string) Logger {
	match := false
	for _, re := range includes {
		if re.MatchString(name) {
			match = true
			break
		}
	}

	if match {
		for _, re := range excludes {
			if re.MatchString(name) {
				match = false
				break
			}
		}
	}

	return Logger(func(a ...interface{}) {
		handle(name, match, a...)
	})
}
