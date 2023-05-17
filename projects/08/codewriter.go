package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	TEMP_BASE_ADDRESS    = 5
	POINTER_BASE_ADDRESS = 3
)

type CodeWriter interface {
	Close() error
	Flush() error
	SetFileName(filename string)
	WriteArithmetic(command string)
	WritePushPop(command CommandType, segment string, index int)
	WriteInit()
	WriteLabel(label string)
	WriteGoto(label string)
	WriteIf(label string)
	WriteCall(functionName string, numArgs int)
	WriteReturn()
	WriteFunction(functionName string, numLocals int)
}

type codeWriter struct {
	currentTranslatedFileName string
	oFile                     *os.File
	writer                    *bufio.Writer
	labelNum                  int // 重複しないラベルを生成するためのカウンター
	currentFunctionName       string
}

func NewCodeWriter(oFilePath string) CodeWriter {
	oFile, _ := os.Create(oFilePath)
	writer := bufio.NewWriter(oFile)
	return &codeWriter{
		oFile:    oFile,
		writer:   writer,
		labelNum: 0,
	}
}

func (cw *codeWriter) Close() error {
	return cw.oFile.Close()
}

func (cw *codeWriter) Flush() error {
	return cw.writer.Flush()
}

func (cw *codeWriter) SetFileName(filename string) {
	cw.currentTranslatedFileName = filename
}

func (cw *codeWriter) WriteArithmetic(command string) {
	switch command {
	case "add", "sub", "and", "or":
		cw.writeBinaryOperation(command)
	case "neg", "not":
		cw.writeUnaryOperation(command)
	case "eq", "gt", "lt":
		cw.writeCompOperation(command)
	}
}

func (cw *codeWriter) writeBinaryOperation(command string) {
	// stackのtopをDレジスタに入れる
	cw.writePopToMRegister()
	cw.writeCode("D=M")
	// stackの次のtopのアドレスをAレジスタに入れる
	cw.writePopToMRegister()
	// D = D (operand) M を計算する
	switch command {
	case "add":
		cw.writeCode("D=D+M")
	case "sub":
		cw.writeCode("D=M-D") // 高級言語→VM言語の変換(コンパイル)にて D=D-M ではなく D=M-D になるっぽい
	case "and":
		cw.writeCode("D=D&M")
	case "or":
		cw.writeCode("D=D|M")
	}
	// Dレジスタ値をスタックに入れスタックをインクリメントする
	cw.writePushFromDRegister()
}

func (cw *codeWriter) writeUnaryOperation(command string) {
	cw.writeCodes([]string{
		"@SP",
		"A=M-1",
	})
	// Mem[A]上でALUを通してそのまま演算できる
	switch command {
	case "neg":
		cw.writeCode("M=-M")
	case "not":
		cw.writeCode("M=!M")
	}
}

// stack上の2つを指定の比較演算で比較し
// Trueならば-1を Falseならば0 (本書籍の機械語の仕様である) を
// スタックのTopに積む
func (cw *codeWriter) writeCompOperation(command string) {
	// stackのtopをDレジスタに入れる
	cw.writePopToMRegister()
	cw.writeCode("D=M")
	// stackの次のtopのアドレスをAレジスタに入れる
	cw.writePopToMRegister()

	// JUMP用のラベルを生成する
	l1 := cw.getNewLabel()
	l2 := cw.getNewLabel()

	var compType string
	switch command {
	case "eq":
		compType = "JEQ"
	case "gt":
		compType = "JGT"
	case "lt":
		compType = "JLT"
	}

	cw.writeCodes([]string{
		"D=M-D",                       // 2つの値の差分を取る
		fmt.Sprintf("@%s", l1),        // 次コマンドでのJMP指定先のラベルをロード
		fmt.Sprintf("D;%s", compType), // D;JEQ or D;JGT or D;JLT のいづれかでTrueならば前コマンドのラベルへ移動する
		"D=0",                         // case False 前コマンドでFalseになったので D=0 (本書の機械語仕様)
		fmt.Sprintf("@%s", l2),        // case False Dレジスタ値をスタックに積むコマンドの手前まで飛ぶ用のロード
		"0;JMP",                       // case False 飛ぶ
		fmt.Sprintf("(%s)", l1),       // case True の場合のラベル先
		"D=-1",                        // case True なので D=-1 (本書の機械語仕様)
		fmt.Sprintf("(%s)", l2),       // case False での飛ぶ先
	})

	// Dレジスタ値をスタックに入れスタックをインクリメントする
	cw.writePushFromDRegister()
}

func (cw *codeWriter) WritePushPop(command CommandType, segment string, index int) {
	switch command {
	case C_PUSH:
		switch segment {
		case "constant":
			cw.writeCodes([]string{
				fmt.Sprintf("@%d", index),
				"D=A",
			})
			cw.writePushFromDRegister()
		case "local", "argument", "this", "that":
			cw.writePushFromVirtualSegment(segment, index)
		case "temp", "pointer":
			cw.writePushFromStaticSegment(segment, index)
		case "static":
			cw.writeCodes([]string{
				fmt.Sprintf("@%s.%d", cw.currentTranslatedFileName, index),
			})
			cw.writeCode("D=M")
			cw.writePushFromDRegister()
		}
	case C_POP:
		switch segment {
		case "local", "argument", "this", "that":
			cw.writePopFromVirtualSegment(segment, index)
		case "temp", "pointer":
			cw.writePopFromStaticSegment(segment, index)
		case "static":
			cw.writePopToMRegister()
			cw.writeCodes([]string{
				"D=M",
				fmt.Sprintf("@%s.%d", cw.currentTranslatedFileName, index),
			})
			cw.writeCode("M=D")
		}

	}
}

