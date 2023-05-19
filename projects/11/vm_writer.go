package main

import (
	"bufio"
	"fmt"
	"os"
)

type VMWriter struct {
	writer *bufio.Writer
}

func NewVMWriter(outputFile *os.File) *VMWriter {
	writer := bufio.NewWriter(outputFile)
	return &VMWriter{
		writer: writer,
	}
}

func (vw *VMWriter) write(cmd string) {
	_, err := vw.writer.WriteString(cmd + "\n")
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

func (vw *VMWriter) done() {
	err := vw.writer.Flush()
	if err != nil {
		fmt.Println("Error flushing writer:", err)
	}
}

func (vw *VMWriter) GetWriter() *bufio.Writer {
	return vw.writer
}

func (vw *VMWriter) WritePush(segment string, index int) {
	vw.write(fmt.Sprintf("push %s %d", segment, index))
	vw.done()
}

func (vw *VMWriter) WritePop(segment string, index int) {
	vw.write(fmt.Sprintf("pop %s %d", segment, index))
	vw.done()
}

func (vw *VMWriter) WriteArithmetic(operator string) {
	switch operator {
	case "+":
		vw.write("add")
	case "-":
		vw.write("sub")
	case "--":
		vw.write("neg")
	case "=":
		vw.write("eq")
	case ">":
		vw.write("gt")
	case "<":
		vw.write("lt")
	case "&":
		vw.write("and")
	case "|":
		vw.write("or")
	case "~":
		vw.write("not")
	default:
		panic("Unknown operator!")
	}
	vw.done()
}

func (vw *VMWriter) WriteLabel(label string) {
	vw.write("label " + label)
	vw.done()
}

func (vw *VMWriter) WriteGoto(label string) {
	vw.write("goto " + label)
	vw.done()
}

func (vw *VMWriter) WriteIf(label string) {
	vw.write("if-goto " + label)
	vw.done()
}

func (vw *VMWriter) WriteCall(name string, argsN int) {
	vw.write(fmt.Sprintf("call %s %d", name, argsN))
	vw.done()
}

func (vw *VMWriter) WriteFunction(name string, argsN int) {
	vw.write(fmt.Sprintf("function %s %d", name, argsN))
	vw.done()
}

func (vw *VMWriter) WriteReturn() {
	vw.write("return")
	vw.done()
}

func (vw *VMWriter) Close() {
	err := vw.writer.Flush()
	if err != nil {
		fmt.Println("Error flushing writer:", err)
	}
}
