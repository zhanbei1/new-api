package common

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type verificationValue struct {
	code string
	time time.Time
}

const (
	EmailVerificationPurpose = "v"
	PasswordResetPurpose     = "r"
	PhoneVerificationPurpose = "p"
	PhoneLoginPurpose        = "pl"
)

const verificationRedisKeyPrefix = "verification:"

var verificationMutex sync.Mutex
var verificationMap map[string]verificationValue
var verificationMapMaxSize = 10
var VerificationValidMinutes = 10

func GenerateVerificationCode(length int) string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	if length == 0 {
		return code
	}
	return code[:length]
}

func verificationStorageKey(purpose, key string) string {
	return purpose + key
}

func verificationRedisKey(purpose, key string) string {
	return verificationRedisKeyPrefix + purpose + ":" + key
}

func RegisterVerificationCodeWithKey(key string, code string, purpose string) {
	ttl := time.Duration(VerificationValidMinutes) * time.Minute
	if RedisEnabled {
		if err := RedisSet(verificationRedisKey(purpose, key), code, ttl); err == nil {
			return
		}
		SysError(fmt.Sprintf("failed to store verification code in Redis, falling back to memory: purpose=%s", purpose))
	}

	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap[verificationStorageKey(purpose, key)] = verificationValue{
		code: code,
		time: time.Now(),
	}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
}

func VerifyCodeWithKey(key string, code string, purpose string) bool {
	if RedisEnabled {
		stored, err := RedisGet(verificationRedisKey(purpose, key))
		if err == nil {
			return stored == code
		}
	}

	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	value, okay := verificationMap[verificationStorageKey(purpose, key)]
	now := time.Now()
	if !okay || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		return false
	}
	return code == value.code
}

func DeleteKey(key string, purpose string) {
	if RedisEnabled {
		_ = RedisDel(verificationRedisKey(purpose, key))
	}
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	delete(verificationMap, verificationStorageKey(purpose, key))
}

// no lock inside, so the caller must lock the verificationMap before calling!
func removeExpiredPairs() {
	now := time.Now()
	for key := range verificationMap {
		if int(now.Sub(verificationMap[key].time).Seconds()) >= VerificationValidMinutes*60 {
			delete(verificationMap, key)
		}
	}
}

func init() {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap = make(map[string]verificationValue)
}
