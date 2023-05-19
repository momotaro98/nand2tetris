package main

const (
	IndexType = iota
	IndexKind
	IndexNum
)

type SymbolTable struct {
	classTable      map[string][]interface{}
	subroutineTable map[string][]interface{}
	staticIndex     int
	fieldIndex      int
	argIndex        int
	varIndex        int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		classTable:      make(map[string][]interface{}),
		staticIndex:     0,
		fieldIndex:      0,
		subroutineTable: make(map[string][]interface{}),
		argIndex:        0,
		varIndex:        0,
	}
}

func (st *SymbolTable) ReMakeSubTable() {
	st.subroutineTable = make(map[string][]interface{})
}

func (st *SymbolTable) isInClassTable(name string) bool {
	_, ok := st.classTable[name]
	return ok
}

func (st *SymbolTable) isInSubTable(name string) bool {
	_, ok := st.subroutineTable[name]
	return ok
}

func (st *SymbolTable) Define(name, typeVal, kind string) {
	switch kind {
	case "static":
		if _, ok := st.classTable[name]; !ok {
			st.classTable[name] = []interface{}{typeVal, kind, st.staticIndex}
			st.staticIndex++
		}
	case "field":
		if _, ok := st.classTable[name]; !ok {
			st.classTable[name] = []interface{}{typeVal, kind, st.fieldIndex}
			st.fieldIndex++
		}
	case "arg":
		if _, ok := st.subroutineTable[name]; !ok {
			st.subroutineTable[name] = []interface{}{typeVal, kind, st.argIndex}
			st.argIndex++
		}
	case "var":
		if _, ok := st.subroutineTable[name]; !ok {
			st.subroutineTable[name] = []interface{}{typeVal, kind, st.varIndex}
			st.varIndex++
		}
	}
}

func (st *SymbolTable) VarCount(kind string) int {
	switch kind {
	case "static":
		return st.staticIndex
	case "field":
		return st.fieldIndex
	case "arg":
		return st.argIndex
	case "var":
		return st.varIndex
	default:
		return 0
	}
}

func (st *SymbolTable) KindOf(name string) string {
	if val, ok := st.classTable[name]; ok {
		return val[IndexKind].(string)
	} else if val, ok := st.subroutineTable[name]; ok {
		return val[IndexKind].(string)
	} else {
		return ""
	}
}

func (st *SymbolTable) TypeOf(name string) string {
	if val, ok := st.classTable[name]; ok {
		return val[IndexType].(string)
	} else if val, ok := st.subroutineTable[name]; ok {
		return val[IndexType].(string)
	} else {
		return ""
	}
}

func (st *SymbolTable) IndexOf(name string) int {
	if val, ok := st.classTable[name]; ok {
		return val[IndexNum].(int)
	} else if val, ok := st.subroutineTable[name]; ok {
		return val[IndexNum].(int)
	} else {
		return 0
	}
}

func (st *SymbolTable) Clear() {
	st.subroutineTable = make(map[string][]interface{})
	st.argIndex = 0
	st.varIndex = 0
}
