package main

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CompilationEngine struct {
	inputFile     *os.File
	outputFile    *os.File
	jackTokenizer *JackTokenizer
	sTable        *SymbolTable
	vmWriter      *VMWriter
	expressionNum int
	className     string
	index         int
}

func NewCompilationEngine(inputFile *os.File) (*CompilationEngine, error) {
	path := filepath.Dir(inputFile.Name())
	prefixFileName := strings.Split(filepath.Base(inputFile.Name()), ".")[0]
	outputFile, err := os.Create(filepath.Join(path, prefixFileName+".vm"))
	if err != nil {
		return nil, err
	}

	return &CompilationEngine{
		inputFile:     inputFile,
		outputFile:    outputFile,
		jackTokenizer: NewJackTokenizer(inputFile),
		sTable:        NewSymbolTable(),
		vmWriter:      NewVMWriter(outputFile),
		index:         0,
	}, nil
}

func (ce *CompilationEngine) GetInputFile() *os.File {
	return ce.inputFile
}

func (ce *CompilationEngine) GetJackTokenizer() *JackTokenizer {
	return ce.jackTokenizer
}

func (ce *CompilationEngine) IsTerm(curToken string) bool {
	is09matched, err := regexp.MatchString("^[0-9]+", curToken)
	if err != nil {
		panic(err)
	}
	isEXmatched, err := regexp.MatchString(ReEXIdentifier, curToken)
	if err != nil {
		panic(err)
	}

	if is09matched || (strings.HasPrefix(curToken, "\"") && strings.HasSuffix(curToken, "\"")) ||
		curToken == "true" || curToken == "false" ||
		curToken == "null" || curToken == "this" ||
		isEXmatched ||
		curToken == "(" || curToken == "-" ||
		curToken == "~" {
		return true
	} else {
		return false
	}
}

func (ce *CompilationEngine) IsStatement(curToken string) bool {
	if curToken == "let" ||
		curToken == "if" ||
		curToken == "else" ||
		curToken == "while" ||
		curToken == "do" ||
		curToken == "return" {
		return true
	} else {
		return false
	}
}

func (ce *CompilationEngine) TransKind(kind string) string {
	switch kind {
	case "arg":
		return "argument"
	case "var":
		return "local"
	case "static":
		return "static"
	case "field":
		return "this"
	default:
		return "none"
	}
}

func (ce *CompilationEngine) CompileClass() {
	ce.jackTokenizer.Advance()
	curToken := ce.jackTokenizer.Keyword()
	if curToken != "class" {
		panic("Syntax error on token \"class\"")
	}

	ce.jackTokenizer.Advance()
	if !regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected token!")
	}
	ce.className = ce.jackTokenizer.GetCurToken()

	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.Symbol() != "{" {
		panic("Syntax error on class declaration, { expected at the end")
	}

	ce.jackTokenizer.Advance()
	for ce.jackTokenizer.GetCurToken() != "}" {
		if ce.jackTokenizer.GetCurToken() == "static" ||
			ce.jackTokenizer.GetCurToken() == "field" {
			ce.CompileClassVarDec()
		} else if ce.jackTokenizer.GetCurToken() == "constructor" ||
			ce.jackTokenizer.GetCurToken() == "function" ||
			ce.jackTokenizer.GetCurToken() == "method" {
			ce.CompileSubroutine()
		} else {
			panic("Unknown class declaration!")
		}
		ce.jackTokenizer.Advance()
	}

	if ce.jackTokenizer.Symbol() != "}" {
		panic("Syntax error on the file end, } expected at the end")
	}
}

