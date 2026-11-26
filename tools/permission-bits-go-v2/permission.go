package main

import (
	"fmt"
	"strings"
)

type Permission uint16

const (
	OtherExecute Permission = 1 << iota
	OtherWrite
	OtherRead
	GroupExecute
	GroupWrite
	GroupRead
	OwnerExecute
	OwnerWrite
	OwnerRead
)

const AllBits Permission = 1<<9 - 1

// 3bit中の1bitをマスク。checkコマンドのr/w/xに対応
const (
	OpExecute Permission = 1 << iota // 001=1
	OpWrite                          // 010=2
	OpRead                           // 100=4
)

// 誰として見るかのためのもの
type Class int

const (
	ClassOwner Class = iota
	ClassGroup
	ClassOther
)

// 処理するための3bitを最下位にまで持ってくるためのもの
// owner(8-6) group(5-3) other(3-0)
func (c Class) shift() uint {
	return uint(2-c) * 3
}

func (p Permission) Triple(c Class) Permission {
	return (p >> c.shift()) & 0b111
}

// 指定された区分(owner, group, other)での操作opが許可されるのかを返す
func (p Permission) Allows(c Class, op Permission) bool {
	return p.Triple(c)&op != 0
}

// ユーザーがファイルに対してどの区分なのかを決めるもの
// 一つが決まれば他は見ない
func Classify(userName, userGroup, fileOwner, filerGroup string) Class {
	if userName == fileOwner {
		return ClassOwner
	}
	if userGroup == filerGroup {
		return ClassGroup
	}
	return ClassOther
}

var bitByName = map[string]Permission{
	"owner-r": OwnerRead,
	"owner-w": OwnerWrite,
	"owner-x": OwnerExecute,
	"group-r": GroupRead,
	"group-w": GroupWrite,
	"group-x": GroupExecute,
	"other-r": OtherRead,
	"other-w": OtherWrite,
	"other-x": OtherExecute,
}

func ParseBitName(name string) (Permission, error) {
	bit, ok := bitByName[name]
	if !ok {
		return 0, fmt.Errorf("不正なビット名です: %q (owner-r 〜 other-x の9種類から選んでください)", name)
	}
	return bit, nil
}

// checkコマンドのrwxと3bit内のマスクの対応表
var operationByName = map[string]Permission{
	"r": OpRead,
	"w": OpWrite,
	"x": OpExecute,
}

func ParseOperation(name string) (Permission, error) {
	op, ok := operationByName[name]
	if !ok {
		return 0, fmt.Errorf("不正な操作です: %q (r, w, x のいずれかを指定してください)", name)
	}
	return op, nil
}

func ParseBits(text string) (Permission, error) {
	if len(text) != 9 {
		return 0, fmt.Errorf("9bit文字列は9文字で指定してください: %q", text)
	}

	var result Permission
	for _, ch := range text {
		result <<= 1
		switch ch {
		case '0':

		case '1':
			result |= 1
		default:
			return 0, fmt.Errorf("9bit文字列は0と1だけで指定してください: %q", text)
		}
	}
	return result, nil
}

func (p Permission) With(bit Permission) Permission {
	return p | bit
}

func (p Permission) Without(bit Permission) Permission {
	return p &^ bit
}

func (p Permission) Flipped(bit Permission) Permission {
	return p ^ bit
}

func (p Permission) Has(bit Permission) bool {
	return p&bit != 0
}

func (p Permission) BitString() string {
	var sb strings.Builder
	for i := 8; i >= 0; i-- {
		if (p>>i)&1 == 1 {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

func (p Permission) Octal() string {
	return fmt.Sprintf("%d%d%d",
		p.Triple(ClassOwner), p.Triple(ClassGroup), p.Triple(ClassOther))
}

func (p Permission) Rwx() string {
	var sb strings.Builder

	writeFlag := func(isOn bool, letter byte) {
		if isOn {
			sb.WriteByte(letter)
		} else {
			sb.WriteByte('-')
		}
	}

	for class := ClassOwner; class <= ClassOther; class++ {
		triple := p.Triple(class)
		writeFlag(triple&OpRead != 0, 'r')
		writeFlag(triple&OpWrite != 0, 'w')
		writeFlag(triple&OpExecute != 0, 'x')
	}
	return sb.String()
}
