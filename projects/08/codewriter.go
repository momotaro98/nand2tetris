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
	ifLabelNum                int // 重複しないラベルを生成するためのカウンター
	returnLabelNum            int
	currentFunctionName       string
}

func NewCodeWriter(oFilePath string) CodeWriter {
	oFile, _ := os.Create(oFilePath)
	writer := bufio.NewWriter(oFile)
	return &codeWriter{
		oFile:      oFile,
		writer:     writer,
		ifLabelNum: 0,
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
	l1 := cw.getNewIfLabel()
	l2 := cw.getNewIfLabel()

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

func (cw *codeWriter) getNewIfLabel() string {
	cw.ifLabelNum++
	return fmt.Sprintf("IF_LABEL_%d", cw.ifLabelNum)
}

func (cw *codeWriter) getNewReturnLabel() string {
	cw.returnLabelNum++
	return fmt.Sprintf("RETURN_LABEL_%d", cw.returnLabelNum)
}

func (cw *codeWriter) WriteInit() {
	// スタックポインタ(SP)を0x0100(256)に初期化する
	cw.writeCodes([]string{
		"@256",
		"D=A",
		"@SP",
		"M=D",
	})

	// (変換されたコードの) Sys.init を実行する
	cw.WriteCall("Sys.init", 0)
}

func (cw *codeWriter) WriteLabel(label string) {
	ln := cw.getLabelName(label)
	cw.writeCode(fmt.Sprintf("(%s)", ln))
}

func (cw *codeWriter) WriteGoto(label string) {
	cw.writeCodes([]string{
		fmt.Sprintf("@%s", cw.getLabelName(label)),
		"0;JMP",
	})
}

func (cw *codeWriter) WriteIf(label string) {
	cw.writePopToMRegister() // ワーキングスタックTopのアドレスをAレジスタに格納する
	cw.writeCodes([]string{
		"D=M", // ワーキングスタックTopの値がDレジスタに入る
		fmt.Sprintf("@%s", cw.getLabelName(label)), // ラベルをAレジスタにロードする
		"D;JNE", // Dの値が0でない(True)ならばラベル先へジャンプする
	})
}

func (cw *codeWriter) WriteCall(functionName string, numArgs int) {
	// Return step1
	// 呼び先のreturn後は次の処理へジャンプしたい。
	// return後のジャンプ先はWriteCallの最後にラベル付けしている(Return step3)。
	// 呼び先側にジャンプ先のラベルのアドレスを教えるためにラベル値のアドレスをスタックに積む。
	// 実際にジャンプするのはReturn Step2である。
	returnLabel := cw.getNewReturnLabel()
	cw.writeCodes([]string{
		fmt.Sprintf("@%s", returnLabel),
		"D=A",
	})
	cw.writePushFromDRegister() // push return-address

	// 呼び先側のreturnから戻ってきたときのために、
	// 呼び出し側LCL,ARG,THIS,THATをスタックに積んでおく

	// LCLが持つ現状の先頭アドレス値をスタックに積む
	cw.writeCodes([]string{
		"@LCL",
		"D=M",
	})
	cw.writePushFromDRegister()

	// ARGが持つ現状の先頭アドレス値をスタックに積む
	cw.writeCodes([]string{
		"@ARG",
		"D=M",
	})
	cw.writePushFromDRegister()

	// THISが持つ現状の先頭アドレス値をスタックに積む
	cw.writeCodes([]string{
		"@THIS",
		"D=M",
	})
	cw.writePushFromDRegister()

	// THATが持つ現状の先頭アドレス値をスタックに積む
	cw.writeCodes([]string{
		"@THAT",
		"D=M",
	})
	cw.writePushFromDRegister()

	// ARGを呼び先側での処理を開始するために移動させる。
	// Callをする前に対象関数のARGをスタックに積んでいるので、
	// 現状のスタック位置(SP)から対象関数のARG個分(n)と
	// return-address,呼び出し側LCL,ARG,THIS,THAT(5個)を
	// 引いた位置になる。
	cw.writeCodes([]string{
		"@SP",
		"D=M",
		"@5",
		"D=D-A",
		fmt.Sprintf("@%d", numArgs),
		"D=D-A",
		"@ARG",
		"M=D", // ARG = SP - n - 5
	})

	// LCLを呼び先側での処理を開始するために移動させる。
	// スタックマシンの設計上、return-address,呼び出し側LCL,ARG,THIS,THATを
	// 積んでいる現状のスタック位置(SP)がLCLの移動先になる。
	cw.writeCodes([]string{
		"@SP",
		"D=M",
		"@LCL",
		"M=D", // LCL = SP
	})

	cw.writeCodes([]string{
		// Return step2
		// 呼び先の関数へジャンプする
		fmt.Sprintf("@%s", functionName),
		"0;JMP", // goto function
		// Return step3
		// 呼び先の関数のreturnの処理によって
		// スタックに積んでいたこの場所のラベルへジャンプしてくる。
		fmt.Sprintf("(%s)", returnLabel),
	})
}

func (cw *codeWriter) WriteReturn() {
	// FRAMEの設定とReturnアドレスを一時領域へ取得する。
	// FRAME(R13)は以降の呼び出し側の関数の状態へ
	// LCL,ARG,THIS,THATを戻し移すために便宜上必要な一時変数。
	// 実態は呼び先でのLCLである。
	cw.writeCodes([]string{
		"@LCL",
		"D=M",
		"@R13",
		"M=D", // R13 = FRAME = LCL
		"@5",
		"D=A",
		"@R13",
		"A=M-D",
		"D=M", // D = *(FRAME-5) = return-address
		"@R14",
		"M=D", // R14 = return-address // R14の一時領域にReturn先アドレスを入れておく
	})

	// ワーキングスタックのTopにある関数の戻り値を現状呼び先ARGの先頭アドレス先に格納する。
	// その理由は、現状呼び先ARGの先頭アドレスは、スタックマシンの設計上、
	// 呼び出し側へreturnから戻ったときのスタックTOP値(SP-1)であるから。
	cw.writePopToMRegister()
	cw.writeCodes([]string{
		"D=M",
		"@ARG",
		"A=M", // M = *ARG
		"M=D", // *ARG = pop()
	})

	cw.writeCodes([]string{
		// 呼び出し側のSPを戻す
		"@ARG",
		"D=M+1",
		"@SP",
		"M=D", // SP = ARG + 1 // スタックマシンの設計上、現ARG+1が呼び出し側のSPになる。

		// 呼び出し側のTHATを戻す
		"@R13",
		"AM=M-1", // A = FRAME-1, R13 = FRAME-1
		"D=M",
		"@THAT",
		"M=D", // THAT = *(FRAME-1)

		// 呼び出し側のTHISを戻す
		"@R13",
		"AM=M-1",
		"D=M",
		"@THIS",
		"M=D", // THIS = *(FRAME-2)

		// 呼び出し側のARGを戻す
		"@R13",
		"AM=M-1",
		"D=M",
		"@ARG",
		"M=D", // ARG = *(FRAME-3)

		// 呼び出し側のLCLを戻す
		"@R13",
		"AM=M-1",
		"D=M",
		"@LCL",
		"M=D", // LCL = *(FRAME-4)

		// 一時領域R14に入れておいたreturn-address先へジャンプする
		"@R14",
		"A=M",
		"0;JMP", // goto return-address
	})

}

func (cw *codeWriter) WriteFunction(functionName string, numLocals int) {
	cw.writeCodes([]string{
		fmt.Sprintf("(%s)", functionName),
		"D=0",
	})
	for i := 0; i < numLocals; i++ {
		// 定義Functionが持つローカル変数個数分のメモリ領域を確保する。
		// スタックマシンの設計上、グローバルスタック上がメモリ領域になる。
		cw.writePushFromDRegister() // Dレジスタ値(=0)を初期値としてメモリ領域を確保する。
	}
	cw.currentFunctionName = functionName
}

func (cw *codeWriter) getLabelName(label string) string {
	if cw.currentFunctionName != "" {
		return fmt.Sprintf("%s:%s", cw.currentFunctionName, label)
	}
	return fmt.Sprintf("global:%s", label)
}
