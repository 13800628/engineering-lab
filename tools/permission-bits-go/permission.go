package main

import (
	"fmt"
	"strings"
)

// 学習のためにuint16でもつ
// だが扱うのは下位9bitだけで上位7bitは常に0にする
type Permission uint16

// iotaはconst内で0から始まり、連番で増えるもの(`1 << iota` だと1をiota回左にシフトさせる)
// other, ownerの順にしているのは、rwx-rwx-rwx の左端(Owner Read)が最上位ビットに対応するようにする、9bit文字列と数値の対応が2進数表記と一致(学習なので)
const (
	OtherExecute Permission = 1 << iota // 1 << 0 = 000000001
	OtherWrite                          // 1 << 1 = 000000010
	OtherRead
	GroupExecute
	GroupWrite
	GroupRead
	OwnerExecute
	OwnerWrite
	OwnerRead // 1 << 8 = 100000000
)

// 全てがたった値
// 1 << 9は10bit(1000000000)なので、1を引くと下位が全て1となる(0111111111)
// 読み込み時: 上限を調べるために扱う
const AllBits Permission = 1<<9 - 1

// CLIで使うビット名と対応するビット定数の対応表
// 入力時: 内部表現に変換する入り口となる
var bitByName = map[string]Permission{
	"owner-x": OwnerExecute,
	"owner-w": OwnerWrite,
	"owner-r": OwnerRead,
	"group-x": GroupExecute,
	"group-w": GroupWrite,
	"group-r": GroupRead,
	"other-r": OtherRead,
	"other-w": OtherWrite,
	"other-x": OtherExecute,
}

// ビット名をPermissionに変換する
// 対応していないものを入力された際は0にすると危険なのでエラーを返すように
func ParseBitName(name string) (Permission, error) {
	bit, ok := bitByName[name]
	if !ok {
		return 0, fmt.Errorf("不正なビット名です: %q (owner-r 〜 other-x の9種類から選んでください)", name)
	}
	return bit, nil
}

// 9bit文字列を数値に変換する
// 他に簡単に表現できる書き方はあるが、一桁ずつ左にずらしながら取り込むというものを明示するため
func ParseBits(text string) (Permission, error) {
	// 入力は短くなることもなりうるのでそれを意図しない権限で受け取らないようにするために厳密に9桁
	if len(text) != 9 {
		return 0, fmt.Errorf("9bit文字列は9文字で指定してください: %q", text)
	}

	var result Permission
	for _, ch := range text {
		result <<= 1 // 既に取り込んだ桁を1つ上にずらして最下位に空きを作る
		switch ch {
		case '0':
			//空いた最下位は0のまま何もしない
		case '1':
			result |= 1 // 空いた最下位ビットだけを1
		default:
			return 0, fmt.Errorf("9bit文字列は 0 と 1 だけで指定してください: %q", text)
		}
	}
	return result, nil
}

// OR
func (p Permission) With(bit Permission) Permission {
	return p | bit
}

// AND NOT: &^(a & (^b))
func (p Permission) Without(bit Permission) Permission {
	return p &^ bit
}

// XOR
func (p Permission) Flipped(bit Permission) Permission {
	return p ^ bit
}

// AND
func (p Permission) Has(bit Permission) bool {
	return p&bit != 0
}

// 9bitの2進数文字列を返す
// 最上位から1bitずつ取り出して並べる
func (p Permission) BitString() string {
	var sb strings.Builder
	for i := 8; i >= 0; i-- {
		// p を i桁右にずらすと見たいbitが必然と最下位にくる
		// そこに&1をかけると、最下位1bitだけが残る
		if (p>>i)&1 == 1 {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

// 3桁の8進数表記を返す
// 3bit区切りで変換する
func (p Permission) Octal() string {
	owner := (p >> 6) & 0b111
	group := (p >> 3) & 0b111
	other := p & 0b111
	return fmt.Sprintf("%d%d%d", owner, group, other)
}

// "rw-r--r--"のような形式の9文字を返す
// 3bitずつ取り出し、各ビットをr(4), w(2), x(1)に対応させて文字にする
func (p Permission) Rwx() string {
	var sb strings.Builder

	// 立っていれば文字(r, w, x)、立ってなければ'-'を書き込む(補助関数として実装)
	writeFlag := func(isOn bool, letter byte) {
		if isOn {
			sb.WriteByte(letter)
		} else {
			sb.WriteByte('-')
		}
	}

	// owner(6) → group(3) → other(0) の順に 3bit ずつ処理
	for shift := 6; shift >= 0; shift -= 3 {
		triple := (p >> shift) & 0b111
		writeFlag(triple&0b100 != 0, 'r') // '1'00
		writeFlag(triple&0b010 != 0, 'w') // 0'1'0
		writeFlag(triple&0b001 != 0, 'x') // 00'1'
	}
	return sb.String()
}
