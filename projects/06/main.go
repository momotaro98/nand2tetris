package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	dirName  = flag.String("dir", "add", "doc here")
	fileName = flag.String("file", "Add", "doc here")
)

func main() {
	flag.Parse()

	code := NewCode()
	symbolT := NewSymbolTable()

	// loop1: 目的: 疑似コマンド (Xxx) のシンボルテーブルの作成。
	// 命令の度に0からインクリメントし(Xxx)の疑似コマンドを見つけたら
	// そのときの命令番号をSymbolTableに格納する。
	iFile1, err := os.Open(fmt.Sprintf("./%s/%s.asm", *dirName, *fileName))
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer iFile1.Close()
	parser1 := NewParser(iFile1)
	instCount := 0
	for parser1.hasMoreCommands() {
		parser1.advance()
		if parser1.commandType() == L_COMMAND {
			symbolT.addEntry(parser1.symbol(), instCount)
			continue
		}
		instCount++
	}

	// loop2: 目的: バイナリ作成。
	// シンボルテーブルを参照/追加しながら動かす。マシン仕様従って変換する。
	iFile2, err := os.Open(fmt.Sprintf("./%s/%s.asm", *dirName, *fileName))
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer iFile2.Close()
	ramAddrCounter := 16 // 変数対応用のシンボルテーブルへのRAMアドレス格納用
	parser2 := NewParser(iFile2)
	oFile, err := os.Create(fmt.Sprintf("./%s/%s-actual.hack", *dirName, *fileName))
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer oFile.Close()
	writer := bufio.NewWriter(oFile)
	for parser2.hasMoreCommands() {
		parser2.advance()
		var (
			result string
		)
		switch parser2.commandType() {
		case A_COMMAND:
			symbol := parser2.symbol()
			if dec, err := strconv.Atoi(symbol); err == nil {
				// @123 ← symbolが数値のパターン → 対象数値をバイナリにする。
				result = fmt.Sprintf("0%015b", dec)
			} else {
				if symbolT.contains(symbol) {
					// @R0 ← シンボルテーブルに既にあるパターン → 対象intをバイナリにする。
					result = fmt.Sprintf("0%015b", symbolT.getAddress(symbol))
				} else {
					// @i ← シンボルテーブルに存在していないパターン → 新規にRAM[16]へテーブル追加し、アドレスをバイナリにする。
					symbolT.addEntry(symbol, ramAddrCounter)
					result = fmt.Sprintf("0%015b", symbolT.getAddress(symbol))
					ramAddrCounter++
				}
			}
		case L_COMMAND:
			continue
		case C_COMMAND:
			compM := parser2.comp()
			destM := parser2.dest()
			jumpM := parser2.jump()
			result = fmt.Sprintf("111%s%s%s",
				code.comp(compM),
				code.dest(destM),
				code.jump(jumpM),
			)
		}

		_, _ = io.WriteString(writer, result+"\n")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			os.Exit(1)
		}
	}

	writer.Flush()
}
