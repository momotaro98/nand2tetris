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

	file, err := os.Open(fmt.Sprintf("./%s/%s.asm", *dirName, *fileName))
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	parser := NewParser(file)
	code := NewCode()

	oFile, err := os.Create(fmt.Sprintf("./%s/%s-actual.hack", *dirName, *fileName))
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()
	writer := bufio.NewWriter(oFile)

	for parser.hasMoreCommands() {
		parser.advance()

		// if parser.hasMoreCommands() {
		// 	parser.advance()
		// }

		var (
			result string
		)
		switch parser.commandType() {
		case A_COMMAND:
			symbol := parser.symbol()
			dec, _ := strconv.Atoi(symbol)
			// Convert int to binary string
			result = fmt.Sprintf("0%015b", dec)
		case L_COMMAND:
			symbol := parser.symbol()
			// TODO: implement symbol table
			result = symbol
		case C_COMMAND:
			compM := parser.comp()
			destM := parser.dest()
			jumpM := parser.jump()
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
