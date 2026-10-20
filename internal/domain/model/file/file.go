package file

import (
	"time"
)

type File struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}
