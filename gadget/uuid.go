package gadget

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

func UUID() string {
	u, _ := uuid.NewRandom()
	return u.String()
}

// RandString 生成随机字符串
func RandString(len int) string {
	rand.Seed(time.Now().UnixNano())

	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := rand.Intn(26) + 65
		bytes[i] = byte(b)
	}

	return string(bytes)
}
