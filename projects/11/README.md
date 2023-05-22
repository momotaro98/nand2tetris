
## 実装したコンパイラの動作方法

```
go run *.go -path=./Average
```

This generates vm code to `./Average/Main.vm`.

## OSとしてのVMコード

それぞれのJackプロジェクト(PongやSquare)などはキーボード、スクリーン操作、メモリ管理をするVMコードのfunctionを呼んでいる。

前提としてそれらのOS的な機能のVMコードは本書提供プロジェクトのtool群に含まれており、それらVMコードと一緒に動作させることでVMランタイムで動く。

OSとしてのVMコードは [./OS](./OS) にコピーで配置した。

## コンパイラ 理解

データ変換とコマンド変換の2つに分けられる。

### 11.1.1 データ変換

コンパイラのデータ変換においてやることが以下の2点

* 変数を型ごとに対応するプラットフォームの表現に変換する
* 変数のライフサイクルとスコープを管理する

この2点の機能のためのコンパイラ内のモジュールがシンボルテーブルである。

#### シンボルテーブル

一回のコンパイルで以下のひとつのシンボルテーブルのオブジェクトを1つ持つ。

```go
type SymbolTable struct {
	classTable      map[string][]interface{}
	subroutineTable map[string][]interface{}
	staticIndex     int
	fieldIndex      int
	argIndex        int
	varIndex        int
}
```

`classTable`にはClassが入り、`subroutineTable`はfunction, methodのサブルーチンが入る。

`classTable`、`subroutineTable` の`map[string][]interface{}`には以下が入る。

* 変数名 (i, sum)
* 型 (int, string, boolean)
* 属性 (static, field, argument, local)
* 番号 (0, 1, 2) [index] ← そのスコープ内にて何番目か

#### 変数操作

変数のメモリ割当が必要であるがその仕事はVMマシン側のスタックマシンで解決済みである。

#### 配列操作

Jack言語側で以下のコードがあるとき

```java
// 事前に bar = new int[10] されている前提で // ベースアドレスが設定される
bar[k] = 19
```

これは `*(bar+k)=19` を意味する。

以下のVMコードになる。

```
push local 2     // push bar (ベースアドレス)
push argument 0  // push k
add
// x[k]にアクセスするためにthatセグメントを使う
pop pointer 1    // pointer 1 は動的であるthat特殊変数のベースアドレス
push constant 19
pop that 0       // pointer 1 が指すアドレス場所の値に19が入る。よって *(bar+k) に19が入る。
```

#### オブジェクト操作

オブジェクトはフィールドとメソッドから成り立つがコンパイル側での扱いは全く異なる。

フィールドはメモリ割り当てが必要な配列と同等な扱いになり、メソッドはオブジェクト自体を引数とするfunctionと同等な扱いになる。

### 11.1.2 コマンド変換

#### 式の評価

パーサ側で構築した構文木(AST)を逆ポーランド表記法で出力する。

Jackソースコードでの式

```java
x + g(2,y,-z) * 5
```

以下に変換される

```
push x
push 2
push y
push z
neg
call g
push 5
call multiply
add
```

#### フロー制御

```java
while (cond)
  s1
```

```
label L1
  ~(cond)を計算するVMコード
  if-goto L2
  s1
  goto L1
label L2
```