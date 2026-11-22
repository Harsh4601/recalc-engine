package recalc

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// runScenarioFile replays a .scenario file against a fresh engine. The format
// is line-based:
//
//	# comment
//	set <cell> = <literal-or-formula>
//	expect <cell> = <canonical-value>   value of a cell (Value.String())
//	expectf <cell> = <formula-text>     formula text of a cell (Engine.Formula)
//	insertrows <before> <count>         structural edits (0-based indices)
//	deleterows <at> <count>
//	insertcols <before> <count>
//	deletecols <at> <count>
//
// The first '=' separates a set/expect command's target from its argument, so a
// formula (which itself starts with '=') is written as:  set B1 = =A1+A2
func runScenarioFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	e := New()
	sc := bufio.NewScanner(f)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "set "):
			cell, arg, ok := splitCommand(line[len("set "):])
			if !ok {
				t.Fatalf("%s:%d malformed set", filepath.Base(path), ln)
			}
			if err := e.Set(cell, arg); err != nil {
				t.Fatalf("%s:%d set %s: %v", filepath.Base(path), ln, cell, err)
			}
		case strings.HasPrefix(line, "expectf "):
			cell, want, ok := splitCommand(line[len("expectf "):])
			if !ok {
				t.Fatalf("%s:%d malformed expectf", filepath.Base(path), ln)
			}
			if got := e.Formula(cell); got != want {
				t.Errorf("%s:%d formula %s = %q, want %q", filepath.Base(path), ln, cell, got, want)
			}
		case strings.HasPrefix(line, "expect "):
			cell, want, ok := splitCommand(line[len("expect "):])
			if !ok {
				t.Fatalf("%s:%d malformed expect", filepath.Base(path), ln)
			}
			got := e.Get(cell).String()
			if got != want {
				t.Errorf("%s:%d %s = %q, want %q", filepath.Base(path), ln, cell, got, want)
			}
		case strings.HasPrefix(line, "insertrows "):
			a, b := twoInts(t, path, ln, line[len("insertrows "):])
			e.InsertRows(a, b)
		case strings.HasPrefix(line, "deleterows "):
			a, b := twoInts(t, path, ln, line[len("deleterows "):])
			e.DeleteRows(a, b)
		case strings.HasPrefix(line, "insertcols "):
			a, b := twoInts(t, path, ln, line[len("insertcols "):])
			e.InsertCols(a, b)
		case strings.HasPrefix(line, "deletecols "):
			a, b := twoInts(t, path, ln, line[len("deletecols "):])
			e.DeleteCols(a, b)
		default:
			t.Fatalf("%s:%d unknown command: %s", filepath.Base(path), ln, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
}

func twoInts(t *testing.T, path string, ln int, s string) (int, int) {
	t.Helper()
	f := strings.Fields(s)
	if len(f) != 2 {
		t.Fatalf("%s:%d expected two integers, got %q", filepath.Base(path), ln, s)
	}
	a, err1 := strconv.Atoi(f[0])
	b, err2 := strconv.Atoi(f[1])
	if err1 != nil || err2 != nil {
		t.Fatalf("%s:%d bad integers %q", filepath.Base(path), ln, s)
	}
	return a, b
}

func splitCommand(s string) (target, arg string, ok bool) {
	idx := strings.IndexByte(s, '=')
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+1:]), true
}

func runScenarioDir(t *testing.T, dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.scenario"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("no scenarios found in %s", dir)
	}
	for _, p := range files {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) { runScenarioFile(t, p) })
	}
}

// TestScenarios runs every scenario file under testdata/scenarios.
func TestScenarios(t *testing.T) {
	runScenarioDir(t, filepath.Join("testdata", "scenarios", "cases"))
}
