package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Demo struct {
	inputFile   string
	outputFile  string
	isDirectory bool
}

func NewDemo(inputFile string) *Demo {
	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		panic(err)
	}

	isDirectory := fileInfo.IsDir()

	return &Demo{
		inputFile:   inputFile,
		isDirectory: isDirectory,
	}
}

func (d *Demo) GetInputFile() string {
	return d.inputFile
}

func (d *Demo) GetOutputFile() string {
	return d.outputFile
}

func (d *Demo) IsDirectory() bool {
	return d.isDirectory
}

func (d *Demo) Compile() {
	if d.isDirectory {
		d.compileDirectory()
	} else {
		d.compileFile()
	}
}

func (d *Demo) compileDirectory() {
	files, err := ioutil.ReadDir(d.inputFile)
	if err != nil {
		panic(err)
	}

	fileList := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jack") {
			fileList = append(fileList, filepath.Join(d.inputFile, file.Name()))
		}
	}

	for _, file := range fileList {
		f, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		cpe, err := NewCompilationEngine(f)
		if err != nil {
			panic(err)
		}
		cpe.CompileClass()
	}
}

func (d *Demo) compileFile() {
	f, err := os.Open(d.inputFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cpe, err := NewCompilationEngine(f)
	if err != nil {
		panic(err)
	}
	cpe.CompileClass()
}

var (
	inputFilePath = flag.String("path", "./ConvertToBin/Main.jack", "file name path")
)

func main() {
	flag.Parse()
	demo := NewDemo(*inputFilePath)
	demo.Compile()
}
