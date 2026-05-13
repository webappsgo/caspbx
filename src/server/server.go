package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/casapps/caspbx/src/config"
	"github.com/casapps/caspbx/src/server/handler"
	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
	"github.com/casapps/caspbx/src/server/store"
)

type Bootstrap struct {
	Routes RouteCatalog
}

type App struct {
	Bootstrap Bootstrap
	mux       *http.ServeMux
	auth      service.AuthService
}

func NewBootstrap(apiVersion string, adminPath string) (Bootstrap, error) {
	routes, routeError := NewRouteCatalog(apiVersion, adminPath)
	if routeError != nil {
		return Bootstrap{}, routeError
	}
	return Bootstrap{Routes: routes}, nil
}

func NewApp(apiVersion string, adminPath string, projectName string, version string, commitID string, officialSite string) (App, error) {
	defaultConfig := config.DefaultConfig()
	return NewAppWithStore(apiVersion, adminPath, projectName, version, commitID, officialSite, defaultConfig.Server, store.NewMemoryStore())
}

func NewAppWithStore(apiVersion string, adminPath string, projectName string, version string, commitID string, officialSite string, serverConfig config.ServerConfig, runtimeStore store.RuntimeStore) (App, error) {
	bootstrap, bootstrapError := NewBootstrap(apiVersion, adminPath)
	if bootstrapError != nil {
		return App{}, bootstrapError
	}

	authService := service.NewAuthService(runtimeStore, service.SessionConfig{
		AdminTTL:         timeDurationHours(serverConfig.Session.Admin.MaxAgeHours),
		UserTTL:          timeDurationHours(serverConfig.Session.User.MaxAgeHours),
		ExtendOnActivity: serverConfig.Session.ExtendOnActivity,
	})
	if _, saveError := runtimeStore.SaveAsteriskState(context.Background(), asteriskState(serverConfig)); saveError != nil {
		return App{}, saveError
	}
	asteriskService := service.NewAsteriskService(runtimeStore)
	pbxService := service.NewPBXService(runtimeStore, runtimeStore)
	userCommunicationsService := service.NewUserCommunicationsService(runtimeStore, runtimeStore, runtimeStore, runtimeStore)
	operatorService := service.NewOperatorService(runtimeStore, runtimeStore, runtimeStore)
	domainService := service.NewDomainService(runtimeStore, customDomainConstraints(serverConfig), platformHosts(officialSite), nil)

	healthResponse := handler.HealthResponse{
		Status:                "ok",
		Project:               projectName,
		Version:               version,
		CommitID:              commitID,
		APIBasePath:           bootstrap.Routes.APIBasePath,
		AdminPath:             bootstrap.Routes.AdminBasePath,
		AsteriskAdminPath:     bootstrap.Routes.AsteriskAdminBasePath,
		OfficialSite:          officialSite,
		RuntimeImplementation: "asterisk-control-foundation",
	}

	userCookie := handler.SessionCookieConfig{
		Name:     serverConfig.Session.User.CookieName,
		Path:     "/",
		HTTPOnly: serverConfig.Session.HTTPOnly,
		Secure:   serverConfig.Session.Secure,
		SameSite: sameSiteMode(serverConfig.Session.SameSite),
	}
	adminCookie := handler.SessionCookieConfig{
		Name:     serverConfig.Session.Admin.CookieName,
		Path:     bootstrap.Routes.AdminBasePath,
		HTTPOnly: serverConfig.Session.HTTPOnly,
		Secure:   serverConfig.Session.Secure,
		SameSite: sameSiteMode(serverConfig.Session.SameSite),
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler.NewRootHandler(projectName, officialSite, bootstrap.Routes.AdminBasePath, bootstrap.Routes.APIBasePath))
	mux.Handle("/health", handler.NewHealthHandler(healthResponse))
	mux.Handle("/healthz", handler.NewHealthHandler(healthResponse))
	mux.Handle("/version", handler.NewVersionHandler(projectName, version, commitID))
	registerSurface(mux, bootstrap.Routes.AuthBasePath, handler.NewAuthHandler(bootstrap.Routes.AuthBasePath, authService, userCookie, model.DefaultRegistrationMode()))
	registerSurface(mux, bootstrap.Routes.AuthAPIBasePath, handler.NewAPIAuthHandler(bootstrap.Routes.AuthAPIBasePath, authService, model.DefaultRegistrationMode()))
	registerSurface(mux, bootstrap.Routes.UsersBasePath, handler.NewUserHandler(bootstrap.Routes.UsersBasePath, authService, domainService, userCookie, userCommunicationsService))
	registerSurface(mux, bootstrap.Routes.UsersAPIBasePath, handler.NewAPIUserHandler(bootstrap.Routes.UsersAPIBasePath, authService, domainService, userCommunicationsService))
	registerSurface(mux, bootstrap.Routes.OrgsBasePath, handler.NewOrgHandler(bootstrap.Routes.OrgsBasePath, authService, domainService, runtimeStore, userCookie))
	registerSurface(mux, bootstrap.Routes.OrgsAPIBasePath, handler.NewAPIOrgHandler(bootstrap.Routes.OrgsAPIBasePath, authService, domainService, runtimeStore))
	registerSurface(mux, bootstrap.Routes.AdminBasePath, handler.NewAdminHandler(bootstrap.Routes.AdminBasePath, authService, domainService, asteriskService, pbxService, adminCookie, operatorService))
	registerSurface(mux, bootstrap.Routes.AdminAPIBasePath, handler.NewAPIAdminHandler(bootstrap.Routes.AdminAPIBasePath, authService, domainService, asteriskService, pbxService, operatorService))
	registerSurface(mux, bootstrap.Routes.AsteriskAdminBasePath, handler.NewAdminHandler(bootstrap.Routes.AsteriskAdminBasePath, authService, domainService, asteriskService, pbxService, adminCookie, operatorService))
	registerSurface(mux, bootstrap.Routes.AsteriskAdminAPIPath, handler.NewAPIAdminHandler(bootstrap.Routes.AsteriskAdminAPIPath, authService, domainService, asteriskService, pbxService, operatorService))

	return App{Bootstrap: bootstrap, mux: mux, auth: authService}, nil
}

