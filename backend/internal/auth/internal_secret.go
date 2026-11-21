package auth

import (
	"crypto/subtle"
	"errors"

	"connectrpc.com/connect"
)

const InternalSecretHeader = "X-Internal-Secret"

func RequireInternalSecret[T any](req *connect.Request[T], expectedSecret string) error {
	if expectedSecret == "" {
		return connect.NewError(connect.CodeInternal, errors.New("internal reminder secret is not configured"))
	}

	provided := req.Header().Get(InternalSecretHeader)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedSecret)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing or invalid internal secret"))
	}

	return nil
}
