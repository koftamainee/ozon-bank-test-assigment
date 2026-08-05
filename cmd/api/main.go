package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/auth"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/config"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/graphql"
	httpserver "github.com/koftamainee/ozon-bank-test-assigment/internal/http"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/http/middleware"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/log"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/notifier"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/pg"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/retry"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/shutdown"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/memory"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres"
)

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	initLogging(cfg.Env)
	validateProd(cfg)

	ctx := context.Background()

	set, err := newStore(ctx, cfg)
	if err != nil {
		log.Error("failed to init storage", "err", err)
		os.Exit(1)
	}

	secret := resolveJWTSecret(cfg)

	n := notifier.New()
	authSvc := service.NewAuth(set.users)
	forumSvc := service.NewForum(set.users, set.posts, set.comments, n)

	authMgr := auth.New(secret, cfg.JWTTTL, cfg.JWTCookieSecure)
	resolver := graphql.NewResolver(forumSvc, n)
	gqlSrv := newGraphQLServer(resolver, cfg.Env)

	mux, srv, health := httpserver.New(
		httpserver.Config{Addr: fmt.Sprintf(":%d", cfg.Port)},
		httpserver.Default(),
	)
	health.Check(cfg.StorageType, set.health)

	mux.Handle("POST /auth/login", auth.NewLoginHandler(authMgr, authSvc, log.With()))
	mux.Handle("POST /auth/logout", auth.NewLogoutHandler(authMgr))
	mux.Handle("/graphql", middleware.LimitBody(maxGraphQLBody)(authMgr.Middleware(gqlSrv)))

	sm := shutdown.NewManager()
	sm.Register("store", func(context.Context) error {
		set.close()
		return nil
	})
	sm.Register("graphql", resolver.Shutdown)
	sm.Register("http", srv.Shutdown)

	go func() {
		log.Info("api server started",
			"addr", srv.Addr,
			"storage", cfg.StorageType,
			"env", cfg.Env,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	sm.RunWithTimeout(ctx, 15*time.Second)
}

func initLogging(env string) {
	level := log.LevelInfo
	if env == envDev {
		level = log.LevelDebug
	}
	log.Init(log.Options{
		Level: level,
		JSON:  env == envProd,
		Color: env == envDev,
	})
}

const envProd = "prod"
const envDev = "dev"

const (
	maxGraphQLBody     = 1 << 20
	gqlComplexityLimit = 1000
	gqlParserToken     = 5000
)

func newGraphQLServer(resolver *graphql.Resolver, env string) *handler.Server {
	srv := handler.New(graphql.NewSchema(resolver))
	srv.AddTransport(transport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.POST{})
	if env != envProd {
		srv.AddTransport(transport.GET{})
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.FixedComplexityLimit(gqlComplexityLimit))
	srv.SetParserTokenLimit(gqlParserToken)
	return srv
}

type storeSet struct {
	users    store.UserStore
	posts    store.PostStore
	comments store.CommentStore
	health   func(ctx context.Context) error
	close    func()
}

func newStore(ctx context.Context, cfg Config) (*storeSet, error) {
	switch cfg.StorageType {
	case "memory":
		s := memory.New()
		return &storeSet{
			users:    s.Users(),
			posts:    s.Posts(),
			comments: s.Comments(),
			health:   func(context.Context) error { return nil },
			close:    func() {},
		}, nil

	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, errors.New("DATABASE_URL is required when STORAGE_TYPE=postgres")
		}
		pool, err := pg.New(ctx, pg.Config{DSN: expandEnvRefs(cfg.DatabaseURL)})
		if err != nil {
			return nil, fmt.Errorf("create pg pool: %w", err)
		}
		if err := retry.Do(ctx, func() error { return pool.Ping(ctx) }, 5, 500*time.Millisecond); err != nil {
			pool.Close()
			return nil, fmt.Errorf("connect to postgres: %w", err)
		}
		s := postgres.New(pool)
		return &storeSet{
			users:    s.Users(),
			posts:    s.Posts(),
			comments: s.Comments(),
			health:   pg.HealthCheck(pool),
			close:    pool.Close,
		}, nil

	default:
		return nil, fmt.Errorf("invalid STORAGE_TYPE %q (want memory or postgres)", cfg.StorageType)
	}
}

func resolveJWTSecret(cfg Config) []byte {
	if cfg.JWTSecret != "" {
		return []byte(cfg.JWTSecret)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Error("generate jwt secret", "err", err)
		os.Exit(1)
	}
	log.Warn("JWT_SECRET not set; generated ephemeral secret (sessions reset on restart)")
	return secret
}

var envRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandEnvRefs(s string) string {
	return envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		value, ok := os.LookupEnv(name)
		if !ok {
			return match
		}
		return value
	})
}

func validateProd(cfg Config) {
	if cfg.Env != envProd {
		return
	}
	if len(cfg.JWTSecret) < 32 {
		log.Error("JWT_SECRET must be set and at least 32 bytes long in prod")
		os.Exit(1)
	}
	if !cfg.JWTCookieSecure {
		log.Error("JWT_COOKIE_SECURE must be true in prod")
		os.Exit(1)
	}
}