func (ce *CompilationEngine) CompileClassVarDec() {
	// Variables for constructing the symbol table
	var name, kind, typ string

	// Determine the kind based on the current token
	if ce.jackTokenizer.Keyword() == "static" {
		kind = "static"
	} else {
		kind = "field"
	}

	// Read the type
	ce.jackTokenizer.Advance()
	if !regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.Keyword()) {
		panic("Syntax error: " + ce.jackTokenizer.Keyword() + " wrong data type")
	}
	typ = ce.jackTokenizer.GetCurToken()

	// Read the variable names
	ce.jackTokenizer.Advance()
	name = ce.jackTokenizer.GetCurToken()
	ce.sTable.Define(name, typ, kind)

	// Read the remaining tokens
	ce.jackTokenizer.Advance()
	for ce.jackTokenizer.GetCurToken() == "," {
		ce.jackTokenizer.Advance()
		name = ce.jackTokenizer.GetCurToken()
		ce.sTable.Define(name, typ, kind)
		ce.jackTokenizer.Advance()
	}

	if ce.jackTokenizer.Symbol() != ";" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected token!")
	}
}

func (ce *CompilationEngine) CompileSubroutine() {
	isConstructor := ce.jackTokenizer.GetCurToken() == "constructor"
	isMethod := ce.jackTokenizer.GetCurToken() == "method"

	// Write the class name.
	io.WriteString(ce.vmWriter.GetWriter(), "function "+ce.className+".")

	// Compile the return type.
	ce.jackTokenizer.Advance()
	if !regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", void or other data type expected.")
	}

	// Compile the subroutine name.
	ce.jackTokenizer.Advance()
	if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		io.WriteString(ce.vmWriter.GetWriter(), ce.jackTokenizer.GetCurToken())
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", identifier expected.")
	}

	// Compile '(' after subroutine name.
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "(" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected.")
	}

	// Compile the parameter list or null.
	ce.jackTokenizer.Advance()
	label := 0
	if ce.jackTokenizer.GetCurToken() == ")" {
		label = 1
	} else if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		if isMethod {
			ce.sTable.Define("this", ce.className, "arg")
		}
		ce.CompileParameterList()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected.")
	}
	if label == 0 {
		ce.jackTokenizer.Advance()
	}

	// Compile the subroutine body.
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "{" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected.")
	} else {
		// Read the remaining body.
		ce.jackTokenizer.Advance()

		// TODO: この辺に問題あり、Output.printInt を処理しないで CompileSubroutine を抜けてしまっている。

		// Variable declarations.
		if ce.jackTokenizer.GetCurToken() == "var" {
			for ce.jackTokenizer.GetCurToken() == "var" {
				ce.CompileVarDec()
				ce.jackTokenizer.Advance()
			}

			// Write the local variable number.
			io.WriteString(ce.vmWriter.GetWriter(), " "+strconv.Itoa(ce.sTable.VarCount("var"))+"\n")
		} else {
			// No local variables.
			io.WriteString(ce.vmWriter.GetWriter(), " 0\n")
		}

		// Allocate memory code, only for constructors.
		if isConstructor {
			ce.vmWriter.WritePush("constant", ce.sTable.VarCount("field"))
			ce.vmWriter.WriteCall("Memory.alloc", 1)
			ce.vmWriter.WritePop("pointer", 0)
		}

		// Get the current object code, only for methods.
		if isMethod {
			ce.vmWriter.WritePush("argument", 0)
			ce.vmWriter.WritePop("pointer", 0)
		}

		// Statements.
		for ce.jackTokenizer.GetQueue()[0] != "}" {
			ce.CompileStatements()
		}

		// When reaching "}".
		ce.jackTokenizer.Advance()
	}

	ce.sTable.Clear()
}

func (ce *CompilationEngine) CompileParameterList() {
	// Variables used to construct the symbol table.
	var name, typ, kind string
	kind = "arg"

	// Append the token firstly.
	typ = ce.jackTokenizer.GetCurToken()

	// The next expected token is an identifier.
	ce.jackTokenizer.Advance()
	if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		name = ce.jackTokenizer.GetCurToken()
		ce.sTable.Define(name, typ, kind)
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Read other tokens until ")"
	ce.jackTokenizer.Advance()
	for ce.jackTokenizer.GetCurToken() != ")" {
		// The next expected token is ",".
		if ce.jackTokenizer.GetCurToken() != "," {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", \",\" expected")
		}
		ce.jackTokenizer.Advance()

		// The next expected token is a type.
		if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
			typ = ce.jackTokenizer.GetCurToken()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}
		ce.jackTokenizer.Advance()

		// The next expected token is an identifier.
		if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
			name = ce.jackTokenizer.GetCurToken()
			ce.sTable.Define(name, typ, kind)
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}
		ce.jackTokenizer.Advance()
	}

	// Add the ")" in the front of the queue.
	ce.jackTokenizer.PutBack()
}

