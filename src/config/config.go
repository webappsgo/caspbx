package config

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"slices"
	"time"
)

var randomHighPortReader io.Reader = rand.Reader

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	Mode           AppMode
	DebugEnabled   bool
	Address        string
	Port           int
	BaseURL        string
	AdminPath      string
	Limits         RequestLimitsConfig
	Compression    CompressionConfig
	TrustedProxies TrustedProxiesConfig
	Session        SessionConfig
	RateLimit      RateLimitConfig
	I18N           I18NConfig
	Contact        ContactConfig
	Tracking       TrackingConfig
	Maintenance    MaintenanceConfig
	Asterisk       AsteriskConfig
	Features       FeatureConfig
}

type RequestLimitsConfig struct {
	MaxBodySizeBytes int64
	ReadTimeoutSec   int
	WriteTimeoutSec  int
	IdleTimeoutSec   int
}

type CompressionConfig struct {
	Enabled bool
	Level   int
	Types   []string
}

type TrustedProxiesConfig struct {
	Additional []string
}

type SessionConfig struct {
	Admin            SessionScopeConfig
	User             SessionScopeConfig
	ExtendOnActivity bool
	Secure           string
	HTTPOnly         bool
	SameSite         string
}

type SessionScopeConfig struct {
	CookieName       string
	MaxAgeHours      int
	IdleTimeoutHours int
}

type RateLimitConfig struct {
	Enabled   bool
	Requests  int
	WindowSec int
}

type I18NConfig struct {
	DefaultLanguage string
	Supported       []string
}

type ContactConfig struct {
	Admin    ContactRoleConfig
	Security ContactRoleConfig
	General  ContactRoleConfig
}

type ContactRoleConfig struct {
	Email    string
	Webhooks ContactWebhookConfig
}

type ContactWebhookConfig struct {
	Telegram string
	Discord  string
	Slack    string
	Generic  string
}

type TrackingConfig struct {
	Type string
	ID   string
	URL  string
}

type MaintenanceConfig struct {
	SelfHealing MaintenanceSelfHealingConfig
	Cleanup     MaintenanceCleanupConfig
	Notify      MaintenanceNotifyConfig
}

type MaintenanceSelfHealingConfig struct {
	Enabled          bool
	RetryIntervalSec int
	MaxAttempts      int
}

type MaintenanceCleanupConfig struct {
	DiskThresholdPercent int
	LogRetentionDays     int
	BackupKeepCount      int
}

type MaintenanceNotifyConfig struct {
	OnEnter bool
	OnExit  bool
}

type AsteriskConfig struct {
	MinimumSupportedVersion string
	DetectedVersion         string
	DetectionStatus         string
	HealthStatus            string
	ChannelDrivers          []string
	EndpointStacks          []string
	Codecs                  []string
	Capabilities            AsteriskCapabilityConfig
	Subsystems              AsteriskSubsystemConfig
}

type AsteriskCapabilityConfig struct {
	Queues           bool
	Conferences      bool
	Recordings       bool
	Voicemail        bool
	Prompts          bool
	MusicOnHold      bool
	Fax              bool
	XMPP             bool
	Presence         bool
	DAHDI            bool
	BrowserCalling   bool
	TLS              bool
	MailDelivery     bool
	Metrics          bool
	Scheduler        bool
	DomainAutomation bool
}

type AsteriskSubsystemConfig struct {
	FaxBackend         string
	MessagingBackend   string
	TTSEngine          string
	MusicOnHoldSources string
}

type FeatureConfig struct {
	CustomDomains CustomDomainsFeatureConfig
}

