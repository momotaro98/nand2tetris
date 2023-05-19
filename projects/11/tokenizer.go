package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type JackTokenizer struct {
	// the parsing file
	file *os.File
	// the using stream
	scanner *bufio.Scanner
	// the current line and this line only contain command
	currentLine string
	// the queue contains all the token of the file
	queue    []string
	curToken string
}

var (
	symbolArr  = []string{"{", "}", "(", ")", "[", "]", ".", ",", ";", "+", "-", "*", "/", "&", "|", "<", ">", "=", "~"}
	keyWordArr = []string{"class", "method", "int", "function", "boolean", "constructor", "char", "void", "var", "static", "field", "let", "do", "if", "else", "while", "return", "true", "false", "null", "this"}
	symbolSet  = make(map[string]bool)
	keyWordSet = make(map[string]bool)
)

const (
	ReEXIdentifier = "^[a-zA-Z_]{1}[a-zA-Z0-9_]*"
)

func init() {
	for _, i := range symbolArr {
		symbolSet[i] = true
	}
	for _, i := range keyWordArr {
		keyWordSet[i] = true
	}
}

func NewJackTokenizer(file *os.File) *JackTokenizer {
	scanner := bufio.NewScanner(file)
	jt := &JackTokenizer{
		file:    file,
		scanner: scanner,
	}
	jt.characterTheFile()
	return jt
}

func (jt *JackTokenizer) GetFile() *os.File {
	return jt.file
}

func (jt *JackTokenizer) GetScanner() *bufio.Scanner {
	return jt.scanner
}

func (jt *JackTokenizer) GetCurrentLine() string {
	return jt.currentLine
}

func (jt *JackTokenizer) GetQueue() []string {
	return jt.queue
}

func (jt *JackTokenizer) GetCurToken() string {
	return jt.curToken
}

func (jt *JackTokenizer) PutBack() {
	jt.queue = append([]string{jt.curToken}, jt.queue...)
}

func (jt *JackTokenizer) getCommand() bool {
	stringLine := jt.currentLine

	// remove the space line
	if stringLine == "" {
		return false
	}

	// Remove the comment line
	firstChar := stringLine[0:1]
	head := ""
	if len(stringLine) > 2 {
		head = stringLine[0:2]
	}
	threeChar := ""
	if len(stringLine) >= 3 {
		threeChar = stringLine[0:3]
	}
	if head == "//" || firstChar == "*" || head == "*/" || threeChar == "/**" {
		return false
	}

	// Throw the in-line comment away
	if strings.Contains(stringLine, "//") {
		subStr := stringLine[0:strings.Index(stringLine, "//")]
		jt.currentLine = strings.TrimSpace(subStr)
		return true
	}

	jt.currentLine = strings.TrimSpace(stringLine)
	return true
}

func (jt *JackTokenizer) readCommand() bool {
	if jt.scanner.Scan() {
		jt.currentLine = strings.TrimSpace(jt.scanner.Text())
		if !jt.getCommand() {
			return jt.readCommand()
		}
		return true
	} else {
		return false
	}
}

func (jt *JackTokenizer) characterTheCurrentLine() {
	if jt.currentLine != "" {
		line := jt.currentLine
		// Character the current line
		cArr := []rune(line)

		// This buffer is used to store the Strings which is stitched by the cArr.
		// And by using the buffer we can split the strings by word.
		lineBuffer := strings.Builder{}
		isStringConstant := 0
		for _, c := range cArr {
			ch := string(c)
			if ch == "\"" {
				isStringConstant = ^isStringConstant
			}
			if isStringConstant == -1 {
				lineBuffer.WriteRune(c)
			} else if symbolSet[ch] {
				lineBuffer.WriteString(" " + ch + " ")
			} else {
				lineBuffer.WriteRune(c)
			}
		}
		strFields := strings.Fields(lineBuffer.String())

		// Set a label to process string constant.
		// When we first meet a """, we negate the label. When we second meet the """, we negate the label.
		label := 0
		strConstantBuffer := strings.Builder{}
		for _, i := range strFields {
			if i == " " || i == "" {
				continue
			}
			if string(i[0]) == "\"" || string(i[len(i)-1]) == "\"" {
				label = ^label
				if label == -1 {
					strConstantBuffer.WriteString(i + " ")
				} else {
					strConstantBuffer.WriteString(i)
					jt.queue = append(jt.queue, strConstantBuffer.String())
					strConstantBuffer.Reset()
					continue
				}
				continue
			}
			if label == -1 {
				strConstantBuffer.WriteString(i + " ")
				continue
			}
			jt.queue = append(jt.queue, i)
		}
	}
}

func (jt *JackTokenizer) characterTheFile() {
	for jt.readCommand() {
		jt.characterTheCurrentLine()
	}
}

func (jt *JackTokenizer) HasMoreToken() bool {
	return len(jt.queue) > 0
}

func (jt *JackTokenizer) Advance() {
	if jt.HasMoreToken() {
		jt.curToken = jt.queue[len(jt.queue)-1]
		jt.queue = jt.queue[:len(jt.queue)-1]
	}
}

func (jt *JackTokenizer) TokenType() TokenType {
	if keyWordSet[jt.curToken] {
		return KEYWORD
	}
	if symbolSet[jt.curToken] {
		return SYMBOL
	}
	if matched, _ := regexp.MatchString(ReEXIdentifier, jt.curToken); matched {
		return IDENTIFIER
	}
	if matched, _ := regexp.MatchString("^[0-9]+", jt.curToken); matched {
		return INT_CONST
	}
	if strings.HasPrefix(jt.curToken, "\"") && strings.HasSuffix(jt.curToken, "\"") {
		return STRING_CONST
	}
	return ""
}

func (jt *JackTokenizer) Keyword() string {
	return jt.curToken
}

func (jt *JackTokenizer) Symbol() string {
	return jt.curToken
}

func (jt *JackTokenizer) Identifier() string {
	return jt.curToken
}

func (jt *JackTokenizer) IntVal() int {
	val, _ := strconv.Atoi(jt.curToken)
	return val
}

func (jt *JackTokenizer) StringVal() string {
	return jt.curToken
}
