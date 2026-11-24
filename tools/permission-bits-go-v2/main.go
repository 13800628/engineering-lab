package main

import (
	"errors"
	"fmt"
	"os"
)

const usage = `使い方: perm <subcommand> <args...>
  user add <user> <group>    ユーザーを登録する
  create <file> <user>       権限 000 のファイルを作る (owner=user, group=userの所属グループ)
  chown  <file> <user>       owner を変更する
  chgrp  <file> <group>      group を変更する
  set    <file> <9bit>       9bit を直接代入する (例: 110100000)
  on     <file> <bit>        指定ビットを立てる
  off    <file> <bit>        指定ビットを落とす
  flip   <file> <bit>        指定ビットを反転する
  can    <file> <bit>        指定ビットが立っているか (区分は見ない)
  show   <file>              owner / group / 9bit / 8進数 / rwx を表示する
  check  <user> <file> <r|w|x>  そのユーザーが操作できるか

ビット名: owner-r owner-w owner-x group-r group-w group-x other-r other-w other-x`

var expectedArgCount = map[string]int{
	"user add": 2,
	"create":   2,
	"chown":    2,
	"chgrp":    2,
	"set":      2,
	"on":       2,
	"off":      2,
	"flip":     2,
	"can":      2,
	"show":     1,
	"check":    3,
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	subcommand, rest := args[0], args[1:]

	if subcommand == "user" {
		if len(rest) == 0 || rest[0] != "add" {
			return fmt.Errorf("user のサブコマンドは add のみです\n%s", usage)
		}
		subcommand, rest = "user add", rest[1:]
	}

	want, known := expectedArgCount[subcommand]
	if !known {
		return fmt.Errorf("不明なサブコマンドです: %q\n%s", subcommand, usage)
	}
	if len(rest) != want {
		return fmt.Errorf("%s の引数の数が正しくありません\n%s", subcommand, usage)
	}

	state, err := loadState(stateFilePath)
	if err != nil {
		return err
	}

	var output string
	var changed bool
	switch subcommand {
	case "user add":
		output, changed, err = runUserAdd(state, rest)
	case "create":
		output, changed, err = runCreate(state, rest)
	case "chown":
		output, changed, err = runChown(state, rest)
	case "chgrp":
		output, changed, err = runChgrp(state, rest)
	case "check":
		output, changed, err = runCheck(state, rest)
	default:
		output, changed, err = runFileOperation(state, subcommand, rest)
	}
	if err != nil {
		return err
	}

	if changed {
		if err := saveState(stateFilePath, state); err != nil {
			return err
		}
	}
	fmt.Print(output)
	return nil
}

func formatFile(name string, file File) string {
	return fmt.Sprintf("file : %s\nowner: %s\ngroup: %s\nbits : %s\noctal: %s\nrwx  : %s\n",
		name, file.Owner, file.Group,
		file.Perm.BitString(), file.Perm.Octal(), file.Perm.Rwx())
}

func lookupFile(state *State, name string) (File, error) {
	file, exists := state.Files[name]
	if !exists {
		return File{}, fmt.Errorf("存在しないファイルです: %s", name)
	}
	return file, nil
}

func lookupUserGroup(state *State, userName string) (string, error) {
	group, exists := state.Users[userName]
	if !exists {
		return "", fmt.Errorf("存在しないユーザーです: %s", userName)
	}
	return group, nil
}

func runUserAdd(state *State, args []string) (string, bool, error) {
	name, group := args[0], args[1]
	if name == "" || group == "" {
		return "", false, errors.New("ユーザー名とグループ名は空にできません")
	}
	if _, exists := state.Users[name]; exists {
		return "", false, fmt.Errorf("すでに登録されているユーザーです: %s", name)
	}
	state.Users[name] = group
	return fmt.Sprintf("ユーザーを登録しました: %s (group: %s)\n", name, group), true, nil
}

func runCreate(state *State, args []string) (string, bool, error) {
	fileName, userName := args[0], args[1]
	if fileName == "" {
		return "", false, errors.New("ファイル名は空にできません")
	}
	if _, exists := state.Files[fileName]; exists {
		return "", false, fmt.Errorf("すでに存在します: %s", fileName)
	}
	group, err := lookupUserGroup(state, userName)
	if err != nil {
		return "", false, err
	}

	file := File{Owner: userName, Group: group, Perm: 0}
	state.Files[fileName] = file
	return formatFile(fileName, file), true, nil
}

func runChown(state *State, args []string) (string, bool, error) {
	fileName, userName := args[0], args[1]
	file, err := lookupFile(state, fileName)
	if err != nil {
		return "", false, err
	}

	if _, err := lookupUserGroup(state, userName); err != nil {
		return "", false, err
	}

	file.Owner = userName
	state.Files[fileName] = file
	return formatFile(fileName, file), true, nil
}

func runChgrp(state *State, args []string) (string, bool, error) {
	fileName, group := args[0], args[1]
	file, err := lookupFile(state, fileName)
	if err != nil {
		return "", false, err
	}
	if !state.HasGroup(group) {
		return "", false, fmt.Errorf("どのユーザーも使っていないグループです: %s", group)
	}

	file.Group = group
	state.Files[fileName] = file
	return formatFile(fileName, file), true, nil
}

func runCheck(state *State, args []string) (string, bool, error) {
	userName, fileName, opName := args[0], args[1], args[2]

	userGroup, err := lookupUserGroup(state, userName)
	if err != nil {
		return "", false, err
	}
	file, err := lookupFile(state, fileName)
	if err != nil {
		return "", false, err
	}
	op, err := ParseOperation(opName)
	if err != nil {
		return "", false, err
	}

	class := Classify(userName, userGroup, file.Owner, file.Group)
	return fmt.Sprintln(file.Perm.Allows(class, op)), false, nil
}

func runFileOperation(state *State, subcommand string, args []string) (string, bool, error) {
	fileName := args[0]
	file, err := lookupFile(state, fileName)
	if err != nil {
		return "", false, err
	}
	changed := false

	switch subcommand {
	case "set":
		newPerm, err := ParseBits(args[1])
		if err != nil {
			return "", false, err
		}
		file.Perm = newPerm
		changed = true

	case "on", "off", "flip", "can":
		bit, err := ParseBitName(args[1])
		if err != nil {
			return "", false, err
		}
		switch subcommand {
		case "on":
			file.Perm = file.Perm.With(bit)
			changed = true
		case "off":
			file.Perm = file.Perm.Without(bit)
			changed = true
		case "flip":
			file.Perm = file.Perm.Flipped(bit)
			changed = true
		case "can":
			return fmt.Sprintln(file.Perm.Has(bit)), false, nil
		}
	case "show":

	}

	if changed {
		state.Files[fileName] = file
	}
	return formatFile(fileName, file), changed, nil
}
