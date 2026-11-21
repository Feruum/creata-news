package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"connectrpc.com/validate"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	ory "github.com/ory/kratos-client-go"
	"github.com/redis/go-redis/v9"
	"github.com/riandyrn/otelchi"
	"github.com/samber/do/v2"
	slogmulti "github.com/samber/slog-multi"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/studyverse/ems-backend/gen/achievements/v1/achievementsv1connect"
	"github.com/studyverse/ems-backend/gen/activity/v1/activityv1connect"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/gen/registrationforms/v1/registrationformsv1connect"
	"github.com/studyverse/ems-backend/gen/rooms/v1/roomsv1connect"
	"github.com/studyverse/ems-backend/gen/search/v1/searchv1connect"
	sundayopenv1connect "github.com/studyverse/ems-backend/gen/sundayopen/v1/sundayopenv1connect"
	"github.com/studyverse/ems-backend/gen/users/v1/usersv1connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/buildinfo"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/db"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/events/handlers"
	"github.com/studyverse/ems-backend/internal/hermes"
	"github.com/studyverse/ems-backend/internal/httpapi"
	"github.com/studyverse/ems-backend/internal/miniappauth"

	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/services"
	achievementsservice "github.com/studyverse/ems-backend/internal/services/achievements"
	activityservice "github.com/studyverse/ems-backend/internal/services/activity"
	attendanceservice "github.com/studyverse/ems-backend/internal/services/eventattendance"
	eventsservice "github.com/studyverse/ems-backend/internal/services/events"
	orgsservice "github.com/studyverse/ems-backend/internal/services/organizations"
	orgmembersservice "github.com/studyverse/ems-backend/internal/services/orgmembers"
	roletemplatesservice "github.com/studyverse/ems-backend/internal/services/roletemplates"
	roomsservice "github.com/studyverse/ems-backend/internal/services/rooms"
	statisticsservice "github.com/studyverse/ems-backend/internal/services/statistics"
	sundayopenservice "github.com/studyverse/ems-backend/internal/services/sundayopen"
	usersservice "github.com/studyverse/ems-backend/internal/services/users"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/studyverse/ems-backend/internal/sundayaccount"
	"github.com/studyverse/ems-backend/internal/telemetry"
	"github.com/studyverse/ems-backend/internal/worker"
)