type CustomDomainsFeatureConfig struct {
	Enabled           bool
	MaxDomainsPerUser int
	MaxDomainsPerOrg  int
	RequireSSL        bool
	AllowApex         bool
	AllowSubdomain    bool
	AllowWildcard     bool
	VerificationTTL   time.Duration
	SSLRenewalDays    int
	Reserved          []string
	BlockedPatterns   []string
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Mode:         AppModeProduction,
			DebugEnabled: false,
			Address:      "0.0.0.0",
			Port:         randomHighPort(),
			BaseURL:      "/",
			AdminPath:    "admin",
			Limits: RequestLimitsConfig{
				MaxBodySizeBytes: 10 * 1024 * 1024,
				ReadTimeoutSec:   30,
				WriteTimeoutSec:  30,
				IdleTimeoutSec:   120,
			},
			Compression: CompressionConfig{
				Enabled: true,
				Level:   5,
				Types: []string{
					"text/html",
					"text/css",
					"text/javascript",
					"application/json",
					"application/xml",
				},
			},
			TrustedProxies: TrustedProxiesConfig{Additional: []string{}},
			Session: SessionConfig{
				Admin: SessionScopeConfig{
					CookieName:       "admin_session",
					MaxAgeHours:      30 * 24,
					IdleTimeoutHours: 24,
				},
				User: SessionScopeConfig{
					CookieName:       "user_session",
					MaxAgeHours:      7 * 24,
					IdleTimeoutHours: 24,
				},
				ExtendOnActivity: true,
				Secure:           "auto",
				HTTPOnly:         true,
				SameSite:         "lax",
			},
			RateLimit: RateLimitConfig{
				Enabled:   true,
				Requests:  0,
				WindowSec: 60,
			},
			I18N: I18NConfig{
				DefaultLanguage: "en",
				Supported:       []string{"en"},
			},
			Contact: ContactConfig{
				Admin: ContactRoleConfig{
					Email: "admin@{fqdn}",
				},
				Security: ContactRoleConfig{
					Email: "security@{fqdn}",
				},
				General: ContactRoleConfig{},
			},
			Tracking: TrackingConfig{},
			Maintenance: MaintenanceConfig{
				SelfHealing: MaintenanceSelfHealingConfig{
					Enabled:          true,
					RetryIntervalSec: 30,
					MaxAttempts:      0,
				},
				Cleanup: MaintenanceCleanupConfig{
					DiskThresholdPercent: 90,
					LogRetentionDays:     7,
					BackupKeepCount:      5,
				},
				Notify: MaintenanceNotifyConfig{
					OnEnter: true,
					OnExit:  true,
				},
			},
			Asterisk: AsteriskConfig{
				MinimumSupportedVersion: "12",
				DetectionStatus:         "pending",
				HealthStatus:            "unknown",
				ChannelDrivers:          []string{},
				EndpointStacks:          []string{},
				Codecs:                  []string{},
				Capabilities: AsteriskCapabilityConfig{
					Recordings:  true,
					Voicemail:   true,
					Prompts:     true,
					MusicOnHold: true,
					Scheduler:   true,
				},
				Subsystems: AsteriskSubsystemConfig{
					TTSEngine:          "flite",
					MusicOnHoldSources: "local",
				},
			},
			Features: FeatureConfig{
				CustomDomains: CustomDomainsFeatureConfig{
					Enabled:           false,
					MaxDomainsPerUser: 5,
					MaxDomainsPerOrg:  20,
					RequireSSL:        true,
					AllowApex:         true,
					AllowSubdomain:    true,
					AllowWildcard:     false,
					VerificationTTL:   24 * time.Hour,
					SSLRenewalDays:    7,
					Reserved: []string{
						"localhost",
						"*.local",
						"*.test",
						"*.example",
						"*.invalid",
					},
					BlockedPatterns: []string{
						`.*\.(gov|mil|edu)$`,
					},
				},
			},
		},
	}
}

