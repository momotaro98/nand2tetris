package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	file, err := os.Open("./add/Add.asm")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	parser := NewParser(file)
	code := NewCode()

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
		fmt.Println(result)
	}
}
