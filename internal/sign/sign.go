package sign

import (
	"crypto/rand"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/IceWhaleTech/CasaOS/pkg/sign"
)

const SecretKeyPath = "/var/lib/casaos/hmac_secret.key"

var once sync.Once
var instance sign.Sign

func Sign(data string) string {

	return NotExpired(data)

}

func WithDuration(data string, d time.Duration) string {
	once.Do(Instance)
	return instance.Sign(data, time.Now().Add(d).Unix())
}

func NotExpired(data string) string {
	once.Do(Instance)
	return instance.Sign(data, 0)
}

func Verify(data string, sign string) error {
	once.Do(Instance)
	return instance.Verify(data, sign)
}

func Instance() {
	keyPath := SecretKeyPath
	key, err := os.ReadFile(keyPath)
	if err != nil {
		// Generate new random key
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("Failed to generate HMAC secret: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
			log.Fatalf("Failed to create directory for HMAC secret: %v", err)
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			log.Fatalf("Failed to write HMAC secret: %v", err)
		}
	}
	instance = sign.NewHMACSign(key)
}
