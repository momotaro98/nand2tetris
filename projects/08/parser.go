package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type CommandType string

const (
	C_ARITHMETIC CommandType = "C_ARITHMETIC"
	C_PUSH       CommandType = "C_PUSH"
	C_POP        CommandType = "C_POP"
	C_LABEL      CommandType = "C_LABEL"
	C_GOTO       CommandType = "C_GOTO"
	C_IF         CommandType = "C_IF"
	C_FUNCTION   CommandType = "C_FUNCTION"
	C_RETURN     CommandType = "C_RETURN"
	C_CALL       CommandType = "C_CALL"
)

type Parser interface {
	hasMoreCommands() bool
	advance()
	commandType() CommandType
	arg1() string
	arg2() int
	Close() error
}

func NewParser(filePath string) Parser {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)
	return &parser{
		iFile:          file,
		scanner:        scanner,
		currentCommand: nil,
		nextCommand:    nil,
	}
}

type parser struct {
	iFile          *os.File
	scanner        *bufio.Scanner
	currentCommand []string
	nextCommand    []string
}

// hasMoreCommands implements Parser
// 入力にまだコマンドが存在するかを確認する。
func (p *parser) hasMoreCommands() bool {
	for {
		if !p.scanner.Scan() {
			return false
		}
		line := p.scanner.Text()
		line = strings.SplitN(line, "//", 2)[0] // コメント文除去
		line = strings.TrimSpace(line)          // 端空白削除
		// strings.ReplaceAll(line, " ", "")       // [Do not] 中間空白除去
		if len(line) < 1 { // 空行スキップ
			continue
		}
		p.nextCommand = strings.Fields(line)
		return true
	}
}

// advance implements Parser
// 入力から次のコマンドを読み、それを現在のコマンドにする。
// このルーチンはhasMoreCommands()がtrueの場合のみ呼ぶようにする。
// 最初は現コマンドは空である。
func (p *parser) advance() {
	p.currentCommand = p.nextCommand
}

// commandType implements Parser
// 現VMコマンドの種類を返す。
// 算術コマンドはすべて C_ARITHMETIC が返される。
func (p *parser) commandType() CommandType {
	switch p.currentCommand[0] {
	case "push":
		return C_PUSH
	case "pop":
		return C_POP
	case "label":
		return C_LABEL
	case "goto":
		return C_GOTO
	case "if-goto":
		return C_IF
	case "function":
		return C_FUNCTION
	case "return":
		return C_RETURN
	case "call":
		return C_CALL
	default:
		return C_ARITHMETIC
	}
}

// 現コマンドの最初の引数が返される。
// C_ARITHMETIC の場合、コマンド自体(add, subなど)が返される。
// 現コマンドが C_RETURN の場合、本メソッドは呼ばないようにする。
func (p *parser) arg1() string {
	if p.commandType() == C_ARITHMETIC {
		return p.currentCommand[0]
	} else {
		return p.currentCommand[1]
	}
}

// 現コマンドの2番目の引数が返される。
// 現コマンドが C_PUSH, C_POP, C_FUNCTION, C_CALLL の場合のみ、本メソッドを呼ぶようにする。
func (p *parser) arg2() int {
	i, err := strconv.Atoi(p.currentCommand[2])
	if err != nil {
		panic(err)
	}
	return i
}

func (p *parser) Close() error {
	return p.iFile.Close()
}
