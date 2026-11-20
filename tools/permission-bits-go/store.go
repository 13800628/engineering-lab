package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

// 検証で使う用のjsonファイル
const stateFilePath = "state.json"

type Files map[string]Permission

// ファイル内の形式
type fileRecord struct {
	Name       *string `json:"name"`
	Permission *int    `json:"permisson"`
}

type stateRecord struct {
	Files []fileRecord `json:"files"`
}

// state.jsonを読み込んで検証し、内部表現に変換する
// state.jsonがないとここは動かない
func loadState(path string) (Files, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// ファイルがないこと自体は初回では仕方ない
		// 読み込んだ上、読み込み処理での失敗ならエラーとする
		if errors.Is(err, os.ErrNotExist) {
			return Files{}, nil
		}
		return nil, fmt.Errorf("%s を読み込めません: %w", path, err)
	}

	var record stateRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("%s の JSON が不正です: %w", path, err)
	}

	if record.Files == nil {
		return nil, fmt.Errorf("%s に必須項目 \"files\" がありません", path)
	}

	files := Files{}
	for i, item := range record.Files {
		if item.Name == nil || *item.Name == "" {
			return nil, fmt.Errorf("%s: files[%d] に name がありません", path, i)
		}
		if item.Permission == nil {
			return nil, fmt.Errorf("%s: files[%d] (%s) に permission がありません", path, i, *item.Name)
		}

		// 範囲チェック(たまためのことを防ぐためキャスト前に検証)
		value := *item.Permission
		if value < 0 || value > int(AllBits) {
			return nil, fmt.Errorf("%s: files[%d] (%s) の permission %d は範囲外です (0〜%d)", path, i, *item.Name, value, int(AllBits))
		}

		// 名前の被り
		if _, duplicated := files[*item.Name]; duplicated {
			return nil, fmt.Errorf("%s: name %q が重複しています", path, *item.Name)
		}

		files[*item.Name] = Permission(value)
	}
	return files, nil
}

// 内部表現をJSONにしてstate.jsonに書き戻す
func saveState(path string, fiels Files) error {
	names := make([]string, 0, len(fiels))
	for name := range fiels {
		names = append(names, name)
	}
	sort.Strings(names)

	// 空の場合nilにするとガード節で弾かれるので空スライスにする
	record := stateRecord{Files: make([]fileRecord, 0, len(names))}
	for _, name := range names {
		n := name
		v := int(fiels[name])
		record.Files = append(record.Files, fileRecord{Name: &n, Permission: &v})
	}

	data, err := json.MarshalIndent(record, "", " ")
	if err != nil {
		return fmt.Errorf("JSON への変換に失敗しました: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("%s に書き込めません: %w", path, err)
	}
	return nil
}