func main() {
	// Bootstrap logger — replaced after telemetry init with trace-correlated JSON logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	slog.Info("Starting EMS Backend",
		"host", cfg.Host,
		"port", cfg.Port,
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
	)

	// --- DI Container ---
	injector := do.New()

	// Pre-existing values
	do.ProvideValue(injector, cfg)

	// Telemetry
	do.Provide(injector, func(i do.Injector) (*telemetry.Providers, error) {
		cfg := do.MustInvoke[*config.Config](i)
		serviceVersion := cfg.OTELServiceVersion
		if serviceVersion == "" {
			serviceVersion = buildinfo.Version
		}
		return telemetry.New(telemetry.Config{
			ServiceName:    cfg.OTELServiceName,
			ServiceVersion: serviceVersion,
			Environment:    cfg.OTELEnvironment,
			OTLPEndpoint:   cfg.OTELEndpoint,
			Enabled:        cfg.OTELEnabled,
		})
	})

	// Database pool
	do.Provide(injector, func(i do.Injector) (*pgxpool.Pool, error) {
		cfg := do.MustInvoke[*config.Config](i)
		pool, err := db.NewPoolWithMaxConns(context.Background(), cfg.DatabaseURL, cfg.OTELEnabled, cfg.DatabaseMaxConns)
		if err != nil {
			return nil, err
		}
		slog.Info("Connected to database")
		return pool, nil
	})

	// Bun ORM
	do.Provide(injector, func(i do.Injector) (*bun.DB, error) {
		pool := do.MustInvoke[*pgxpool.Pool](i)
		cfg := do.MustInvoke[*config.Config](i)
		bunDB := db.NewBunDB(pool, cfg.OTELEnabled)
		slog.Info("Bun ORM initialized")
		return bunDB, nil
	})

	// Kratos client
	do.Provide(injector, func(i do.Injector) (*ory.APIClient, error) {
		cfg := do.MustInvoke[*config.Config](i)
		client := auth.NewKratosClient(cfg.KratosPublicURL)
		slog.Info("Kratos client initialized", "url", cfg.KratosPublicURL)
		return client, nil
	})

	// Redis storage for Telegram Mini App sign-in grants.
	do.Provide(injector, func(i do.Injector) (*redis.Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		opts, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("parse redis URL: %w", err)
		}
		return redis.NewClient(opts), nil
	})

	do.Provide(injector, func(i do.Injector) (*perms.Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		client, err := perms.NewClient(cfg.SpiceDBEndpoint, cfg.SpiceDBPresharedKey, cfg.SpiceDBInsecure, cfg.SpiceDBSkipVerifyCA, cfg.PlatformID)
		if err != nil {
			return nil, fmt.Errorf("SpiceDB client init failed (endpoint=%s): %w", cfg.SpiceDBEndpoint, err)
		}
		slog.Info("SpiceDB client initialized", "endpoint", cfg.SpiceDBEndpoint)
		return client, nil
	})

	// Meilisearch client
	do.Provide(injector, func(i do.Injector) (*search.Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		client, err := search.NewClient(cfg.MeilisearchURL, cfg.MeilisearchMasterKey)
		if err != nil {
			slog.Warn("Failed to initialize Meilisearch client - search will be unavailable",
				"url", cfg.MeilisearchURL,
				"error", err,
			)
			return nil, nil
		}
		slog.Info("Meilisearch client initialized", "url", cfg.MeilisearchURL)
		return client, nil
	})

	// Storage provider
	do.Provide(injector, func(i do.Injector) (storage.Provider, error) {
		cfg := do.MustInvoke[*config.Config](i)
		if cfg.StorageAccountName == "" || cfg.StorageAccountKey == "" {
			slog.Info("Storage not configured - image uploads disabled")
			return nil, nil
		}
		provider, err := storage.NewProvider(cfg)
		if err != nil {
			slog.Warn("Failed to initialize storage provider - image uploads will be unavailable",
				"provider", cfg.StorageProvider,
				"error", err,
			)
			return nil, nil
		}
		slog.Info("Storage provider initialized", "provider", cfg.StorageProvider, "container", cfg.StorageContainer)
		return provider, nil
	})

	// Authz enforcer
	do.Provide(injector, func(i do.Injector) (*authz.Enforcer, error) {
		bunDB := do.MustInvoke[*bun.DB](i)
		permsClient := do.MustInvoke[*perms.Client](i)
		return authz.NewEnforcer(bunDB, permsClient), nil
	})

	// Event bus (Watermill + Redis Streams)
	do.Provide(injector, func(i do.Injector) (*events.Bus, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := watermill.NewSlogLoggerWithLevelMapping(slog.Default(), nil)

		newRedis := func() (*redis.Client, error) {
			opts, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				return nil, fmt.Errorf("parse redis URL: %w", err)
			}
			return redis.NewClient(opts), nil
		}

		pubClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		searchClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		telegramClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		activityClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		historyClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		stickClient, err := newRedis()
		if err != nil {
			return nil, err
		}

		return events.NewBus(logger, pubClient, searchClient, telegramClient, activityClient, historyClient, stickClient)
	})

	// Event publisher (typed wrapper)
	do.Provide(injector, func(i do.Injector) (*events.Publisher, error) {
		bus := do.MustInvoke[*events.Bus](i)
		return events.NewDurablePublisher(do.MustInvoke[*bun.DB](i), bus.Publisher()), nil
	})

	// --- Services ---

	do.Provide(injector, func(i do.Injector) (*eventsservice.Service, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return eventsservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[storage.Provider](i),
			do.MustInvoke[*events.Publisher](i),
			cfg.MapAppURL,
			cfg.InternalReminderSecret,
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*orgsservice.Service, error) {
		return orgsservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[storage.Provider](i),
			do.MustInvoke[*events.Publisher](i),
			do.MustInvoke[*orgmembersservice.Service](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*services.OrganizationTypesService, error) {
		return services.NewOrganizationTypesService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*roletemplatesservice.Service, error) {
		return roletemplatesservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*orgmembersservice.Service, error) {
		return orgmembersservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[*events.Publisher](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*services.CategoriesService, error) {
		return services.NewCategoriesService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*services.EventRegistrationsService, error) {
		return services.NewEventRegistrationsService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*events.Publisher](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[*config.Config](i).InternalReminderSecret,
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*attendanceservice.Service, error) {
		return attendanceservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[*config.Config](i).MapAppURL,
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*statisticsservice.Service, error) {
		return statisticsservice.NewService(do.MustInvoke[*bun.DB](i), do.MustInvoke[*authz.Enforcer](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*usersservice.Service, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return usersservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[storage.Provider](i),
			do.MustInvoke[*events.Publisher](i),
			cfg.TelegramBotURL,
			cfg.TelegramBotToken,
			cfg.InternalReminderSecret,
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*services.SearchService, error) {
		return services.NewSearchService(
			do.MustInvoke[*search.Client](i),
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*roomsservice.Service, error) {
		return roomsservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*perms.Client](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[*events.Publisher](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*activityservice.Service, error) {
		return activityservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*sundayopenservice.Service, error) {
		return sundayopenservice.NewService(
			do.MustInvoke[*bun.DB](i),
			do.MustInvoke[*authz.Enforcer](i),
			do.MustInvoke[*config.Config](i).ExternalRegistrationCancelSecret,
		), nil
	})

	// --- Eagerly initialize telemetry for early shutdown registration ---
	providers, err := do.Invoke[*telemetry.Providers](injector)
	if err != nil {
		slog.Warn("Failed to initialize telemetry", "error", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if providers != nil {
			if err := providers.Shutdown(ctx); err != nil {
				slog.Error("Telemetry shutdown error", "error", err)
			}
		}
	}()

	// --- Set up production slog: JSON + trace correlation + optional OTEL Logs pipeline ---
	{
		var level slog.Level
		switch strings.ToLower(cfg.LogLevel) {
		case "debug":
			level = slog.LevelDebug
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		traceHandler := telemetry.NewTraceHandler(jsonHandler)

		var handler slog.Handler
		if providers != nil && providers.LoggerProvider != nil {
			otelHandler := otelslog.NewHandler("ems-backend", otelslog.WithLoggerProvider(providers.LoggerProvider))
			handler = slogmulti.Fanout(traceHandler, otelHandler)
		} else {
			handler = traceHandler
		}

		slog.SetDefault(slog.New(handler))
	}

	// --- Start runtime metrics (goroutine count, memory, GC) ---
	if err := otelruntime.Start(); err != nil {
		slog.Warn("Failed to start runtime metrics", "error", err)
	}

	// --- Eagerly initialize pool for defer Close ---
	pool := do.MustInvoke[*pgxpool.Pool](injector)
	defer pool.Close()

	// Close Bun DB wrapper on shutdown
	bunDB := do.MustInvoke[*bun.DB](injector)
	defer db.CloseBunDB(bunDB)

	// Start finalization worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	finalizationWorker := worker.NewFinalizationWorker(bunDB)
	go finalizationWorker.Start(workerCtx)
	go achievementsservice.NewWorker(bunDB).Start(workerCtx)
	privateStore, privateStorageErr := storage.NewPrivateProvider(cfg)
	if privateStorageErr != nil {
		slog.Warn("Private registration file storage unavailable", "error", privateStorageErr)
		privateStore = nil
	}
	registrationFiles := registrationfiles.New(bunDB, privateStore)
	go registrationFiles.Start(workerCtx)
	registrationForms := registrationforms.NewService(bunDB, do.MustInvoke[*authz.Enforcer](injector), registrationFiles).
		WithPublisher(do.MustInvoke[*events.Publisher](injector))
	sundayAccountHandler := sundayaccount.New(bunDB, sundayaccount.Config{
		Origin: cfg.SundayOpenOrigins, Secret: cfg.SundayOpenAuthSecret,
		MapAppURL: cfg.MapAppURL,
		SMTPURL:   cfg.SundayOpenSMTPURL, SMTPFrom: cfg.SundayOpenSMTPFrom,
		BotToken: cfg.SundayOpenTelegramBotToken, BotUsername: cfg.SundayOpenTelegramBotUsername,
		BotConfigEncrypted: cfg.SundayOpenTelegramBotConfigEncrypted,
		WebhookSecret:      cfg.SundayOpenTelegramWebhookSecret,
		TrustedProxyCIDRs:  cfg.SundayOpenTrustedProxyCIDRs,
	}, nil).WithOrganizer(do.MustInvoke[*attendanceservice.Service](injector).WithForms(registrationForms)).WithForms(registrationForms)
	go sundayAccountHandler.Start(workerCtx)

	// --- Domain Event Bus ---
	bus := do.MustInvoke[*events.Bus](injector)
	if err := bus.InitStick(context.Background()); err != nil {
		slog.Error("Failed to initialize event stream stickiness", "error", err)
	}

	meilisearchClient := do.MustInvoke[*search.Client](injector)
	permsClient := do.MustInvoke[*perms.Client](injector)

	var searchClient handlers.SearchClient
	if meilisearchClient != nil {
		searchClient = meilisearchClient
	}

	handlers.RegisterHandlers(bus.Router(), handlers.HandlerDeps{
		DB:                         bunDB,
		SearchClient:               searchClient,
		TelegramBotURL:             cfg.TelegramBotURL,
		InternalNotificationSecret: cfg.InternalReminderSecret,
		MapAppURL:                  cfg.MapAppURL,
		Perms:                      permsClient,
	}, bus.SearchSub(), bus.TelegramSub(), bus.ActivitySub(), bus.HistorySub())

	busErrCh := make(chan error, 1)
	startBus(bus, busErrCh)
	outbox := events.NewOutboxDispatcher(bunDB, bus.Publisher(), do.MustInvoke[*authz.Enforcer](injector).SyncOrganizationMember, do.MustInvoke[*authz.Enforcer](injector).SyncPlatformUserRole)
	go outbox.Start(workerCtx)

	// --- HTTP Router ---
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		campusRealIP := chimiddleware.RealIP(next)
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/sundayopen/") {
				next.ServeHTTP(w, request)
				return
			}
			campusRealIP.ServeHTTP(w, request)
		})
	})
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	if cfg.OTELEnabled {
		r.Use(otelchi.Middleware(cfg.OTELServiceName, otelchi.WithChiRoutes(r)))
	}

	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTrustRemote(),
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		slog.Error("Failed to create otelconnect interceptor", "error", err)
		os.Exit(1)
	}
	interceptors := connect.WithInterceptors(validate.NewInterceptor(), loggingInterceptor(), otelInterceptor, achievementsservice.BadgeInterceptor(bunDB))

	// Events services
	formsPath, formsHandler := registrationformsv1connect.NewRegistrationFormsServiceHandler(registrationForms, interceptors)
	formsHandler = uncachedFormsHandler(formsHandler)
	r.Mount(formsPath, formsHandler)
	path, handler := eventsv1connect.NewEventsServiceHandler(do.MustInvoke[*eventsservice.Service](injector), interceptors, connect.WithReadMaxBytes(storage.ImageRPCReadMaxBytes))
	r.Mount(path, handler)
	path, handler = achievementsv1connect.NewAchievementsServiceHandler(achievementsservice.NewService(bunDB, do.MustInvoke[*authz.Enforcer](injector), do.MustInvoke[storage.Provider](injector)), interceptors, connect.WithReadMaxBytes(storage.ImageRPCReadMaxBytes))
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewOrganizationsServiceHandler(do.MustInvoke[*orgsservice.Service](injector), interceptors, connect.WithReadMaxBytes(storage.ImageRPCReadMaxBytes))
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewOrganizationTypesServiceHandler(do.MustInvoke[*services.OrganizationTypesService](injector), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewRoleTemplatesServiceHandler(do.MustInvoke[*roletemplatesservice.Service](injector), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewOrganizationMembersServiceHandler(do.MustInvoke[*orgmembersservice.Service](injector), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewCategoriesServiceHandler(do.MustInvoke[*services.CategoriesService](injector), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewEventRegistrationsServiceHandler(do.MustInvoke[*services.EventRegistrationsService](injector).WithForms(registrationForms), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewEventAttendanceServiceHandler(do.MustInvoke[*attendanceservice.Service](injector).WithForms(registrationForms), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewStatisticsServiceHandler(do.MustInvoke[*statisticsservice.Service](injector), interceptors)
	r.Mount(path, handler)
	path, handler = eventsv1connect.NewOrganizationSubscriptionsServiceHandler(do.MustInvoke[*eventsservice.Service](injector), interceptors)
	r.Mount(path, handler)

	// Users service
	path, handler = usersv1connect.NewUsersServiceHandler(do.MustInvoke[*usersservice.Service](injector), interceptors)
	r.Mount(path, handler)

	// Search service
	path, handler = searchv1connect.NewSearchServiceHandler(do.MustInvoke[*services.SearchService](injector), interceptors)
	r.Mount(path, handler)

	// Rooms service
	path, handler = roomsv1connect.NewRoomsServiceHandler(do.MustInvoke[*roomsservice.Service](injector), interceptors)
	r.Mount(path, handler)

	// Activity service
	path, handler = activityv1connect.NewActivityServiceHandler(do.MustInvoke[*activityservice.Service](injector), interceptors)
	r.Mount(path, handler)

	// Sunday Open service
	path, handler = sundayopenv1connect.NewSundayOpenServiceHandler(do.MustInvoke[*sundayopenservice.Service](injector).WithForms(registrationForms), interceptors)
	r.Mount(path, handler)
	sundayRPC := http.NewServeMux()
	sundayRPC.Handle(path, handler)
	sundayRPC.Handle(formsPath, formsHandler)
	r.Handle("/api/sundayopen/rpc/*", sundayPublicRPCHandler(sundayRPC))
	r.Handle("/api/sundayopen/account/*", sundayAccountHandler)
	r.Handle("/check-in/organizer-access", auth.NewCookieMiddleware(do.MustInvoke[*ory.APIClient](injector))(
		do.MustInvoke[*attendanceservice.Service](injector).OrganizerAccessHandler(cfg.SundayOpenAuthSecret, cfg.SundayOpenOrigins),
	))
	r.Handle("/api/sundayopen/telegram/webhook", sundayAccountHandler)

	httpapi.RegisterOperationalRoutes(
		r,
		buildinfo.Current("ems-backend", cfg.OTELEnvironment),
		httpapi.Check{Name: "postgres", Run: pool.Ping},
		httpapi.Check{Name: "redis", Run: bus.Ready},
	)
	r.Post("/internal/academic-notifications", httpapi.NewAcademicNotificationsHandler(bunDB, cfg))

	miniAppRedisClient := do.MustInvoke[*redis.Client](injector)
	defer miniAppRedisClient.Close()
	miniAppHandler, miniAppErr := miniappauth.NewHandler(
		miniappauth.Config{
			Issuer:        strings.TrimRight(cfg.MapAppURL, "/") + "/api/auth/telegram/oidc",
			RedirectURI:   cfg.TelegramMiniAppRedirectURI,
			ClientSecret:  cfg.TelegramMiniAppClientSecret,
			PrivateKeyB64: cfg.TelegramMiniAppPrivateKeyB64,
			BotToken:      cfg.TelegramBotToken,
		},
		miniappauth.NewRedisStore(miniAppRedisClient),
		miniappauth.NewResolver(bunDB, cfg.KratosAdminURL, cfg.KratosPublicURL, cfg.TelegramMiniAppClientSecret,
			miniappauth.WithSessionCookieName(cfg.KratosSessionCookieName)),
	)
	if miniAppErr != nil {
		slog.Error("Telegram Mini App sign-in configuration is invalid", "error", miniAppErr)
		miniAppHandler, _ = miniappauth.NewHandler(miniappauth.Config{}, nil, nil)
	}
	r.Mount("/telegram-miniapp", miniAppHandler)

	// Admin endpoint to promote users to platform admin/staff
	// Protected by ADMIN_SECRET environment variable
	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret != "" && permsClient != nil {
		r.Post("/admin/promote", adminPromotionHandler(bunDB, do.MustInvoke[*authz.Enforcer](injector), adminSecret))
		slog.Info("Admin promotion endpoint enabled at /admin/promote")
	}

	// Cookie-only operator endpoints are selected before global auth and response instrumentation.
	kratosClient := do.MustInvoke[*ory.APIClient](injector)
	authMiddleware := auth.NewMiddleware(kratosClient)
	campusHandler := authMiddleware(r)
	isolatedHandler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/sundayopen/") {
			r.ServeHTTP(w, request)
			return
		}
		campusHandler.ServeHTTP(w, request)
	})
	eventAuthor := hermes.NewEventAuthor(bunDB, do.MustInvoke[*authz.Enforcer](injector),
		do.MustInvoke[*eventsservice.Service](injector), do.MustInvoke[*roomsservice.Service](injector), cfg.MapAppURL).
		WithPaidForm(registrationForms.PrepareHermesPaidForm)
	operatorHandler := httpapi.WithHermesRoutes(httpapi.WithSigNozRoutes(
		isolatedHandler, cfg, do.MustInvoke[*authz.Enforcer](injector), kratosClient,
	), cfg, kratosClient, hermes.NewTelegramStore(bunDB), eventAuthor)
	finalHandler := corsMiddleware(append(cfg.CORSOrigins, strings.Split(cfg.SundayOpenOrigins, ",")...), operatorHandler)

	// Create server with h2c (HTTP/2 cleartext) support for Connect-RPC
	server := &http.Server{
		Addr:         cfg.Host + ":" + cfg.Port,
		Handler:      h2c.NewHandler(finalHandler, &http2.Server{}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Enable HTTP/2 without TLS
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	server.Protocols = p

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	select {
	case <-shutdown:
		slog.Info("Shutting down server (signal)...")
	case err := <-busErrCh:
		slog.Error("event bus exited unexpectedly, shutting down", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	if err := bus.Close(); err != nil {
		slog.Error("close event bus", "error", err)
	}

	slog.Info("Server stopped")
}

func uncachedFormsHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("CDN-Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		handler.ServeHTTP(w, r)
	})
}

func sundayPublicRPCHandler(handler http.Handler) http.Handler {
	stripped := http.StripPrefix("/api/sundayopen/rpc", handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		const publishedForm = "/api/sundayopen/rpc/registrationforms.v1.RegistrationFormsService/GetPublishedForm"
		if r.URL.Path == publishedForm {
			stripped.ServeHTTP(w, r)
			return
		}
		const servicePrefix = "/api/sundayopen/rpc/sundayopen.v1.SundayOpenService/"
		if !strings.HasPrefix(r.URL.Path, servicePrefix) {
			http.NotFound(w, r)
			return
		}
		switch strings.TrimPrefix(r.URL.Path, servicePrefix) {
		case "GetUpcomingEvents", "ListPublicEvents", "GetEventBySlug", "RegisterExternally", "CancelExternalRegistration":
			stripped.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func corsMiddleware(origins []string, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool)
	for _, origin := range origins {
		allowedOrigins[origin] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/internal/hermes/events/") {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if r.URL.Path == "/admin/signoz/access" || r.URL.Path == "/admin/signoz/info" || httpapi.IsHermesRoute(r.URL.Path) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Referrer-Policy", "no-referrer")
		}

		// Check if origin is allowed
		if allowedOrigins[origin] || len(origins) == 0 || (len(origins) == 1 && origins[0] == "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Content-Language, Authorization, Connect-Protocol-Version, Connect-Timeout-Ms, X-Grpc-Timeout, X-User-Agent, X-Session-Token, X-CSRF-Token, X-Check-In-Ticket, X-Organizer-Ticket, X-Studyverse-SigNoz, X-Studyverse-Hermes, Traceparent, Tracestate")
		w.Header().Set("Access-Control-Expose-Headers", "Connect-Content-Encoding, Connect-Timeout-Ms, Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			resp, err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				slog.ErrorContext(ctx, "RPC failed", "procedure", procedure,
					"duration", duration,
					"error", err)
			} else {
				slog.InfoContext(ctx, "RPC completed", "procedure", procedure,
					"duration", duration)
			}

			return resp, err
		}
	}
}

// startBus runs bus.Run in a goroutine and forwards non-context.Canceled
// errors to errCh so the main goroutine can trigger a graceful shutdown.
func startBus(bus *events.Bus, errCh chan<- error) {
	startBusRunner(errCh, func() error {
		return bus.Run(context.Background())
	})
}

func startBusRunner(errCh chan<- error, run func() error) {
	go func() {
		if err := run(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("event bus exited", "error", err)
			errCh <- err
			return
		}
		errCh <- nil
	}()
}
