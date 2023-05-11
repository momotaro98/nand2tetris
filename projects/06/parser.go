package main

import (
	"bufio"
	"os"
	"strings"
)

type Command string

const A_COMMAND Command = "A_COMMAND"
const C_COMMAND Command = "C_COMMAND"
const L_COMMAND Command = "L_COMMAND"

type Parser interface {
	hasMoreCommands() bool
	advance()
	commandType() Command
	symbol() string
	dest() Mnemonic
	comp() Mnemonic
	jump() Mnemonic
}

func NewParser(file *os.File) Parser {
	scanner := bufio.NewScanner(file)
	return &parser{
		scanner:        scanner,
		currentCommand: "",
		nextCommand:    "",
	}
}

type parser struct {
	scanner        *bufio.Scanner
	currentCommand string
	nextCommand    string
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
		strings.ReplaceAll(line, " ", "")       // 中間空白除去
		if len(line) < 1 {                      // 空行スキップ
			continue
		}
		p.nextCommand = line
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
// 現コマンドの種類を返す。
func (p *parser) commandType() Command {
	cmd := p.currentCommand
	if strings.HasPrefix(cmd, "@") {
		return A_COMMAND
	}
	if strings.HasPrefix(cmd, "(") && strings.HasSuffix(cmd, ")") {
		return L_COMMAND
	}
	return C_COMMAND
}

// symbol implements Parser
// 現コマンド@Xxxまたは(Xxx)のXxxを返す。Xxxはシンボルまたは10進数の数値である。
// このルーチンはcommandType()がA_COMMANDまたはL_COMMANDのときだけ呼ぶように
// する。
func (p *parser) symbol() string {
	s, found := strings.CutPrefix(p.currentCommand, "@")
	if found {
		return s
	}
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	return s
}

// dest implements Parser
// 現C命令のdestニーモニックを返す。
func (p *parser) dest() Mnemonic {
	s := strings.Split(p.currentCommand, "=")
	if len(s) > 1 {
		return Mnemonic(s[0])
	}
	return ""
}

// comp implements Parser
// 現C命令のcompニーモニックを返す。
func (p *parser) comp() Mnemonic {
	s := p.currentCommand
	if strings.Contains(p.currentCommand, "=") {
		s = strings.Split(s, "=")[1]
	}
	if strings.Contains(p.currentCommand, ";") {
		s = strings.Split(s, ";")[0]
	}
	return Mnemonic(s)
}

// jump implements Parser
// 現C命令のjumpニーモニックを返す。
func (p *parser) jump() Mnemonic {
	s := strings.Split(p.currentCommand, ";")
	if len(s) > 1 {
		return Mnemonic(s[1])
	}
	return ""
}
