package model

import "fmt"

type Task struct {
	Task string   `json:"task"`
	Type FileType `json:"type"`
}

type Tasks []Task

func (t Tasks) TypeByID(id int) (FileType, error) {
	if id >= len(t) {
		return FileTypeUnknown, fmt.Errorf("invalid taskID")
	}
	return t[id].Type, nil
}
