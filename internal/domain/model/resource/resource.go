package resource

import (
	"context"
	"path/filepath"
)

const (
	publicUser = "public"
)

type contextResouce string

const (
	contextResourceKey contextResouce = "contextResourceKey"
)

type Resource struct {
	User string
	Path string
}

func (r Resource) IsPublic() bool {
	return r.User == publicUser
}

func (r Resource) FullPath() string {
	return filepath.Join(r.User, r.Path)
}

func WithContext(ctx context.Context, res Resource) context.Context {
	return context.WithValue(ctx, contextResourceKey, res)
}

func FromContext(ctx context.Context) (Resource, bool) {
	res, ok := ctx.Value(contextResourceKey).(Resource)

	return res, ok
}
