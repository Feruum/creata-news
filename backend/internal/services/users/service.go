package users

import (
	"github.com/studyverse/ems-backend/gen/users/v1/usersv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
)

var _ usersv1connect.UsersServiceHandler = (*Service)(nil)

type Service struct {
	usersv1connect.UnimplementedUsersServiceHandler
	db                         *bun.DB
	perms                      *perms.Client
	authz                      *authz.Enforcer
	storage                    storage.Provider
	publisher                  *events.Publisher
	telegramBotURL             string
	telegramBotToken           string
	internalNotificationSecret string
}

func NewService(db *bun.DB, permsClient *perms.Client, authzEnforcer *authz.Enforcer, storageProvider storage.Provider, publisher *events.Publisher, telegramBotURL, telegramBotToken string, notificationSecrets ...string) *Service {
	secret := ""
	if len(notificationSecrets) > 0 {
		secret = notificationSecrets[0]
	}
	return &Service{
		db:                         db,
		perms:                      permsClient,
		authz:                      authzEnforcer,
		storage:                    storageProvider,
		publisher:                  publisher,
		telegramBotURL:             telegramBotURL,
		telegramBotToken:           telegramBotToken,
		internalNotificationSecret: secret,
	}
}
