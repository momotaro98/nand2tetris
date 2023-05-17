## 実装したVM変換プログラムの動作方法

```
go run *.go -path=./StackArithmetic/SimpleAdd/SimpleAdd.vm
```

This generates asm code to `./StackArithmetic/SimpleAdd/SimpleAdd.asm`.

## スタックマシン 理解

スタックメモリ領域はグローバルに1つだけ存在する。

local,this,などそれぞれのスコープで専用のメモリ領域が存在する。

## VM code 理解

```
push local 52       // localメモリ領域のindex 57の値をスタックに積む
push this 2         // thisメモリ領域のindex 2の値をスタックに積む
add                 // スタックメモリ領域上で演算
pop local 53        // 現スタック値をlocalメモリ領域のindex 53に格納する
push constant 3000  // 3000という数値がスタックに積まれる。(constantにはメモリ領域は存在しない)
pop pointer 1       // pointerのindex 1はthisメモリ領域の先頭アドレスを指す。そのためthisメモリ領域の先頭アドレスが3000に移動した。
```

## VM translation 理解

vm code

```
push constant 7
push constant 8
add
```

translated assembly code

```
@7     // start push constant 7
D=A    // 数値7をDに格納
@SP    // 現スタック先頭アドレスが格納されている特殊アドレス(SP)をAレジスタに格納
A=M    // M=Mem[A]=Mem[SP]なので、現スタック先頭アドレスがAレジスタに格納される
M=D    // M=Mem[A]=Mem[現スタック先頭アドレス]なので、現スタックの値がDレジスタの値になる
@SP    // SPをAレジスタに格納
M=M+1  // 現スタック先頭アドレスがインクリメントされる
@8     // start push constant 8
D=A
@SP
A=M
M=D
@SP
M=M+1
@SP     // start add
M=M-1   // 現スタックアドレスをデクリメントする
A=M     // Aレジスタに現スタックアドレスが入る
D=M     // D=M=Mem[A]=Mem[現スタックアドレス]=現スタック値
@SP
M=M-1
A=M
D=D+M   // D=8+7
@SP
A=M
M=D     // 現スタック値が15になる
@SP
M=M+1
```