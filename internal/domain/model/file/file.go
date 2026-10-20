package file

import (
	"time"
)

type Type string

const (
	TypeFile      Type = "file"
	TypeDirectory Type = "directory"
)

func (t Type) String() string {
	return string(t)
}

type File struct {
	Name    string
	Type    Type
	Size    int64
	ModTime time.Time
}

func (f File) IsDir() bool {
	return f.Type == TypeDirectory
}
