package auth

import (
	"time"
)

// Claims describes an authorized token holder.
type Claims struct {
	Subject   string
	ExpiresAt time.Time
}
