package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const encodedPrefix = "myrax-argon2id"

type PasswordParams struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory:  64 * 1024,
		Time:    3,
		Threads: 2,
		KeyLen:  32,
	}
}

func HashPassword(password string) (string, error) {
	params := DefaultPasswordParams()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
	return fmt.Sprintf(
		"$%s$v=19$m=%d,t=%d,p=%d,l=%d$%s$%s",
		encodedPrefix,
		params.Memory,
		params.Time,
		params.Threads,
		params.KeyLen,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
	if len(actual) != len(expected) {
		return false, nil
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encoded string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != encodedPrefix || parts[2] != "v=19" {
		return PasswordParams{}, nil, nil, fmt.Errorf("unsupported password hash")
	}
	params, err := parseParams(parts[3])
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("invalid password salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("invalid password hash: %w", err)
	}
	if params.Memory < 19*1024 || params.Time < 1 || params.Threads < 1 || params.KeyLen < 16 {
		return PasswordParams{}, nil, nil, fmt.Errorf("weak password hash parameters")
	}
	return params, salt, hash, nil
}

func parseParams(value string) (PasswordParams, error) {
	params := PasswordParams{}
	for _, item := range strings.Split(value, ",") {
		keyValue := strings.SplitN(item, "=", 2)
		if len(keyValue) != 2 {
			return PasswordParams{}, fmt.Errorf("invalid password hash parameters")
		}
		number, err := strconv.ParseUint(keyValue[1], 10, 32)
		if err != nil {
			return PasswordParams{}, fmt.Errorf("invalid password hash parameter %s: %w", keyValue[0], err)
		}
		switch keyValue[0] {
		case "m":
			params.Memory = uint32(number)
		case "t":
			params.Time = uint32(number)
		case "p":
			params.Threads = uint8(number)
		case "l":
			params.KeyLen = uint32(number)
		default:
			return PasswordParams{}, fmt.Errorf("unknown password hash parameter: %s", keyValue[0])
		}
	}
	return params, nil
}
