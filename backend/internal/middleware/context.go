package middleware

import (
	"context"
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
)

type contextKey string

const userContextKey contextKey = "authUser"

func SetAuthUser(ctx context.Context, user *model.AuthUser) context.Context {
	if user == nil {
		return ctx
	}
	return context.WithValue(ctx, userContextKey, *user)
}

func GetAuthUser(ctx context.Context) *model.AuthUser {
	user, ok := ctx.Value(userContextKey).(model.AuthUser)
	if !ok {
		return nil
	}
	return &user
}

func GetAuthUserOrPanic(ctx context.Context) *model.AuthUser {
	user := GetAuthUser(ctx)
	if user == nil {
		panic("auth user not found in context")
	}
	return user
}

func GetAuthUserOrError(ctx context.Context) (*model.AuthUser, error) {
	user := GetAuthUser(ctx)
	if user == nil {
		return nil, fmt.Errorf("auth user not found in context")
	}
	return user, nil
}
