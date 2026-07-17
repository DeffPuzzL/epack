package epack

import (
	"fmt"
	"os"
	"strings"
)

func (e *EPack) String() string {
	return e.unitString(0, e.units)
}

func (u *Unit) String() string {
	return fmt.Sprintf("U{idx:%v, kind:%v, name:%v, cld:%v}", u.seq, u.kind, u.name, len(u.child))
}

func (e *EPack) unitString(indent int, units []*Unit) string {
	if len(units) == 0 {
		return ""
	}

	s := ""
	f := strings.Repeat("\t", indent)
	for _, unit := range units {
		if unit == nil {
			continue
		}

		s += fmt.Sprintf("%s%s\n", f, unit)

		if len(unit.child) == 0 {
			continue
		}

		s += e.unitString(indent+1, unit.child)
	}

	return s
}

func print(format string, a ...any) {
	fmt.Fprintf(os.Stdout, format, a...)
}
