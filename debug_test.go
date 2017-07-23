package debugger

import (
	"bytes"
	"regexp"
	"testing"
	"time"
)

func TestDebugger(t *testing.T) {
	pattern := "test:other,test:child*,-test:child2"
	buffer := &bytes.Buffer{}
	output = buffer

	parse(pattern)

	debug := Debug("test")
	if debug.Name() != "test" {
		t.Fatalf("Logger name expecting 'test' found '%s'", debug.Name())
	}
	if debug.Enabled() {
		t.Fatalf("Logger '%s' is enabled for spec: '%s'", debug.Name(), pattern)
	}

	debug("Message for disabed logger")
	if len(buffer.Bytes()) > 0 {
		t.Fatalf("Logger '%s' has output when disabled: %s", debug.Name(), string(buffer.Bytes()))
	}

	other := debug.Child("other")
	if other.Name() != "test:other" {
		t.Fatalf("Logger name expecting 'test:other' found '%s'", other.Name())
	}
	if !other.Enabled() {
		t.Fatalf("Logger '%s' is disabled for spec: '%s'", other.Name(), pattern)
	}

	other("Message for enabled logger")
	if ok, _ := regexp.Match("  test:other \\+0ms: Message for enabled logger\n", buffer.Bytes()); !ok {
		t.Fatalf("Logger '%s' unexpected output: '%s'", other.Name(), string(buffer.Bytes()))
	}
	buffer.Reset()

	child1 := debug.Child("child1")
	if child1.Name() != "test:child1" {
		t.Fatalf("Logger name expecting 'test:child1' found '%s'", child1.Name())
	}
	if !child1.Enabled() {
		t.Fatalf("Logger '%s' is disabled for spec: '%s'", child1.Name(), pattern)
	}

	time.Sleep(15 * time.Millisecond)

	child1("Message for %s logger", child1.Name())
	if ok, _ := regexp.Match("  test:child1 \\+1\\dms: Message for test:child1 logger\n", buffer.Bytes()); !ok {
		t.Fatalf("Logger '%s' unexpected output: '%s'", child1.Name(), string(buffer.Bytes()))
	}
	buffer.Reset()

	time.Sleep(105 * time.Millisecond)

	child1.F("Message for %s logger", child1.Name())
	if ok, _ := regexp.Match("  test:child1 \\+10\\dms: Message for test:child1 logger\n", buffer.Bytes()); !ok {
		t.Fatalf("Logger '%s' unexpected printf output: '%s'", child1.Name(), string(buffer.Bytes()))
	}
	buffer.Reset()

	time.Sleep(1100 * time.Millisecond)

	child1.Ln("Message", "for", "test:child1", "logger")
	if ok, _ := regexp.Match("  test:child1 \\+1s: Message for test:child1 logger\n", buffer.Bytes()); !ok {
		t.Fatalf("Logger '%s' unexpected println output: '%s'", child1.Name(), string(buffer.Bytes()))
	}
	buffer.Reset()

	child2 := debug.Child("child2")
	if child2.Name() != "test:child2" {
		t.Fatalf("Logger name expecting 'test:child2' found '%s'", child2.Name())
	}
	if child2.Enabled() {
		t.Fatalf("Logger '%s' is enabled for spec: '%s'", child2.Name(), pattern)
	}
}