func (ce *CompilationEngine) CompileVarDec() {
	// Variables used to construct the symbol table.
	var name, typ, kind string
	kind = "var"

	// The next expected token is an identifier.
	ce.jackTokenizer.Advance()
	if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		typ = ce.jackTokenizer.GetCurToken()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.jackTokenizer.Advance()

	// The next expected token is an identifier.
	if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
		name = ce.jackTokenizer.GetCurToken()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.sTable.Define(name, typ, kind)

	// Read other tokens until ";"
	ce.jackTokenizer.Advance()
	for ce.jackTokenizer.GetCurToken() != ";" {
		// The next expected token is ",".
		if ce.jackTokenizer.GetCurToken() != "," {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected. \",\" expected.")
		}
		ce.jackTokenizer.Advance()

		// The next expected token is an identifier.
		if regexp.MustCompile(ReEXIdentifier).MatchString(ce.jackTokenizer.GetCurToken()) {
			name = ce.jackTokenizer.GetCurToken()
			ce.sTable.Define(name, typ, kind)
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}
		ce.jackTokenizer.Advance()
	}

	// Last token is ";", do nothing.
}

func (ce *CompilationEngine) CompileStatements() {
	for ce.jackTokenizer.GetCurToken() != "}" {
		switch ce.jackTokenizer.GetCurToken() {
		case "var":
			ce.CompileVarDec()
		case "let":
			ce.CompileLet()
		case "do":
			ce.CompileDo()
		case "if":
			ce.CompileIf()
		case "while":
			ce.CompileWhile()
		case "return":
			ce.CompileReturn()
		default:
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected.")
		}
		ce.jackTokenizer.Advance()
	}
	ce.jackTokenizer.PutBack()
}

func (ce *CompilationEngine) CompileDo() {
	callName := ""
	objectName := ""

	// The next expected token is identifier.
	ce.jackTokenizer.Advance()
	if matched, _ := regexp.MatchString(ReEXIdentifier, ce.jackTokenizer.GetCurToken()); matched {
		callName = ce.jackTokenizer.GetCurToken()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.jackTokenizer.Advance()

	// Judge if the call name is a class name.
	isClassCall := true
	firstChar := callName[0:1]
	if !regexp.MustCompile("[A-Z]").MatchString(firstChar) && ce.jackTokenizer.GetCurToken() == "." {
		objectName = callName
		callName = ce.sTable.TypeOf(callName)
		isClassCall = false
		ce.expressionNum++
	} else if !regexp.MustCompile("[A-Z]").MatchString(firstChar) && ce.jackTokenizer.GetCurToken() != "." {
		// Push the current object into stack.
		ce.vmWriter.WritePush("pointer", 0)
		// Let parameter number plus.
		ce.expressionNum++
		// Let the call name add the class name.
		callName = ce.className + "." + callName
	} else {
	}

	// Maybe the next expected token is "." or "(".
	if ce.jackTokenizer.GetCurToken() == "." {
		// Append the dot.
		callName += ce.jackTokenizer.GetCurToken()

		// The next expected token is identifier.
		ce.jackTokenizer.Advance()
		if matched, _ := regexp.MatchString(ReEXIdentifier, ce.jackTokenizer.GetCurToken()); matched {
			// Append the subroutine name.
			callName += ce.jackTokenizer.GetCurToken()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}
	} else if ce.jackTokenizer.GetCurToken() == "(" {
		ce.jackTokenizer.PutBack()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.jackTokenizer.Advance()

	// The next expected token is "(".
	if ce.jackTokenizer.GetCurToken() != "(" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	if !isClassCall {
		segmentName := ce.TransKind(ce.sTable.KindOf(objectName))
		ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(objectName))
	}

	// Expression list maybe null
	if ce.IsTerm(ce.jackTokenizer.GetQueue()[0]) {
		ce.CompileExpressionList()
	}

	// The next expected token is ")".
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ")" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected, ) expected.")
	}

	// The next expected token is ";".
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ";" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected, ) expected.")
	}

	ce.vmWriter.WriteCall(callName, ce.expressionNum)
	ce.expressionNum = 0 // clear this variable.
	ce.vmWriter.WritePop("temp", 0)
}

