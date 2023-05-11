package main

type SymbolTable interface {
	addEntry(symbol string, address int)
	contains(symbol string) bool
	getAddress(symbol string) int
}

type symbolTable map[string]int

var st = make(symbolTable)

func init() {
	st["SP"] = 0
	st["LCL"] = 1
	st["ARG"] = 2
	st["THIS"] = 3
	st["THAT"] = 4
	st["R0"] = 0
	st["R1"] = 1
	st["R2"] = 2
	st["R3"] = 3
	st["R4"] = 4
	st["R5"] = 5
	st["R6"] = 6
	st["R7"] = 7
	st["R8"] = 8
	st["R9"] = 9
	st["R10"] = 10
	st["R11"] = 11
	st["R12"] = 12
	st["R13"] = 13
	st["R14"] = 14
	st["R15"] = 15
	st["SCREEN"] = 16384
	st["KBD"] = 24576
}

func NewSymbolTable() SymbolTable {
	return st
}

// addEntry implements SymbolTable
func (st symbolTable) addEntry(symbol string, address int) {
	st[symbol] = address
}

// contains implements SymbolTable
func (st symbolTable) contains(symbol string) bool {
	_, ok := st[symbol]
	return ok
}

// getAddress implements SymbolTable
func (st symbolTable) getAddress(symbol string) int {
	if v, ok := st[symbol]; ok {
		return v
	}
	return -1
}
