// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/mvdan/sh/syntax"
)

// non-empty string is true, empty string is false
func (r *Runner) bashTest(expr syntax.TestExpr) string {
	switch x := expr.(type) {
	case *syntax.Word:
		return r.loneWord(x)
	case *syntax.ParenTest:
		return r.bashTest(x.X)
	case *syntax.BinaryTest:
		switch x.Op {
		case syntax.TsMatch, syntax.TsNoMatch:
			str := r.loneWord(x.X.(*syntax.Word))
			var buf bytes.Buffer
			yw := x.Y.(*syntax.Word)
			for _, field := range r.wordFields(yw.Parts, false) {
				escaped, _ := escapedGlob(field)
				buf.WriteString(escaped)
			}
			if match(buf.String(), str) == (x.Op == syntax.TsMatch) {
				return "1"
			}
			return ""
		}
		if r.binTest(x.Op, r.bashTest(x.X), r.bashTest(x.Y)) {
			return "1"
		}
		return ""
	case *syntax.UnaryTest:
		if r.unTest(x.Op, r.bashTest(x.X)) {
			return "1"
		}
		return ""
	}
	return ""
}

func (r *Runner) binTest(op syntax.BinTestOperator, x, y string) bool {
	switch op {
	case syntax.TsReMatch:
		re, err := regexp.Compile(y)
		if err != nil {
			r.exit = 2
			return false
		}
		return re.MatchString(x)
	case syntax.TsNewer:
		i1, i2 := stat(r.Dir, x), stat(r.Dir, y)
		if i1 == nil || i2 == nil {
			return false
		}
		return i1.ModTime().After(i2.ModTime())
	case syntax.TsOlder:
		i1, i2 := stat(r.Dir, x), stat(r.Dir, y)
		if i1 == nil || i2 == nil {
			return false
		}
		return i1.ModTime().Before(i2.ModTime())
	case syntax.TsDevIno:
		i1, i2 := stat(r.Dir, x), stat(r.Dir, y)
		return os.SameFile(i1, i2)
	case syntax.TsEql:
		return atoi(x) == atoi(y)
	case syntax.TsNeq:
		return atoi(x) != atoi(y)
	case syntax.TsLeq:
		return atoi(x) <= atoi(y)
	case syntax.TsGeq:
		return atoi(x) >= atoi(y)
	case syntax.TsLss:
		return atoi(x) < atoi(y)
	case syntax.TsGtr:
		return atoi(x) > atoi(y)
	case syntax.AndTest:
		return x != "" && y != ""
	case syntax.OrTest:
		return x != "" || y != ""
	case syntax.TsBefore:
		return x < y
	default: // syntax.TsAfter
		return x > y
	}
}

func stat(dir, name string) os.FileInfo {
	info, _ := os.Stat(filepath.Join(dir, name))
	return info
}

func statMode(dir, name string, mode os.FileMode) bool {
	info := stat(dir, name)
	return info != nil && info.Mode()&mode != 0
}

func (r *Runner) unTest(op syntax.UnTestOperator, x string) bool {
	switch op {
	case syntax.TsExists:
		return stat(r.Dir, x) != nil
	case syntax.TsRegFile:
		info := stat(r.Dir, x)
		return info != nil && info.Mode().IsRegular()
	case syntax.TsDirect:
		return statMode(r.Dir, x, os.ModeDir)
	//case syntax.TsCharSp:
	//case syntax.TsBlckSp:
	case syntax.TsNmPipe:
		return statMode(r.Dir, x, os.ModeNamedPipe)
	case syntax.TsSocket:
		return statMode(r.Dir, x, os.ModeSocket)
	case syntax.TsSmbLink:
		info, _ := os.Lstat(x)
		return info != nil && info.Mode()&os.ModeSymlink != 0
	case syntax.TsSticky:
		return statMode(r.Dir, x, os.ModeSticky)
	case syntax.TsUIDSet:
		return statMode(r.Dir, x, os.ModeSetuid)
	case syntax.TsGIDSet:
		return statMode(r.Dir, x, os.ModeSetgid)
	//case syntax.TsGrpOwn:
	//case syntax.TsUsrOwn:
	//case syntax.TsModif:
	case syntax.TsRead:
		f, err := os.OpenFile(x, os.O_RDONLY, 0)
		if err == nil {
			f.Close()
		}
		return err == nil
	case syntax.TsWrite:
		f, err := os.OpenFile(x, os.O_WRONLY, 0)
		if err == nil {
			f.Close()
		}
		return err == nil
	case syntax.TsExec:
		// use an absolute path to not use $PATH
		_, err := exec.LookPath(filepath.Join(r.Dir, x))
		return err == nil
	case syntax.TsNoEmpty:
		info := stat(r.Dir, x)
		return info != nil && info.Size() > 0
	//case syntax.TsFdTerm:
	case syntax.TsEmpStr:
		return x == ""
	case syntax.TsNempStr:
		return x != ""
	//case syntax.TsOptSet:
	case syntax.TsVarSet:
		_, e := r.lookupVar(x)
		return e
	//case syntax.TsRefVar:
	case syntax.TsNot:
		return x == ""
	default:
		r.runErr(0, "unhandled unary test op: %v", op)
		return false
	}
}