func (configValue *Config) Validate() []string {
	defaultConfig := DefaultConfig()
	warnings := []string{}

	if configValue.Server.Address == "" {
		configValue.Server.Address = defaultConfig.Server.Address
		warnings = append(warnings, "server.address invalid, using default")
	}

	if configValue.Server.Port < 1 || configValue.Server.Port > 65535 {
		configValue.Server.Port = randomHighPort()
		warnings = append(warnings, "server.port invalid, using random default")
	}

	normalizedBaseURL, normalizeBaseURLError := NormalizeBaseURL(configValue.Server.BaseURL)
	if normalizeBaseURLError != nil {
		configValue.Server.BaseURL = defaultConfig.Server.BaseURL
		warnings = append(warnings, "server.baseurl invalid, using default")
	} else {
		configValue.Server.BaseURL = normalizedBaseURL
	}

	normalizedAdminPath, normalizeAdminPathError := SafePath(configValue.Server.AdminPath)
	if normalizeAdminPathError != nil || normalizedAdminPath == "" {
		configValue.Server.AdminPath = defaultConfig.Server.AdminPath
		warnings = append(warnings, "server.admin_path invalid, using default")
	} else {
		configValue.Server.AdminPath = normalizedAdminPath
	}

	if configValue.Server.Limits.MaxBodySizeBytes <= 0 {
		configValue.Server.Limits.MaxBodySizeBytes = defaultConfig.Server.Limits.MaxBodySizeBytes
		warnings = append(warnings, "server.limits.max_body_size invalid, using default")
	}
	if configValue.Server.Limits.ReadTimeoutSec <= 0 {
		configValue.Server.Limits.ReadTimeoutSec = defaultConfig.Server.Limits.ReadTimeoutSec
		warnings = append(warnings, "server.limits.read_timeout invalid, using default")
	}
	if configValue.Server.Limits.WriteTimeoutSec <= 0 {
		configValue.Server.Limits.WriteTimeoutSec = defaultConfig.Server.Limits.WriteTimeoutSec
		warnings = append(warnings, "server.limits.write_timeout invalid, using default")
	}
	if configValue.Server.Limits.IdleTimeoutSec <= 0 {
		configValue.Server.Limits.IdleTimeoutSec = defaultConfig.Server.Limits.IdleTimeoutSec
		warnings = append(warnings, "server.limits.idle_timeout invalid, using default")
	}

	if configValue.Server.Compression.Level < 1 || configValue.Server.Compression.Level > 9 {
		configValue.Server.Compression.Level = defaultConfig.Server.Compression.Level
		warnings = append(warnings, "server.compression.level invalid, using default")
	}
	if len(configValue.Server.Compression.Types) == 0 {
		configValue.Server.Compression.Types = slices.Clone(defaultConfig.Server.Compression.Types)
		warnings = append(warnings, "server.compression.types invalid, using default")
	}

	if configValue.Server.Session.Admin.CookieName == "" {
		configValue.Server.Session.Admin.CookieName = defaultConfig.Server.Session.Admin.CookieName
		warnings = append(warnings, "server.session.admin.cookie_name invalid, using default")
	}
	if configValue.Server.Session.Admin.MaxAgeHours <= 0 {
		configValue.Server.Session.Admin.MaxAgeHours = defaultConfig.Server.Session.Admin.MaxAgeHours
		warnings = append(warnings, "server.session.admin.max_age invalid, using default")
	}
	if configValue.Server.Session.Admin.IdleTimeoutHours <= 0 {
		configValue.Server.Session.Admin.IdleTimeoutHours = defaultConfig.Server.Session.Admin.IdleTimeoutHours
		warnings = append(warnings, "server.session.admin.idle_timeout invalid, using default")
	}
	if configValue.Server.Session.User.CookieName == "" {
		configValue.Server.Session.User.CookieName = defaultConfig.Server.Session.User.CookieName
		warnings = append(warnings, "server.session.user.cookie_name invalid, using default")
	}
	if configValue.Server.Session.User.MaxAgeHours <= 0 {
		configValue.Server.Session.User.MaxAgeHours = defaultConfig.Server.Session.User.MaxAgeHours
		warnings = append(warnings, "server.session.user.max_age invalid, using default")
	}
	if configValue.Server.Session.User.IdleTimeoutHours <= 0 {
		configValue.Server.Session.User.IdleTimeoutHours = defaultConfig.Server.Session.User.IdleTimeoutHours
		warnings = append(warnings, "server.session.user.idle_timeout invalid, using default")
	}
	if configValue.Server.Session.Secure != "auto" && configValue.Server.Session.Secure != "true" && configValue.Server.Session.Secure != "false" {
		configValue.Server.Session.Secure = defaultConfig.Server.Session.Secure
		warnings = append(warnings, "server.session.secure invalid, using default")
	}
	if configValue.Server.Session.SameSite != "strict" && configValue.Server.Session.SameSite != "lax" && configValue.Server.Session.SameSite != "none" {
		configValue.Server.Session.SameSite = defaultConfig.Server.Session.SameSite
		warnings = append(warnings, "server.session.same_site invalid, using default")
	}

	if configValue.Server.RateLimit.Requests < 0 {
		configValue.Server.RateLimit.Requests = defaultConfig.Server.RateLimit.Requests
		warnings = append(warnings, "server.rate_limit.requests invalid, using default")
	}
	if configValue.Server.RateLimit.WindowSec <= 0 {
		configValue.Server.RateLimit.WindowSec = defaultConfig.Server.RateLimit.WindowSec
		warnings = append(warnings, "server.rate_limit.window invalid, using default")
	}

	if configValue.Server.I18N.DefaultLanguage == "" {
		configValue.Server.I18N.DefaultLanguage = defaultConfig.Server.I18N.DefaultLanguage
		warnings = append(warnings, "server.i18n.default_language invalid, using default")
	}
	if len(configValue.Server.I18N.Supported) == 0 {
		configValue.Server.I18N.Supported = slices.Clone(defaultConfig.Server.I18N.Supported)
		warnings = append(warnings, "server.i18n.supported invalid, using default")
	}

	switch configValue.Server.Tracking.Type {
	case "", "none", "google", "matomo", "piwik", "owa", "fathom", "plausible", "umami", "simple", "cloudflare":
	default:
		configValue.Server.Tracking = defaultConfig.Server.Tracking
		warnings = append(warnings, "server.tracking.type invalid, using default")
	}

	if configValue.Server.Maintenance.SelfHealing.RetryIntervalSec <= 0 {
		configValue.Server.Maintenance.SelfHealing.RetryIntervalSec = defaultConfig.Server.Maintenance.SelfHealing.RetryIntervalSec
		warnings = append(warnings, "server.maintenance.self_healing.retry_interval invalid, using default")
	}
	if configValue.Server.Maintenance.Cleanup.DiskThresholdPercent < 1 || configValue.Server.Maintenance.Cleanup.DiskThresholdPercent > 100 {
		configValue.Server.Maintenance.Cleanup.DiskThresholdPercent = defaultConfig.Server.Maintenance.Cleanup.DiskThresholdPercent
		warnings = append(warnings, "server.maintenance.cleanup.disk_threshold invalid, using default")
	}
	if configValue.Server.Maintenance.Cleanup.LogRetentionDays <= 0 {
		configValue.Server.Maintenance.Cleanup.LogRetentionDays = defaultConfig.Server.Maintenance.Cleanup.LogRetentionDays
		warnings = append(warnings, "server.maintenance.cleanup.log_retention_days invalid, using default")
	}
	if configValue.Server.Maintenance.Cleanup.BackupKeepCount <= 0 {
		configValue.Server.Maintenance.Cleanup.BackupKeepCount = defaultConfig.Server.Maintenance.Cleanup.BackupKeepCount
		warnings = append(warnings, "server.maintenance.cleanup.backup_keep_count invalid, using default")
	}

	if configValue.Server.Contact.Admin.Email == "" {
		configValue.Server.Contact.Admin.Email = defaultConfig.Server.Contact.Admin.Email
		warnings = append(warnings, "server.contact.admin.email invalid, using default")
	}
	if configValue.Server.Contact.Security.Email == "" {
		configValue.Server.Contact.Security.Email = defaultConfig.Server.Contact.Security.Email
	}
	if configValue.Server.Contact.General.Email == "" {
		configValue.Server.Contact.General.Email = configValue.Server.Contact.Admin.Email
	}
	if configValue.Server.Asterisk.MinimumSupportedVersion == "" {
		configValue.Server.Asterisk.MinimumSupportedVersion = defaultConfig.Server.Asterisk.MinimumSupportedVersion
		warnings = append(warnings, "server.asterisk.minimum_supported_version invalid, using default")
	}
	if !slices.Contains([]string{"pending", "detected", "failed"}, configValue.Server.Asterisk.DetectionStatus) {
		configValue.Server.Asterisk.DetectionStatus = defaultConfig.Server.Asterisk.DetectionStatus
		warnings = append(warnings, "server.asterisk.detection_status invalid, using default")
	}
	if !slices.Contains([]string{"unknown", "ready", "degraded", "error"}, configValue.Server.Asterisk.HealthStatus) {
		configValue.Server.Asterisk.HealthStatus = defaultConfig.Server.Asterisk.HealthStatus
		warnings = append(warnings, "server.asterisk.health_status invalid, using default")
	}
	if configValue.Server.Asterisk.ChannelDrivers == nil {
		configValue.Server.Asterisk.ChannelDrivers = slices.Clone(defaultConfig.Server.Asterisk.ChannelDrivers)
	}
	if configValue.Server.Asterisk.EndpointStacks == nil {
		configValue.Server.Asterisk.EndpointStacks = slices.Clone(defaultConfig.Server.Asterisk.EndpointStacks)
	}
	if configValue.Server.Asterisk.Codecs == nil {
		configValue.Server.Asterisk.Codecs = slices.Clone(defaultConfig.Server.Asterisk.Codecs)
	}
	if configValue.Server.Asterisk.Subsystems.TTSEngine == "" {
		configValue.Server.Asterisk.Subsystems.TTSEngine = defaultConfig.Server.Asterisk.Subsystems.TTSEngine
		warnings = append(warnings, "server.asterisk.subsystems.tts_engine invalid, using default")
	}
	if !slices.Contains([]string{"local", "remote", "mixed"}, configValue.Server.Asterisk.Subsystems.MusicOnHoldSources) {
		configValue.Server.Asterisk.Subsystems.MusicOnHoldSources = defaultConfig.Server.Asterisk.Subsystems.MusicOnHoldSources
		warnings = append(warnings, "server.asterisk.subsystems.music_on_hold_sources invalid, using default")
	}
	if configValue.Server.Features.CustomDomains.MaxDomainsPerUser < 0 {
		configValue.Server.Features.CustomDomains.MaxDomainsPerUser = defaultConfig.Server.Features.CustomDomains.MaxDomainsPerUser
		warnings = append(warnings, "server.features.custom_domains.max_domains_per_user invalid, using default")
	}
	if configValue.Server.Features.CustomDomains.MaxDomainsPerOrg < 0 {
		configValue.Server.Features.CustomDomains.MaxDomainsPerOrg = defaultConfig.Server.Features.CustomDomains.MaxDomainsPerOrg
		warnings = append(warnings, "server.features.custom_domains.max_domains_per_org invalid, using default")
	}
	if configValue.Server.Features.CustomDomains.VerificationTTL <= 0 {
		configValue.Server.Features.CustomDomains.VerificationTTL = defaultConfig.Server.Features.CustomDomains.VerificationTTL
		warnings = append(warnings, "server.features.custom_domains.verification_ttl invalid, using default")
	}
	if configValue.Server.Features.CustomDomains.SSLRenewalDays <= 0 {
		configValue.Server.Features.CustomDomains.SSLRenewalDays = defaultConfig.Server.Features.CustomDomains.SSLRenewalDays
		warnings = append(warnings, "server.features.custom_domains.ssl_renewal_days invalid, using default")
	}
	if len(configValue.Server.Features.CustomDomains.Reserved) == 0 {
		configValue.Server.Features.CustomDomains.Reserved = slices.Clone(defaultConfig.Server.Features.CustomDomains.Reserved)
		warnings = append(warnings, "server.features.custom_domains.reserved invalid, using default")
	}
	if len(configValue.Server.Features.CustomDomains.BlockedPatterns) == 0 {
		configValue.Server.Features.CustomDomains.BlockedPatterns = slices.Clone(defaultConfig.Server.Features.CustomDomains.BlockedPatterns)
		warnings = append(warnings, "server.features.custom_domains.blocked_patterns invalid, using default")
	}

	return warnings
}

func (configValue Config) Sanitized() Config {
	sanitizedConfig := configValue
	sanitizedConfig.Server.Contact.Admin.Webhooks = ContactWebhookConfig{}
	sanitizedConfig.Server.Contact.Security.Webhooks = ContactWebhookConfig{}
	sanitizedConfig.Server.Contact.General.Webhooks = ContactWebhookConfig{}
	return sanitizedConfig
}

func randomHighPort() int {
	randomOffset, randomError := rand.Int(randomHighPortReader, big.NewInt(1000))
	if randomError != nil {
		return 64580
	}
	return 64000 + int(randomOffset.Int64())
}

func (configValue Config) String() string {
	return fmt.Sprintf("mode=%s debug=%t address=%s port=%d baseurl=%s admin_path=%s",
		configValue.Server.Mode.String(),
		configValue.Server.DebugEnabled,
		configValue.Server.Address,
		configValue.Server.Port,
		configValue.Server.BaseURL,
		configValue.Server.AdminPath,
	)
}
