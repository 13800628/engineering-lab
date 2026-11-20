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

func ParseBitName(name string) (Permission, error) {
	bit, ok := bitByName[name]
	if !ok {
		return 0, fmt.Errorf("不正なビット名です: %q (owner-r 〜 other-x の9種類から選んでください)", name)
	}
	return bit, nil
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
			//何もしない
		case '1':
			result |= 1 // 空いた最下位ビットを1
		default:
			return 0, fmt.Errorf("9bit文字列は 0 と 1 だけで指定してください: %q", text)
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
	owner := (p >> 6) & 0b111
	group := (p >> 3) & 0b111
	other := p & 0b111
	return fmt.Sprintf("%d%d%d", owner, group, other)
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

	for shift := 6; shift >= 0; shift -= 3 {
		triple := (p >> shift) & 0b111
		writeFlag(triple&0b100 != 0, 'r')
		writeFlag(triple&0b010 != 0, 'w')
		writeFlag(triple&0b001 != 0, 'x')
	}
	return sb.String()
}
