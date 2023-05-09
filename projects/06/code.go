package main

type Mnemonic string

type Code interface {
	dest(n Mnemonic) string // return 3bit
	comp(n Mnemonic) string // return 7bit
	jump(n Mnemonic) string // return 3bit
}

func NewCode() Code {
	return &code{}
}

type code struct {
}

func (c *code) dest(n Mnemonic) string {
	switch string(n) {
	case "null":
		return "000"
	case "M":
		return "001"
	case "D":
		return "010"
	case "MD":
		return "011"
	case "A":
		return "100"
	case "AM":
		return "101"
	case "AD":
		return "110"
	case "AMD":
		return "111"
	}
	return "000"
}

func (c *code) jump(n Mnemonic) string {
	switch string(n) {
	case "null":
		return "000"
	case "JGT":
		return "001"
	case "JEQ":
		return "010"
	case "JGE":
		return "011"
	case "JLT":
		return "100"
	case "JNE":
		return "101"
	case "JLE":
		return "110"
	case "JMP":
		return "111"
	}
	return "000"
}

func (c *code) comp(n Mnemonic) string {
	switch string(n) {
	case "0":
		return "0101010"
	case "1":
		return "0111111"
	case "-1":
		return "0111010"
	case "D":
		return "0001100"
	}
	// TODO: 途中
	return "0000000"
}
