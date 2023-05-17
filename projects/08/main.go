package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	pathName = flag.String("path", ".", "file name or dir name where vm file exists")
)

func main() {
	flag.Parse()
	path := *pathName

	if strings.HasSuffix(path, ".vm") { // File
		outputPath := strings.TrimSuffix(path, ".vm") + ".asm"
		codeWriter := NewCodeWriter(outputPath)
		defer codeWriter.Close()

		translateFile(path, codeWriter)
		fmt.Println("Translated to", outputPath)
	} else { // Directory
		if !strings.HasSuffix(path, "/") {
			path = path + "/"
		}
		d := filepath.Dir(path)
		b := filepath.Base(path)
		outputPath := filepath.Join(d, b) + ".asm"
		codeWriter := NewCodeWriter(outputPath)
		defer codeWriter.Close()

		files, err := filepath.Glob(filepath.Join(path, "/*"))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		for _, file := range files {
			if strings.HasSuffix(file, ".vm") {
				translateFile(file, codeWriter)
			}
		}

		fmt.Println("Translated to", outputPath)
	}
}

func translateFile(file string, codeWriter CodeWriter) {
	parser := NewParser(file)
	defer parser.Close()
	codeWriter.SetFileName(strings.Split(filepath.Base(file), ".")[0])
	for parser.hasMoreCommands() {
		parser.advance()

		switch parser.commandType() {
		case C_ARITHMETIC:
			codeWriter.WriteArithmetic(parser.arg1())
		case C_PUSH:
			codeWriter.WritePushPop(C_PUSH, parser.arg1(), parser.arg2())
		case C_POP:
			codeWriter.WritePushPop(C_POP, parser.arg1(), parser.arg2())
		case C_LABEL:
			codeWriter.WriteLabel(parser.arg1())
		// case C_GOTO:
		// 	codeWriter.WriteGoto(parser.arg1())
		case C_IF:
			codeWriter.WriteIf(parser.arg1())
			// case C_FUNCTION:
			// 	codeWriter.WriteFunction(parser.arg1(), parser.arg2())
			// case C_RETURN:
			// 	codeWriter.WriteReturn()
			// case C_CALL:
			// 	codeWriter.WriteCall(parser.arg1(), parser.arg2())
		}

	}
	codeWriter.Flush()
}