func (ce *CompilationEngine) CompileLet() {
	varName := ""
	isArr := false

	// The next expected token is identifier.
	ce.jackTokenizer.Advance()
	if matched, _ := regexp.MatchString(ReEXIdentifier, ce.jackTokenizer.GetCurToken()); matched {
		varName = ce.jackTokenizer.GetCurToken()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.jackTokenizer.Advance()

	// The next expected token is "[" or "=".
	startBrackets := 0
	if ce.jackTokenizer.GetCurToken() == "[" {
		isArr = true
		startBrackets = ^startBrackets
	} else if ce.jackTokenizer.GetCurToken() == "=" {

	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.jackTokenizer.Advance()

	// The next expected token is term.
	curToken := ce.jackTokenizer.GetCurToken()
	if ce.IsTerm(curToken) {
		ce.CompileExpression()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	if isArr {
		// Write array name.
		segmentName := ce.TransKind(ce.sTable.KindOf(varName))
		ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(varName))
		// Compute address.
		ce.vmWriter.WriteArithmetic("+")
	}

	// The right brackets
	if startBrackets == -1 {
		// Next expected token is right brackets
		ce.jackTokenizer.Advance()
		startBrackets = ^startBrackets

		// Next token "="?
		ce.jackTokenizer.Advance()
		if ce.jackTokenizer.GetCurToken() != "=" {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Next expected tokens is expression
		ce.jackTokenizer.Advance()
		if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
			ce.CompileExpression()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Next expected token is ";"
		if ce.jackTokenizer.GetQueue()[0] != ";" {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}
		ce.vmWriter.WritePop("temp", 0)
	}

	// The end token is ";"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ";" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	segmentName := ce.TransKind(ce.sTable.KindOf(varName))
	if isArr {
		ce.vmWriter.WritePop("pointer", 1)
		ce.vmWriter.WritePush("temp", 0)
		ce.vmWriter.WritePop("that", 0)
	} else {
		ce.vmWriter.WritePop(segmentName, ce.sTable.IndexOf(varName))
	}
}

func (ce *CompilationEngine) CompileWhile() {
	originIndex := ce.index
	ce.vmWriter.WriteLabel("WHILE_EXP" + strconv.Itoa(ce.index))

	// Next token "("
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "(" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next tokens expression
	ce.jackTokenizer.Advance()
	if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
		ce.CompileExpression()
	} else if ce.jackTokenizer.GetCurToken() == ")" {
		ce.jackTokenizer.PutBack()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token ")"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ")" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// This code segment is to judge the condition of the loop.
	ce.vmWriter.WriteArithmetic("~")
	ce.vmWriter.WriteIf("WHILE_END" + strconv.Itoa(originIndex))

	// Next token "{"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "{" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token statements
	ce.jackTokenizer.Advance()
	if ce.IsStatement(ce.jackTokenizer.GetCurToken()) {
		ce.CompileStatements()
	} else if ce.jackTokenizer.GetCurToken() == "}" {
		ce.jackTokenizer.PutBack()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token "}"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "}" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}
	ce.vmWriter.WriteGoto("WHILE_EXP" + strconv.Itoa(originIndex))
	ce.vmWriter.WriteLabel("WHILE_END" + strconv.Itoa(originIndex))
}

func (ce *CompilationEngine) CompileReturn() {
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() == ";" {
		ce.jackTokenizer.PutBack()
		ce.vmWriter.WritePush("constant", 0)
	} else if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
		ce.CompileExpression()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token ";"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ";" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	ce.vmWriter.WriteReturn()
}

func (ce *CompilationEngine) CompileIf() {
	// Next token "("
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "(" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token expression
	ce.jackTokenizer.Advance()
	if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
		ce.CompileExpression()
	} else if ce.jackTokenizer.GetCurToken() == ")" {
		ce.jackTokenizer.PutBack()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token ")"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != ")" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Write if code
	originIfIndex := ce.index + 1
	ce.vmWriter.WriteIf("IF_TRUE" + strconv.Itoa(originIfIndex))
	ce.vmWriter.WriteGoto("IF_FALSE" + strconv.Itoa(originIfIndex))
	ce.vmWriter.WriteLabel("IF_TRUE" + strconv.Itoa(originIfIndex))

	// Next token "{"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "{" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next tokens statements
	ce.jackTokenizer.Advance()
	if ce.IsStatement(ce.jackTokenizer.GetCurToken()) {
		ce.CompileStatements()
	} else if ce.jackTokenizer.GetCurToken() == "}" {
		ce.jackTokenizer.PutBack()
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Next token "}"
	ce.jackTokenizer.Advance()
	if ce.jackTokenizer.GetCurToken() != "}" {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
	}

	// Write label code
	if ce.jackTokenizer.GetQueue()[0] == "else" {
		// Only if the next token is "else", we need to set this label
		ce.vmWriter.WriteGoto("IF_END" + strconv.Itoa(originIfIndex))
	}
	ce.vmWriter.WriteLabel("IF_FALSE" + strconv.Itoa(originIfIndex))

	// Next token may be "else" or others
	if ce.jackTokenizer.GetQueue()[0] == "else" {
		// Append the token
		ce.jackTokenizer.Advance()

		// Next token "{"
		ce.jackTokenizer.Advance()
		if ce.jackTokenizer.GetCurToken() != "{" {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Next tokens "statements"
		ce.jackTokenizer.Advance()
		if ce.IsStatement(ce.jackTokenizer.GetCurToken()) {
			ce.CompileStatements()
		} else if ce.jackTokenizer.GetCurToken() == "}" {
			ce.jackTokenizer.PutBack()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Next token "}"
		ce.jackTokenizer.Advance()
		if ce.jackTokenizer.GetCurToken() != "}" {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Write the if statement exit gate
		ce.vmWriter.WriteLabel("IF_END" + strconv.Itoa(originIfIndex))
	}
}

func (ce *CompilationEngine) CompileExpression() {
	// Compile current token
	ce.CompileTerm()

	// The next expected token is an operator.
	// If the next token is "]" or ";" or ")" or ",", end this compilation.
	ce.jackTokenizer.Advance()
	endFlag := ce.jackTokenizer.GetCurToken() == "]" ||
		ce.jackTokenizer.GetCurToken() == ")" ||
		ce.jackTokenizer.GetCurToken() == ";" ||
		ce.jackTokenizer.GetCurToken() == ","
	for !endFlag {
		// Operator
		op := ce.jackTokenizer.GetCurToken()
		if matched, _ := regexp.MatchString(`\+|-|\*|/|\&|\||<|=|>`, op); matched {
			// Save the operator
			op = ce.jackTokenizer.GetCurToken()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + ", unexpected")
		}

		// Term
		ce.jackTokenizer.Advance()
		curToken := ce.jackTokenizer.GetCurToken()
		if ce.IsTerm(curToken) {
			ce.CompileTerm()
		}

		if op == "*" {
			ce.vmWriter.WriteCall("Math.multiply", 2)
		} else if op == "/" {
			ce.vmWriter.WriteCall("Math.divide", 2)
		} else {
			ce.vmWriter.WriteArithmetic(op)
		}

		// Check if it has reached the end
		ce.jackTokenizer.Advance()
		endFlag = ce.jackTokenizer.GetCurToken() == "]" ||
			ce.jackTokenizer.GetCurToken() == ")" ||
			ce.jackTokenizer.GetCurToken() == ";" ||
			ce.jackTokenizer.GetCurToken() == ","
	}
	ce.jackTokenizer.PutBack()
}

func (ce *CompilationEngine) CompileTerm() {
	// Integer constant
	if ce.jackTokenizer.TokenType() == INT_CONST {
		num, _ := strconv.Atoi(ce.jackTokenizer.GetCurToken())
		if num >= 0 && num < 32767 {
			ce.vmWriter.WritePush("constant", num)
		} else {
			panic("Integer over than max Integer: " + ce.jackTokenizer.GetCurToken())
		}
		// String constant
	} else if ce.jackTokenizer.TokenType() == STRING_CONST {
		stripString := ce.jackTokenizer.GetCurToken()[1 : len(ce.jackTokenizer.GetCurToken())-1]
		charString := []rune(stripString)
		ce.vmWriter.WritePush("constant", len(charString))
		ce.vmWriter.WriteCall("String.new", 1)
		for _, char := range charString {
			asci := int(char)
			ce.vmWriter.WritePush("constant", asci)
			ce.vmWriter.WriteCall("String.appendChar", 2)
		}
		// Keyword constant
	} else if ce.jackTokenizer.GetCurToken() == "true" ||
		ce.jackTokenizer.GetCurToken() == "false" ||
		ce.jackTokenizer.GetCurToken() == "null" ||
		ce.jackTokenizer.GetCurToken() == "this" {
		curToken := ce.jackTokenizer.GetCurToken()
		switch curToken {
		case "true":
			ce.vmWriter.WritePush("constant", 0)
			ce.vmWriter.WriteArithmetic("~")
		case "false":
			ce.vmWriter.WritePush("constant", 0)
		case "this":
			ce.vmWriter.WritePush("pointer", 0)
		case "null":
			ce.vmWriter.WritePush("constant", 0)
		default:
			break
		}
		// varName...
	} else if ce.jackTokenizer.TokenType() == IDENTIFIER {
		subName := ce.jackTokenizer.GetCurToken()
		head := ce.jackTokenizer.GetQueue()[0]
		if head == "[" {
			ce.jackTokenizer.Advance()
			ce.jackTokenizer.Advance()
			if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
				ce.CompileExpression()
			} else {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
			segmentName := ce.TransKind(ce.sTable.KindOf(subName))
			ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(subName))
			ce.jackTokenizer.Advance()
			if ce.jackTokenizer.GetCurToken() != "]" {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
			ce.vmWriter.WriteArithmetic("+")
			ce.vmWriter.WritePop("pointer", 1)
			ce.vmWriter.WritePush("that", 0)
		} else if head == "(" {
			ce.jackTokenizer.Advance()
			if ce.jackTokenizer.GetQueue()[0] == ce.jackTokenizer.GetQueue()[1] {
				ce.CompileExpressionList()
			} else if ce.jackTokenizer.GetQueue()[0] == ")" {
				// Do nothing
			} else {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
			ce.jackTokenizer.Advance()
			if ce.jackTokenizer.GetCurToken() != ")" {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
			ce.vmWriter.WriteCall(subName, ce.expressionNum)
			ce.expressionNum = 0
		} else if head == "." {
			isClassCall := true
			objectName := ""
			if matched, _ := regexp.MatchString(`[a-z]+`, subName); matched {
				isClassCall = false
				objectName = subName
				subName = ce.sTable.TypeOf(subName)
			}
			if !isClassCall {
				segmentName := ce.TransKind(ce.sTable.KindOf(objectName))
				ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(objectName))
				ce.expressionNum++
			}
			ce.jackTokenizer.Advance()
			subName += ce.jackTokenizer.GetCurToken()
			ce.jackTokenizer.Advance()
			if matched, _ := regexp.MatchString(ReEXIdentifier, ce.jackTokenizer.GetCurToken()); matched {
				subName += ce.jackTokenizer.GetCurToken()
				ce.jackTokenizer.Advance()
				if ce.jackTokenizer.GetCurToken() == "(" {
					if ce.IsTerm(ce.jackTokenizer.GetQueue()[0]) {
						ce.CompileExpressionList()
					} else if ce.jackTokenizer.GetQueue()[0] == ")" {
						// Do nothing
					} else {
						panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
					}
					ce.jackTokenizer.Advance()
					if ce.jackTokenizer.GetCurToken() != ")" {
						panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
					}
					ce.vmWriter.WriteCall(subName, ce.expressionNum)
					ce.expressionNum = 0
				} else {
					panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
				}
			} else {
				panic("subroutine call error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
		} else if matched, _ := regexp.MatchString(`\+|-|\*|/|\&|\||<|=|>`, head); matched {
			if matched, _ := regexp.MatchString(`[0-9]+`, ce.jackTokenizer.GetCurToken()); matched {
				i, err := strconv.Atoi(ce.jackTokenizer.GetCurToken())
				if err != nil {
					panic(err)
				}
				ce.vmWriter.WritePush("constant", i)
			} else {
				segmentName := ce.TransKind(ce.sTable.KindOf(ce.jackTokenizer.GetCurToken()))
				ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(ce.jackTokenizer.GetCurToken()))
			}
		} else if head == ")" || head == "]" || head == ";" || head == "," {
			segmentName := ce.TransKind(ce.sTable.KindOf(ce.jackTokenizer.GetCurToken()))
			ce.vmWriter.WritePush(segmentName, ce.sTable.IndexOf(ce.jackTokenizer.GetCurToken()))
		} else {
			panic("subroutine call error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
		}
		// ( expression )
	} else if ce.jackTokenizer.GetCurToken() == "(" {
		ce.jackTokenizer.Advance()
		curToken := ce.jackTokenizer.GetCurToken()
		if ce.IsTerm(curToken) {
			ce.CompileExpression()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
		}
		ce.jackTokenizer.Advance()
		if ce.jackTokenizer.GetCurToken() != ")" {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
		}
		// UnaryOp term
	} else if matched, _ := regexp.MatchString(`-|~`, ce.jackTokenizer.GetCurToken()); matched {
		op := "~"
		if ce.jackTokenizer.GetCurToken() == "-" {
			op = "--"
		}
		matchedReEX, _ := regexp.MatchString(ReEXIdentifier, ce.jackTokenizer.GetQueue()[0])
		matchedNum, _ := regexp.MatchString(`[0-9]+`, ce.jackTokenizer.GetQueue()[0])
		if matchedReEX || matchedNum {
			ce.jackTokenizer.Advance()
			ce.CompileTerm()
		} else if ce.jackTokenizer.GetQueue()[0] == "(" {
			ce.jackTokenizer.Advance()
			ce.CompileExpression()
		} else {
			panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
		}
		ce.vmWriter.WriteArithmetic(op)
	} else {
		panic("Term token compile error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
	}
}

func (ce *CompilationEngine) CompileExpressionList() {
	// Next token
	ce.jackTokenizer.Advance()
	if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
		ce.CompileExpression()
	}
	// Count the number of local variable number.
	ce.expressionNum++

	// next token maybe "," or ")"
	if len(ce.jackTokenizer.GetQueue()) > 0 {
		curToken := ce.jackTokenizer.GetQueue()[0]
		for curToken != ")" {
			if curToken == "," {
				ce.jackTokenizer.Advance()
			} else {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}

			ce.jackTokenizer.Advance()
			if ce.IsTerm(ce.jackTokenizer.GetCurToken()) {
				// Count the number of local variable number.
				ce.expressionNum++
				ce.CompileExpression()
			} else {
				panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
			}
			if len(ce.jackTokenizer.GetQueue()) > 0 {
				curToken = ce.jackTokenizer.GetQueue()[0]
			} else {
				break
			}
		}
	} else {
		panic("Syntax error: " + ce.jackTokenizer.GetCurToken() + " unexpected.")
	}
}
