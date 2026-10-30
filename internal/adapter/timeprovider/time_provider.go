package timeprovider

import (
	"time"
)

type TimeProvider struct {
}

func New() *TimeProvider {
	return &TimeProvider{}
}

func (t *TimeProvider) Now() time.Time {
	return time.Now()
}
