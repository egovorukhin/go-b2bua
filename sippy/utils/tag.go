package sippy_utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenTag() string {
	ltag := make([]byte, 16)
	rand.Read(ltag)
	return hex.EncodeToString(ltag)
}
