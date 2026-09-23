package main

import (
	"context"
	"errors"
	"os"

	"github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/modules/core/group"
	"github.com/vernal96/go-cms-kernel/modules/core/user"
	"github.com/vernal96/go-cms-kernel/security"
)

// Bootstrap uses the same domain validation and protected group assignment as
// ordinary user creation. Existing users are never overwritten.
func bootstrapAdmin(ctx context.Context, a *app.App) error {
	password := os.Getenv("CMS_ADMIN_PASSWORD")
	if password == "" {
		return errors.New("CMS_ADMIN_PASSWORD is required for bootstrap-admin")
	}
	administrators, err := a.Groups().GetByCode(ctx, security.System(), "admin")
	if err != nil {
		return err
	}
	_, err = a.Users().Create(ctx, security.System(), user.CreateInput{Login: env("CMS_ADMIN_LOGIN", "admin"), Email: env("CMS_ADMIN_EMAIL", "admin@example.test"), Password: password, Name: env("CMS_ADMIN_NAME", "Administrator"), GroupIDs: []group.ID{administrators.ID}})
	return err
}
