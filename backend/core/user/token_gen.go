package user

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/trezcool/masomo/backend/core"
)

var (
	salt    = []byte("masomo.backend.core.user.token_gen")
	NowFunc = time.Now // for test mock

	// errors
	errInvalidToken = errors.New("invalid token")
	errTokenExpired = errors.New("token expired")
)

// EncodeUID base64 encodes given User ID
func EncodeUID(usr User) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(usr.ID)))
}

// decodeUID base64 decodes given UID
func decodeUID(uid string) (int, error) {
	idBytes, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(string(idBytes))
	if err != nil {
		return 0, err
	}
	return id, nil
}

// MakeToken generates a password reset token for a given User.
func MakeToken(usr User) (string, error) {
	return _makeTokenWithTimestamp(usr, _numDaysSince2001(NowFunc()))
}

// verifyToken checks that a password reset token for a given User is valid.
func verifyToken(usr User, token string) error {
	if token == "" {
		return errInvalidToken
	}

	parts := strings.SplitN(token, "-", 2)
	if len(parts) < 2 {
		return errInvalidToken
	}
	tsB32 := parts[0]

	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(tsB32)
	if err != nil {
		return errInvalidToken
	}
	ts, err := strconv.Atoi(string(data))
	if err != nil {
		return errInvalidToken
	}

	// check that token has not been tampered with
	newToken, err := _makeTokenWithTimestamp(usr, ts)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(newToken), []byte(token)) == 0 {
		return errInvalidToken
	}

	// check that the timestamp is within limit
	passwordResetTimeoutDelta := core.Conf.GetDuration("passwordResetTimeoutDelta")
	if (_numDaysSince2001(time.Now()) - ts) > int(passwordResetTimeoutDelta/(24*time.Hour)) {
		return errTokenExpired
	}
	return nil
}

func _makeTokenWithTimestamp(usr User, ts int) (string, error) {
	tsB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(strconv.Itoa(ts)))
	sig, err := _sign(_hashValue(usr, ts))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", tsB32, sig), nil
}

func _numDaysSince2001(t time.Time) int {
	ref := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return int(math.Ceil(t.Sub(ref).Hours() / 24))
}

func _sign(val []byte) (string, error) {
	secretKey := core.Conf.GetString("secretKey")
	key := sha256.Sum256(append(salt, secretKey...))
	h := hmac.New(sha256.New, key[:])
	_, err := h.Write(val)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}

func _hashValue(usr User, ts int) []byte {
	var val bytes.Buffer
	val.WriteString(strconv.Itoa(usr.ID))
	val.Write(usr.PasswordHash)
	if !usr.LastLogin.IsZero() {
		val.WriteString(usr.LastLogin.String())
	}
	val.WriteString(strconv.Itoa(ts))
	return val.Bytes()
}