func (bootstrap Bootstrap) Summary() string {
	return fmt.Sprintf(
		"API base path: %s\nAdmin path: %s\nAsterisk admin path: %s",
		bootstrap.Routes.APIBasePath,
		bootstrap.Routes.AdminBasePath,
		bootstrap.Routes.AsteriskAdminBasePath,
	)
}

func (app App) Handler() http.Handler {
	return app.mux
}

func (app App) Summary() string {
	return fmt.Sprintf(
		"%s\nHealth path: /health\nVersion path: /version",
		app.Bootstrap.Summary(),
	)
}

func registerSurface(mux *http.ServeMux, routePrefix string, surfaceHandler http.Handler) {
	mux.Handle(routePrefix, surfaceHandler)
	mux.Handle(routePrefix+"/", surfaceHandler)
}

func sameSiteMode(value string) http.SameSite {
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func timeDurationHours(hours int) time.Duration {
	return time.Duration(hours) * time.Hour
}

func customDomainConstraints(serverConfig config.ServerConfig) model.DomainConstraints {
	return model.DomainConstraints{
		Enabled:           serverConfig.Features.CustomDomains.Enabled,
		MaxDomainsPerUser: serverConfig.Features.CustomDomains.MaxDomainsPerUser,
		MaxDomainsPerOrg:  serverConfig.Features.CustomDomains.MaxDomainsPerOrg,
		RequireSSL:        serverConfig.Features.CustomDomains.RequireSSL,
		AllowApex:         serverConfig.Features.CustomDomains.AllowApex,
		AllowSubdomain:    serverConfig.Features.CustomDomains.AllowSubdomain,
		AllowWildcard:     serverConfig.Features.CustomDomains.AllowWildcard,
		VerificationTTL:   serverConfig.Features.CustomDomains.VerificationTTL,
		SSLRenewalDays:    serverConfig.Features.CustomDomains.SSLRenewalDays,
		Reserved:          serverConfig.Features.CustomDomains.Reserved,
		BlockedPatterns:   serverConfig.Features.CustomDomains.BlockedPatterns,
	}
}

func platformHosts(officialSite string) []string {
	parsedURL, parseError := url.Parse(officialSite)
	if parseError != nil || parsedURL.Host == "" {
		return nil
	}
	return []string{parsedURL.Hostname()}
}

func asteriskState(serverConfig config.ServerConfig) model.AsteriskState {
	defaultState := model.DefaultAsteriskState()
	capabilities := []model.AsteriskCapability{
		{Key: "queues", Label: "Queues", Family: "queue", Available: serverConfig.Asterisk.Capabilities.Queues},
		{Key: "conferences", Label: "Conferences", Family: "conference", Available: serverConfig.Asterisk.Capabilities.Conferences},
		{Key: "recordings", Label: "Recordings", Family: "media", Available: serverConfig.Asterisk.Capabilities.Recordings},
		{Key: "voicemail", Label: "Voicemail", Family: "media", Available: serverConfig.Asterisk.Capabilities.Voicemail},
		{Key: "prompts", Label: "Prompts", Family: "media", Available: serverConfig.Asterisk.Capabilities.Prompts},
		{Key: "music_on_hold", Label: "Music on Hold", Family: "media", Available: serverConfig.Asterisk.Capabilities.MusicOnHold},
		{Key: "fax", Label: "Fax", Family: "fax", Available: serverConfig.Asterisk.Capabilities.Fax},
		{Key: "xmpp", Label: "XMPP", Family: "messaging", Available: serverConfig.Asterisk.Capabilities.XMPP},
		{Key: "presence", Label: "Presence", Family: "messaging", Available: serverConfig.Asterisk.Capabilities.Presence},
		{Key: "dahdi", Label: "DAHDI", Family: "hardware", Available: serverConfig.Asterisk.Capabilities.DAHDI},
		{Key: "browser_calling", Label: "Browser Calling", Family: "webphone", Available: serverConfig.Asterisk.Capabilities.BrowserCalling},
		{Key: "tls", Label: "TLS", Family: "security", Available: serverConfig.Asterisk.Capabilities.TLS},
		{Key: "mail_delivery", Label: "Mail Delivery", Family: "messaging", Available: serverConfig.Asterisk.Capabilities.MailDelivery},
		{Key: "metrics", Label: "Metrics", Family: "operations", Available: serverConfig.Asterisk.Capabilities.Metrics},
		{Key: "scheduler", Label: "Scheduler", Family: "operations", Available: serverConfig.Asterisk.Capabilities.Scheduler},
		{Key: "domain_automation", Label: "Domain Automation", Family: "operations", Available: serverConfig.Asterisk.Capabilities.DomainAutomation},
	}
	subsystems := []model.AsteriskManagedSubsystem{}
	if serverConfig.Asterisk.Subsystems.FaxBackend != "" {
		subsystems = append(subsystems, model.AsteriskManagedSubsystem{Key: "fax_backend", Label: "Fax backend", Provider: serverConfig.Asterisk.Subsystems.FaxBackend, Healthy: true})
	}
	if serverConfig.Asterisk.Subsystems.MessagingBackend != "" {
		subsystems = append(subsystems, model.AsteriskManagedSubsystem{Key: "messaging_backend", Label: "Messaging backend", Provider: serverConfig.Asterisk.Subsystems.MessagingBackend, Healthy: true})
	}
	if serverConfig.Asterisk.Subsystems.TTSEngine != "" {
		subsystems = append(subsystems, model.AsteriskManagedSubsystem{Key: "tts_engine", Label: "TTS engine", Provider: serverConfig.Asterisk.Subsystems.TTSEngine, Healthy: true})
	}
	if serverConfig.Asterisk.Subsystems.MusicOnHoldSources != "" {
		subsystems = append(subsystems, model.AsteriskManagedSubsystem{Key: "music_on_hold", Label: "Music on hold", Provider: serverConfig.Asterisk.Subsystems.MusicOnHoldSources, Healthy: true})
	}
	state := defaultState
	state.MinimumSupportedVersion = firstNonEmpty(serverConfig.Asterisk.MinimumSupportedVersion, defaultState.MinimumSupportedVersion)
	state.DetectedVersion = serverConfig.Asterisk.DetectedVersion
	state.DetectionStatus = firstNonEmpty(serverConfig.Asterisk.DetectionStatus, defaultState.DetectionStatus)
	state.HealthStatus = model.AsteriskHealthStatus(firstNonEmpty(serverConfig.Asterisk.HealthStatus, string(defaultState.HealthStatus)))
	state.ChannelDrivers = append([]string{}, serverConfig.Asterisk.ChannelDrivers...)
	state.EndpointStacks = append([]string{}, serverConfig.Asterisk.EndpointStacks...)
	state.Codecs = append([]string{}, serverConfig.Asterisk.Codecs...)
	state.Capabilities = capabilities
	state.Subsystems = subsystems
	state.UpdatedAt = time.Now().UTC()
	return state
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