func (cw *codeWriter) writePushFromVirtualSegment(segment string, index int) {
	var registerName string

	switch segment {
	case "local":
		registerName = "LCL"
	case "argument":
		registerName = "ARG"
	case "this":
		registerName = "THIS"
	case "that":
		registerName = "THAT"
	}

	cw.writeCodes([]string{
		fmt.Sprintf("@%s", registerName),
		"A=M",
	})

	// 先頭のアドレスからindex分運ぶ
	for i := 0; i < index; i++ {
		cw.writeCode("A=A+1")
	}
	// 対象のアドレス上の値をDレジスタに格納する
	cw.writeCode("D=M")
	// Dレジスタ値をスタックに積む
	cw.writePushFromDRegister()
}

func (cw *codeWriter) writePushFromStaticSegment(segment string, index int) {
	var baseAddress int

	switch segment {
	case "temp":
		baseAddress = TEMP_BASE_ADDRESS
	case "pointer":
		baseAddress = POINTER_BASE_ADDRESS
	}

	cw.writeCodes([]string{
		fmt.Sprintf("@%d", baseAddress),
	})

	// 先頭のアドレスからindex分運ぶ
	for i := 0; i < index; i++ {
		cw.writeCode("A=A+1")
	}
	// 対象のアドレス上の値をDレジスタに格納する
	cw.writeCode("D=M")
	// Dレジスタ値をスタックに積む
	cw.writePushFromDRegister()
}

func (cw *codeWriter) writePopFromVirtualSegment(segment string, index int) {
	var registerName string

	switch segment {
	case "local":
		registerName = "LCL"
	case "argument":
		registerName = "ARG"
	case "this":
		registerName = "THIS"
	case "that":
		registerName = "THAT"
	}

	cw.writePopToMRegister() // スタック先頭のアドレスをAレジスタに格納する
	cw.writeCodes([]string{
		"D=M",                            // D=M=Mem[A] なので Dレジスタにスタック先頭の値を入れる
		fmt.Sprintf("@%s", registerName), // 指定セグメントアドレス先頭値を持つ特殊アドレスをAレジスタに格納する
		"A=M",                            // A=M=Mem[A] なので 対象セグメントの先頭アドレスをAレジスタに格納する
	})

	for i := 0; i < index; i++ {
		cw.writeCode("A=A+1") // Index指定先まで運ぶ
	}

	cw.writeCode("M=D") // M=Mem[A]=D なのでDレジスタ上にある値を指定アドレス先へ格納する
}

func (cw *codeWriter) writePopFromStaticSegment(segment string, index int) {
	var baseAddress int

	switch segment {
	case "temp":
		baseAddress = TEMP_BASE_ADDRESS
	case "pointer":
		baseAddress = POINTER_BASE_ADDRESS
	}

	cw.writePopToMRegister()
	cw.writeCodes([]string{
		"D=M",
		fmt.Sprintf("@%d", baseAddress), // tempとpointerはセグメントの先頭アドレスが固定である
	})

	for i := 0; i < index; i++ {
		cw.writeCode("A=A+1")
	}

	cw.writeCode("M=D")
}

func (cw *codeWriter) writePushFromDRegister() {
	cw.writeCodes([]string{
		// put value of D register onto stack
		"@SP",
		"A=M",
		"M=D",
		// and increment stack address
		"@SP",
		"M=M+1",
	})
}

func (cw *codeWriter) writePopToMRegister() {
	// load stack value to A register
	cw.writeCodes([]string{
		"@SP",
		"M=M-1",
		"A=M",
	})
}

func (cw *codeWriter) writeCodes(s []string) {
	_, _ = io.WriteString(cw.writer, strings.Join(s, "\n")+"\n")
}

func (cw *codeWriter) writeCode(s string) {
	_, _ = io.WriteString(cw.writer, s+"\n")
}

func (cw *codeWriter) getNewLabel() string {
	cw.labelNum++
	return fmt.Sprintf("LABEL%d", cw.labelNum)
}

func (cw *codeWriter) WriteInit() {}

func (cw *codeWriter) WriteLabel(label string) {
	ln := cw.getLabelName(label)
	cw.writeCode(fmt.Sprintf("(%s)", ln))
}

func (cw *codeWriter) WriteGoto(label string) {}

func (cw *codeWriter) WriteIf(label string) {
	cw.writePopToMRegister() // ワーキングスタックTopのアドレスをAレジスタに格納する
	cw.writeCodes([]string{
		"D=M", // ワーキングスタックTopの値がDレジスタに入る
		fmt.Sprintf("@%s", cw.getLabelName(label)), // ラベルをAレジスタにロードする
		"D;JNE", // Dの値が0でない(True)ならばラベル先へジャンプする
	})
}

func (cw *codeWriter) WriteCall(functionName string, numArgs int) {}

func (cw *codeWriter) WriteReturn() {}

func (cw *codeWriter) WriteFunction(functionName string, numLocals int) {}

func (cw *codeWriter) getLabelName(label string) string {
	if cw.currentFunctionName != "" {
		return fmt.Sprintf("%s:%s", cw.currentFunctionName, label)
	}
	return fmt.Sprintf("global:%s", label)
}
