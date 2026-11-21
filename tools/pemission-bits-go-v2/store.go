package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

const stateFilePath = "state.json"

type File struct {
	Owner string
	Group string
	Perm  Permission
}

type State struct {
	Users map[string]string
	Files map[string]File
}

func newState() *State {
	return &State{
		Users: map[string]string{},
		Files: map[string]File{},
	}
}

func (s *State) HasGroup(group string) bool {
	for _, userGroup := range s.Users {
		if userGroup == group {
			return true
		}
	}
	return false
}

type userRecord struct {
	Name  *string `json:"name"`
	Group *string `json:"group"`
}

type fileRecord struct {
	Name       *string `json:"name"`
	Owner      *string `json:"owner"`
	Group      *string `json:"group"`
	Permission *int    `json:"permission"`
}

type stateRecord struct {
	Users []userRecord `json:"users"`
	Files []fileRecord `json:"files"`
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newState(), nil
		}
		return nil, fmt.Errorf("%s を読み込めません: %w", path, err)
	}

	var record stateRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("%s の JSON が不正です: %w", path, err)
	}

	state, err := buildState(record)
	if err != nil {
		return nil, fmt.Errorf("%s の内容が不正です: %w", path, err)
	}
	return state, nil
}

func requireText(value *string, label string) (string, error) {
	if value == nil || *value == "" {
		return "", fmt.Errorf("%s がありません(または空です)", label)
	}
	return *value, nil
}

func buildState(record stateRecord) (*State, error) {
	if record.Users == nil {
		return nil, errors.New(`必須項目 "users" がありません`)
	}
	if record.Files == nil {
		return nil, errors.New(`必須項目 "files" がありません`)
	}

	state := newState()

	for i, item := range record.Users {
		name, err := requireText(item.Name, fmt.Sprintf("users[%d].name", i))
		if err != nil {
			return nil, err
		}
		group, err := requireText(item.Group, fmt.Sprintf("users[%d].group", i))
		if err != nil {
			return nil, err
		}
		if _, duplicated := state.Users[name]; duplicated {
			return nil, fmt.Errorf("ユーザー名 %q が重複しています", name)
		}
		state.Users[name] = group
	}

	for i, item := range record.Files {
		label := fmt.Sprintf("files[%d]", i)

		name, err := requireText(item.Name, label+".name")
		if err != nil {
			return nil, err
		}
		owner, err := requireText(item.Owner, label+".owner")
		if err != nil {
			return nil, err
		}
		group, err := requireText(item.Group, label+".group")
		if err != nil {
			return nil, err
		}
		if item.Permission == nil {
			return nil, fmt.Errorf("%s (%s) にPermissionがありません", label, name)
		}

		value := *item.Permission
		if value < 0 || value > int(AllBits) {
			return nil, fmt.Errorf("%s (%s)のpermission %d は範囲外です(0~%d)", label, name, value, int(AllBits))
		}

		if _, duplicated := state.Files[name]; duplicated {
			return nil, fmt.Errorf("ファイル名 %q が重複しています", name)
		}

		if _, exists := state.Users[owner]; !exists {
			return nil, fmt.Errorf("%s (%s) の owner %q は登録されていないユーザーです", label, name, owner)
		}
		if !state.HasGroup(group) {
			return nil, fmt.Errorf("%s (%s) の group %q はどのユーザーも使っていません", label, name, group)
		}
		state.Files[name] = File{Owner: owner, Group: group, Perm: Permission(value)}
	}
	return state, nil
}

func ptr[T any](value T) *T {
	return &value
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func saveState(path string, state *State) error {
	record := stateRecord{
		Users: make([]userRecord, 0, len(state.Users)),
		Files: make([]fileRecord, 0, len(state.Files)),
	}
	for _, name := range sortedKeys(state.Users) {
		record.Users = append(record.Users, userRecord{
			Name:  ptr(name),
			Group: ptr(state.Users[name]),
		})
	}
	for _, name := range sortedKeys(state.Files) {
		file := state.Files[name]
		record.Files = append(record.Files, fileRecord{
			Name:       ptr(name),
			Owner:      ptr(file.Owner),
			Group:      ptr(file.Group),
			Permission: ptr(int(file.Perm)),
		})
	}

	data, err := json.MarshalIndent(record, "", " ")
	if err != nil {
		return fmt.Errorf("JSON への変換に失敗しました: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("%sに書き込めません: %w", path, err)
	}
	return nil
}
