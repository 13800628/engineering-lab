package main

import (
	"errors"
	"fmt"
	"os"
)

const usage = `使い方: perm <sucommand> <filename> [argument]

  create <name>           権限 000 の仮想ファイルを作成
	set    <name> <9bit>    9bit を直接代入
	on    <name> <bit>     指定ビットを立てる
	off   <name> <bit>     指定ビットを落とす
	flip    <name> <bit>   指定ビットを反転する
	can   <name> <bit>      指定ビットが立っているかの判定
	show  <name>           9bit 8進数 rwx表記を表示

ビット名: owner-r owner-w owner-x group-r group-w group-x other-r other-w other-x`

var expectedArgCount = map[string]int{
	"create": 0,
	"set":    1,
	"on":     1,
	"off":    1,
	"flip":   1,
	"can":    1,
	"show":   0,
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "エラー: ", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return errors.New(usage)
	}
	subcommand, filename, extra := args[0], args[1], args[2:]

	want, known := expectedArgCount[subcommand]
	if !known {
		return fmt.Errorf("不明なサブコマンドです: %q\n%s", subcommand, usage)
	}
	if len(extra) != want {
		return fmt.Errorf("%s の引数の数が正しくありません\n%s", subcommand, usage)
	}

	files, err := loadState(stateFilePath)
	if err != nil {
		return err
	}

	if subcommand == "create" {
		if _, exists := files[filename]; exists {
			return fmt.Errorf("すでに存在します: %s", filename)
		}
		files[filename] = 0
		if err := saveState(stateFilePath, files); err != nil {
			return nil
		}
		fmt.Printf("作成しました: %s (%s)\n", filename, Permission(0).BitString())
		return nil
	}

	perm, exists := files[filename]
	if !exists {
		return fmt.Errorf("存在しないファイルです: %s", filename)
	}

	changed := false

	switch subcommand {
	case "set":
		newPerm, err := ParseBits(extra[0])
		if err != nil {
			return err
		}
		perm = newPerm
		changed = true
	case "on", "off", "flip", "can":
		bit, err := ParseBitName(extra[0])
		if err != nil {
			return err
		}

		switch subcommand {
		case "on":
			perm = perm.With(bit)
			changed = true
		case "off":
			perm = perm.Without(bit)
			changed = true
		case "flip":
			perm = perm.Flipped(bit)
			changed = true
		case "can":
			fmt.Println(perm.Has(bit))
		}

	case "show":

	}

	if changed {
		files[filename] = perm
		if err := saveState(stateFilePath, files); err != nil {
			return nil
		}
	}

	if subcommand != "can" {
		fmt.Printf("file : %s\n", filename)
		fmt.Printf("bits : %s\n", perm.BitString())
		fmt.Printf("octal: %s\n", perm.Octal())
		fmt.Printf("rwx  : %s\n", perm.Rwx())
	}
	return nil
}
