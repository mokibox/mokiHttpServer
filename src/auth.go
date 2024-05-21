package src

import (
	"errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// 认证密钥
var authCode = os.Getenv("AUTH_CODE")

var (
	sessionStore   = make(map[string]time.Time)
	pathMd5Store   = make(map[string]string)
	mu             sync.Mutex
	sessionTimeout = 30 * time.Minute // Session 超时时间设置为30分钟
)

// 验证cookie中密钥是否正取
func authToCookie(cookie *http.Cookie) error {
	if authCode == "" {
		logrus.Warning("cookie验证, 验证密码未设置!")
		return nil
	}
	if cookie == nil {
		logrus.Error("cookie验证, cookie为空!")
		return errors.New("未验证, 请先进行验证! ")
	}
	sessionId := cookie.Value
	sessionTime := sessionStore[sessionId]
	// 判断是否过期
	if sessionTime.Add(sessionTimeout).Before(time.Now()) {
		mu.Lock()
		defer mu.Unlock()
		delete(sessionStore, sessionId)
		logrus.Error("cookie验证, session已经过期!")
		return errors.New("验证已过期, 请重新验证! ")
	}
	logrus.Info("cookie验证, 验证通过!")
	return nil
}

// 验证密码并下发cookie
func authToCode(valAuthCode string, now time.Time) (*http.Cookie, error) {
	if authCode == "" {
		logrus.Warning("密码验证, 验证密码未设置!")
		return nil, nil
	}
	if valAuthCode == generateHash(authCode, -1) {
		id := generateSessionID()
		pathMd5 := generateHash(authCode+strconv.FormatInt(now.Unix(), 10), 8)
		cookie := &http.Cookie{
			Name:  "session_id",
			Value: id,
		}
		mu.Lock()
		defer mu.Unlock()
		sessionStore[id] = now
		pathMd5Store[pathMd5] = "saved"
		logrus.Info("密码验证, 验证已通过! ")
		return cookie, nil
	}
	logrus.Error("密码验证, 验证密码错误!")
	return nil, errors.New("验证密码错误! ")
}

// 验证路径中的码
func authToPath(pathCode string) (bool, string) {
	if authCode == "" {
		logrus.Warning("路径验证, 验证密码未设置!")
		return true, ""
	}
	_, exists := pathMd5Store[pathCode]
	if exists {
		logrus.Info("路径验证, 验证通过!")
		return true, "验证通过! "
	}
	logrus.Error("路径验证, 密码过期或者被篡改过!")
	return false, "验证密码被篡改或过期! "
}
