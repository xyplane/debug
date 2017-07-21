package debug

import (
    "os"
    "fmt"
    "time"
    "sync"
    "regexp"
    "strings"
)

var spec = struct {
   sync.Mutex
   initialized bool
   includes []*regexp.Regexp
   excludes []*regexp.Regexp
}{}

var last = struct {
    sync.Mutex
    now time.Time
}{
    now: time.Now(),
}

var output = os.Stderr

type loggerType int

const (
    debugType   loggerType = iota
    debugfType
    debuglnType
)

var placeholder_regexp = regexp.MustCompile("%[0-9.+\\-\\[\\]#]*[vTt%bcdoqUespEfFgGxX]")


type DebugLogger func(a ...interface{})

func (logger DebugLogger) Debug(args ...interface{}) {
    logger(append([]interface{}{ debugType }, args...)...)
}

func (logger DebugLogger) F(format string, args ...interface{}) {
    logger(append([]interface{}{ debugfType, format }, args...)...)
}

func (logger DebugLogger) Debugf(format string, args ...interface{}) {
    logger(append([]interface{}{ debugfType, format }, args...)...)
}

func (logger DebugLogger) Ln(args ...interface{}) {
    logger(append([]interface{} { debuglnType }, args...)...)
}

func (logger DebugLogger) Debugln(args ...interface{}) {
    logger(append([]interface{} { debuglnType }, args...)...)
}


func log(name string, args ...interface{}) {
    last.Lock()
    defer last.Unlock()

    now := time.Now()
    delta := int64(now.Sub(last.now)/time.Millisecond)
    last.now = now

    prefix := fmt.Sprintf("  %s +%dms: ", name, delta)

    if len(args) == 0 {
        fmt.Fprintln(output, prefix)
        return
    }

    fmt.Fprint(output, prefix)

    switch arg := args[0].(type) {
    case loggerType:
        switch arg {
        case debugType:
            fmt.Fprint(output, args[1:]...)
        case debugfType:
            fmt.Fprintf(output, fmt.Sprint(args[1]), args[2:]...)
        case debuglnType:
            fmt.Fprintln(output, args[1:]...)
        }
    case string:
        if placeholder_regexp.MatchString(arg) {
            if strings.HasPrefix(arg, "\n") {
                fmt.Fprintf(output, arg, args[1:]...)
            } else {
                fmt.Fprintf(output, arg+"\n", args[1:]...)
            }
        } else {
            fmt.Fprintln(output, args...)
        }
    default:
        fmt.Fprintln(output, args...)
    } 
}


func initialize() {
    if spec.initialized {
        return
    }

    d := os.Getenv("DEBUG")

    for _, pat := range(strings.Split(d, ",")) {
        
        exclude := false;
        if strings.HasPrefix(pat, "-") {
            pat = strings.TrimPrefix(pat, "-")
            exclude = true
        }

        pat = regexp.QuoteMeta(pat)
        pat = strings.Replace(pat, "\\*", ".*", -1)
        re := regexp.MustCompile("^" + pat + "$")

        if exclude {
            spec.excludes = append(spec.excludes, re)
        } else {
            spec.includes = append(spec.includes, re)
        }
    }

    spec.initialized = true
}


func Debug(name string) DebugLogger {
    spec.Lock()
    defer spec.Unlock()
    initialize()

    noopLogger := DebugLogger(func(a ...interface{}) {})

    match := false
    for _, re := range(spec.includes) {
        if re.MatchString(name) {
            match = true
            break
        }
    }

    if !match {
        return noopLogger
    }

    for _, re := range(spec.excludes) {
        if re.MatchString(name) {
            match = false
            break
        }
    }

    if !match {
        return noopLogger
    }

    return DebugLogger(func(a ...interface{}) { log(name, a...) })
}

