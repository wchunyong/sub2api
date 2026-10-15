package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrDeviceRegistrationLimit = infraerrors.TooManyRequests(
	"DEVICE_REGISTRATION_LIMIT",
	"设备注册抵达上限，请稍后重试或联系管理员",
)

type registrationOriginKey struct{}

type RegistrationOrigin struct {
	IPHash     string
	DeviceHash string
}

// WithRegistrationOrigin carries the server-resolved address and a browser
// cookie through every public signup path. Raw values are never persisted.
func WithRegistrationOrigin(ctx context.Context, ip, deviceID string) context.Context {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return context.WithValue(ctx, registrationOriginKey{}, RegistrationOrigin{})
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return context.WithValue(ctx, registrationOriginKey{}, RegistrationOrigin{})
	}
	return context.WithValue(ctx, registrationOriginKey{}, RegistrationOrigin{
		IPHash:     registrationHash(parsed.String()),
		DeviceHash: registrationHash(deviceID),
	})
}

func RegistrationOriginFromContext(ctx context.Context) (RegistrationOrigin, bool) {
	origin, ok := ctx.Value(registrationOriginKey{}).(RegistrationOrigin)
	return origin, ok
}

func registrationHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
