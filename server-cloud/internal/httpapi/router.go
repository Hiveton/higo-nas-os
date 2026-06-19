package httpapi

import (
	"log/slog"
	"net/http"

	"higoos/server-cloud/internal/account"
	"higoos/server-cloud/internal/auth"
	"higoos/server-cloud/internal/bindings"
	"higoos/server-cloud/internal/devices"
	"higoos/server-cloud/internal/oauth"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/push"
	"higoos/server-cloud/internal/relay"
	"higoos/server-cloud/internal/store"
	"higoos/server-cloud/internal/verification"
)

// Dependencies are the injectable building blocks. Anything left nil is built
// from config with a sensible default — the same nil-fallback pattern server-go
// uses, so tests inject fakes and main injects nothing.
type Dependencies struct {
	Config  platform.Config
	Logger  *slog.Logger
	Store   store.Store
	Tokens  *auth.Manager
	Account *account.Service
	Devices *devices.Service
	Binding *bindings.Service
	Relay   *relay.Hub
	Push    *push.Service
	// Provisioner overrides how bindings provision NAS-local users. Defaults to
	// the dev stub; production passes relay.NewProvisioner(hub).
	Provisioner bindings.Provisioner
}

// NewRouter assembles the cloud control plane: build (or accept) every service,
// register routes on a ServeMux, and wrap the middleware chain.
func NewRouter(deps Dependencies) (http.Handler, error) {
	cfg := deps.Config.WithDefaults()
	logger := deps.Logger
	if logger == nil {
		logger = platform.NewLogger(cfg.Environment)
	}

	st := deps.Store
	if st == nil {
		// Only the memory store is wired here; the Postgres backend is selected
		// in main (it needs a live connection) and injected via deps.Store.
		st = store.NewMemory()
	}

	tokens := deps.Tokens
	if tokens == nil {
		var err error
		tokens, err = auth.NewManager(cfg.JWTSecret, cfg.DeviceTokenSeedHex,
			cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.DeviceTokenTTL)
		if err != nil {
			return nil, err
		}
	}

	hub := deps.Relay
	if hub == nil {
		hub = relay.NewHub()
	}

	acct := deps.Account
	if acct == nil {
		verifier := verification.NewService(cfg.VerificationTTL, verification.LogSender{Logger: logger}, cfg.VerificationDebug)
		apple := oauth.Apple{ClientID: cfg.AppleClientID, DevStub: cfg.IsDev()}
		wechat := oauth.WeChat{AppID: cfg.WeChatAppID, Secret: cfg.WeChatSecret, DevStub: cfg.IsDev()}
		acct = account.NewService(st, tokens, verifier, apple, wechat)
	}

	dev := deps.Devices
	if dev == nil {
		dev = devices.NewService(st)
	}

	provisioner := deps.Provisioner
	if provisioner == nil {
		if cfg.IsDev() {
			// Dev: synthetic local users, no NAS needed.
			provisioner = bindings.StubProvisioner{}
		} else {
			// Production: provision the NAS-local user over the relay tunnel so
			// the NAS stays the source of truth for local users.
			provisioner = relay.NewProvisioner(hub)
		}
	}

	bind := deps.Binding
	if bind == nil {
		bind = bindings.NewService(st, dev, tokens, provisioner, cfg.PairingTTL)
	}

	pushSvc := deps.Push
	if pushSvc == nil {
		pushSvc = push.NewService(st, logger)
		// Attach a real APNs sender when token-auth (.p8) is configured.
		if cfg.APNsKeyPath != "" && cfg.APNsKeyID != "" && cfg.APNsTeamID != "" {
			sender, err := push.NewAPNsSender(push.APNsConfig{
				KeyPath:  cfg.APNsKeyPath,
				KeyID:    cfg.APNsKeyID,
				TeamID:   cfg.APNsTeamID,
				BundleID: cfg.APNsBundleID,
			})
			if err != nil {
				logger.Warn("APNs disabled", slog.Any("error", err))
			} else {
				pushSvc = pushSvc.WithSender(sender)
				logger.Info("APNs sender enabled")
			}
		}
	}

	api := &API{
		config:  cfg,
		logger:  logger,
		tokens:  tokens,
		account: acct,
		devices: dev,
		binding: bind,
		relay:   hub,
		push:    pushSvc,
	}

	mux := http.NewServeMux()

	// Health -----------------------------------------------------------------
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /v1/health", api.health)

	// Auth (open) ------------------------------------------------------------
	mux.HandleFunc("POST /v1/auth/sms/start", api.smsStart)
	mux.HandleFunc("POST /v1/auth/sms/verify", api.smsVerify)
	mux.HandleFunc("POST /v1/auth/email/register", api.emailRegister)
	mux.HandleFunc("POST /v1/auth/email/login", api.emailLogin)
	mux.HandleFunc("POST /v1/auth/apple", api.appleLogin)
	mux.HandleFunc("POST /v1/auth/wechat", api.wechatLogin)
	mux.HandleFunc("POST /v1/auth/refresh", api.authRefresh)
	mux.HandleFunc("POST /v1/auth/logout", api.authLogout)

	// Account (cloud-authed) -------------------------------------------------
	mux.HandleFunc("GET /v1/account/me", api.accountMe)

	// Devices ----------------------------------------------------------------
	mux.HandleFunc("POST /v1/devices/register", api.deviceRegister)     // open (NAS self-register)
	mux.HandleFunc("POST /v1/devices/heartbeat", api.deviceHeartbeat)   // device-authed
	mux.HandleFunc("POST /v1/devices/{deviceId}/access-ticket", api.accessTicket)

	// Bindings ---------------------------------------------------------------
	mux.HandleFunc("POST /v1/bindings/pairing/issue", api.pairingIssue) // device-authed
	mux.HandleFunc("POST /v1/bindings/pairing/claim", api.pairingClaim)
	mux.HandleFunc("POST /v1/bindings/lan-confirm", api.lanConfirm)
	mux.HandleFunc("POST /v1/bindings/serial", api.bindSerial)
	mux.HandleFunc("GET /v1/bindings", api.bindingsList)
	mux.HandleFunc("DELETE /v1/bindings/{deviceId}", api.bindingUnbind)

	// Push -------------------------------------------------------------------
	mux.HandleFunc("POST /v1/push/register", api.pushRegister)

	// Relay ------------------------------------------------------------------
	mux.HandleFunc("/v1/agent/connect", api.agentConnect) // WebSocket (device-authed)
	mux.HandleFunc("/d/{deviceId}/{rest...}", api.relayForward)

	handler := chain(mux,
		platform.RequestIDMiddleware,
		recoverPanic(logger),
		secureHeaders,
		cors(cfg.PublicOrigin),
		cloudAuthGuard(api),
		accessLog(logger),
	)
	return handler, nil
}
