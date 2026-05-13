package config

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
)

type failingRandomReader struct{}

func (failingRandomReader) Read(_ []byte) (int, error) {
	return 0, errors.New("random reader failed")
}

func TestParseBool(t *testing.T) {
	if _, parseError := ParseBool(""); parseError == nil {
		t.Fatalf("expected empty boolean to fail parsing")
	}

	trueValue, trueError := ParseBool("YES")
	if trueError != nil {
		t.Fatalf("expected truthy value to parse, got %v", trueError)
	}
	if !trueValue {
		t.Fatalf("expected YES to parse as true")
	}

	falseValue, falseError := ParseBool("disabled")
	if falseError != nil {
		t.Fatalf("expected falsy value to parse, got %v", falseError)
	}
	if falseValue {
		t.Fatalf("expected disabled to parse as false")
	}

	if _, parseError := ParseBool("sometimes"); parseError == nil {
		t.Fatalf("expected invalid boolean to fail parsing")
	}

	if !IsTruthy("on") {
		t.Fatalf("expected IsTruthy to recognize truthy value")
	}
	if !IsFalsy("off") {
		t.Fatalf("expected IsFalsy to recognize falsy value")
	}
	if IsTruthy("maybe") {
		t.Fatalf("expected invalid value to be non-truthy")
	}
}

func TestResolveAppModePriority(t *testing.T) {
	if AppModeProduction.String() != "production" {
		t.Fatalf("expected production string output")
	}
	if AppModeDevelopment.String() != "development" {
		t.Fatalf("expected development string output")
	}

	if appMode, parseError := ParseAppMode("production"); parseError != nil || appMode != AppModeProduction {
		t.Fatalf("expected production mode parse, got %v / %v", appMode, parseError)
	}
	if appMode, parseError := ParseAppMode(""); parseError != nil || appMode != AppModeProduction {
		t.Fatalf("expected empty mode to map to production, got %v / %v", appMode, parseError)
	}
	if appMode, parseError := ParseAppMode("development"); parseError != nil || appMode != AppModeDevelopment {
		t.Fatalf("expected development mode parse, got %v / %v", appMode, parseError)
	}
	if _, parseError := ParseAppMode("mystery"); parseError == nil {
		t.Fatalf("expected invalid mode parse to fail")
	}

	if appMode := ResolveAppMode("development", true, "production"); appMode != AppModeDevelopment {
		t.Fatalf("expected CLI mode to win, got %s", appMode.String())
	}

	if appMode := ResolveAppMode("mystery", true, "development"); appMode != AppModeProduction {
		t.Fatalf("expected invalid CLI mode to fall back to production, got %s", appMode.String())
	}

	if appMode := ResolveAppMode("production", false, "dev"); appMode != AppModeDevelopment {
		t.Fatalf("expected env mode to be used, got %s", appMode.String())
	}

	if appMode := ResolveAppMode("production", false, "invalid"); appMode != AppModeProduction {
		t.Fatalf("expected invalid env mode to fall back to production, got %s", appMode.String())
	}

	if label := FormatAppModeLabel(AppModeDevelopment, true); label != "development [debugging]" {
		t.Fatalf("unexpected debug label %q", label)
	}
	if label := FormatAppModeLabel(AppModeProduction, false); label != "production" {
		t.Fatalf("unexpected production label %q", label)
	}
}

func TestResolveDebugEnabledPriority(t *testing.T) {
	if !ResolveDebugEnabled(true, true, "false") {
		t.Fatalf("expected CLI debug to win")
	}

	if ResolveDebugEnabled(false, true, "true") {
		t.Fatalf("expected explicit CLI false to win")
	}

	if !ResolveDebugEnabled(false, false, "enabled") {
		t.Fatalf("expected env debug to be used")
	}
}

func TestSafePath(t *testing.T) {
	if normalizedPath := normalizePath(""); normalizedPath != "" {
		t.Fatalf("expected empty normalized path, got %q", normalizedPath)
	}
	if normalizedPath := normalizePath(".."); normalizedPath != "" {
		t.Fatalf("expected traversal normalized path to be blank, got %q", normalizedPath)
	}

	normalizedPath, normalizeError := SafePath("//admin//queue")
	if normalizeError != nil {
		t.Fatalf("expected valid path, got %v", normalizeError)
	}
	if normalizedPath != "admin/queue" {
		t.Fatalf("expected normalized path, got %q", normalizedPath)
	}

	if _, pathError := SafePath("/admin/../secret"); pathError == nil {
		t.Fatalf("expected traversal path to be rejected")
	}

	if _, pathError := SafePath("/Admin"); pathError == nil {
		t.Fatalf("expected uppercase path to be rejected")
	}

	if validationError := validatePathSegment(""); !errors.Is(validationError, ErrInvalidPath) {
		t.Fatalf("expected empty segment error, got %v", validationError)
	}
	if validationError := validatePathSegment("."); !errors.Is(validationError, ErrPathTraversal) {
		t.Fatalf("expected dot segment traversal error, got %v", validationError)
	}
	if validationError := validatePathSegment(strings.Repeat("a", 65)); !errors.Is(validationError, ErrPathTooLong) {
		t.Fatalf("expected long segment error, got %v", validationError)
	}
	if validationError := validatePath(strings.Repeat("a", 2049)); !errors.Is(validationError, ErrPathTooLong) {
		t.Fatalf("expected long path error, got %v", validationError)
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	normalizedBaseURL, normalizeError := NormalizeBaseURL("pbx/admin/")
	if normalizeError != nil {
		t.Fatalf("expected baseurl to normalize, got %v", normalizeError)
	}
	if normalizedBaseURL != "/pbx/admin" {
		t.Fatalf("expected normalized baseurl, got %q", normalizedBaseURL)
	}

	rootBaseURL, rootError := NormalizeBaseURL("/")
	if rootError != nil {
		t.Fatalf("expected root baseurl to be valid, got %v", rootError)
	}
	if rootBaseURL != "/" {
		t.Fatalf("expected root baseurl, got %q", rootBaseURL)
	}

	emptyBaseURL, emptyError := NormalizeBaseURL("")
	if emptyError != nil {
		t.Fatalf("expected empty baseurl to normalize to root, got %v", emptyError)
	}
	if emptyBaseURL != "/" {
		t.Fatalf("expected empty baseurl to normalize to root, got %q", emptyBaseURL)
	}

	if _, traversalError := NormalizeBaseURL("/pbx/../secret"); traversalError == nil {
		t.Fatalf("expected traversal baseurl to fail")
	}

	if _, invalidError := NormalizeBaseURL("/PBX"); invalidError == nil {
		t.Fatalf("expected uppercase baseurl to fail")
	}
}

func TestResolveRuntimePaths(t *testing.T) {
	linuxUserPaths, linuxUserError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:    "linux",
		HomeDir: "/home/alice",
	})
	if linuxUserError != nil {
		t.Fatalf("expected linux user paths, got %v", linuxUserError)
	}
	if linuxUserPaths.ConfigFile != "/home/alice/.config/casapps/caspbx/server.yml" {
		t.Fatalf("unexpected linux user config file %q", linuxUserPaths.ConfigFile)
	}

	linuxRootPaths, linuxRootError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:         "linux",
		IsPrivileged: true,
	})
	if linuxRootError != nil {
		t.Fatalf("expected linux root paths, got %v", linuxRootError)
	}
	if linuxRootPaths.DataDir != "/var/lib/casapps/caspbx/" {
		t.Fatalf("unexpected linux root data dir %q", linuxRootPaths.DataDir)
	}

	if _, linuxMissingHomeError := ResolveRuntimePaths(RuntimePathOptions{GoOS: "linux"}); linuxMissingHomeError == nil {
		t.Fatalf("expected linux missing home error")
	}

	containerPaths, containerError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:        "linux",
		IsContainer: true,
	})
	if containerError != nil {
		t.Fatalf("expected container paths, got %v", containerError)
	}
	if containerPaths.ConfigDir != "/config/caspbx/" {
		t.Fatalf("unexpected container config dir %q", containerPaths.ConfigDir)
	}

	darwinPaths, darwinError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:    "darwin",
		HomeDir: "/Users/alice",
	})
	if darwinError != nil {
		t.Fatalf("expected darwin paths, got %v", darwinError)
	}
	if darwinPaths.LogDir != "/Users/alice/Library/Logs/casapps/caspbx/" {
		t.Fatalf("unexpected darwin log dir %q", darwinPaths.LogDir)
	}

	darwinRootPaths, darwinRootError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:         "darwin",
		IsPrivileged: true,
	})
	if darwinRootError != nil {
		t.Fatalf("expected darwin root paths, got %v", darwinRootError)
	}
	if darwinRootPaths.ConfigDir != "/Library/Application Support/casapps/caspbx/" {
		t.Fatalf("unexpected darwin root config dir %q", darwinRootPaths.ConfigDir)
	}
	if _, darwinMissingHomeError := ResolveRuntimePaths(RuntimePathOptions{GoOS: "darwin"}); darwinMissingHomeError == nil {
		t.Fatalf("expected darwin missing home error")
	}

	bsdUserPaths, bsdUserError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:    "freebsd",
		HomeDir: "/home/alice",
	})
	if bsdUserError != nil {
		t.Fatalf("expected bsd user paths, got %v", bsdUserError)
	}
	if bsdUserPaths.ConfigDir != "/home/alice/.config/casapps/caspbx/" {
		t.Fatalf("unexpected bsd config dir %q", bsdUserPaths.ConfigDir)
	}
	bsdRootPaths, bsdRootError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:         "openbsd",
		IsPrivileged: true,
	})
	if bsdRootError != nil {
		t.Fatalf("expected bsd root paths, got %v", bsdRootError)
	}
	if bsdRootPaths.LogDir != "/var/log/casapps/caspbx/" {
		t.Fatalf("unexpected bsd root log dir %q", bsdRootPaths.LogDir)
	}
	if _, bsdMissingHomeError := ResolveRuntimePaths(RuntimePathOptions{GoOS: "netbsd"}); bsdMissingHomeError == nil {
		t.Fatalf("expected bsd missing home error")
	}

	windowsPaths, windowsError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:    "windows",
		HomeDir: `C:\Users\alice`,
	})
	if windowsError != nil {
		t.Fatalf("expected windows paths, got %v", windowsError)
	}
	if windowsPaths.ConfigFile != `C:\Users\alice\AppData\Roaming\casapps\caspbx\server.yml` {
		t.Fatalf("unexpected windows config file %q", windowsPaths.ConfigFile)
	}

	windowsRootPaths, windowsRootError := ResolveRuntimePaths(RuntimePathOptions{
		GoOS:         "windows",
		IsPrivileged: true,
	})
	if windowsRootError != nil {
		t.Fatalf("expected windows root paths, got %v", windowsRootError)
	}
	if windowsRootPaths.DataDir != `%ProgramData%\casapps\caspbx\data\` {
		t.Fatalf("unexpected windows root data dir %q", windowsRootPaths.DataDir)
	}
	if _, windowsMissingHomeError := ResolveRuntimePaths(RuntimePathOptions{GoOS: "windows"}); windowsMissingHomeError == nil {
		t.Fatalf("expected windows missing home error")
	}

	if _, unsupportedOSError := ResolveRuntimePaths(RuntimePathOptions{GoOS: "plan9"}); unsupportedOSError == nil {
		t.Fatalf("expected unsupported os error")
	}
}

func TestDefaultConfigValidate(t *testing.T) {
	configValue := DefaultConfig()
	configValue.Server.Address = ""
	configValue.Server.Port = 70000
	configValue.Server.BaseURL = "/PBX"
	configValue.Server.AdminPath = "../admin"
	configValue.Server.Limits.MaxBodySizeBytes = 0
	configValue.Server.Limits.ReadTimeoutSec = 0
	configValue.Server.Limits.WriteTimeoutSec = 0
	configValue.Server.Limits.IdleTimeoutSec = 0
	configValue.Server.Compression.Level = 10
	configValue.Server.Compression.Types = nil
	configValue.Server.Session.Admin.CookieName = ""
	configValue.Server.Session.Admin.MaxAgeHours = 0
	configValue.Server.Session.Admin.IdleTimeoutHours = 0
	configValue.Server.Session.User.CookieName = ""
	configValue.Server.Session.User.MaxAgeHours = 0
	configValue.Server.Session.User.IdleTimeoutHours = 0
	configValue.Server.Session.Secure = "sometimes"
	configValue.Server.Session.SameSite = "wide-open"
	configValue.Server.RateLimit.Requests = -1
	configValue.Server.RateLimit.WindowSec = 0
	configValue.Server.I18N.DefaultLanguage = ""
	configValue.Server.I18N.Supported = nil
	configValue.Server.Tracking.Type = "mystery"
	configValue.Server.Maintenance.SelfHealing.RetryIntervalSec = 0
	configValue.Server.Maintenance.Cleanup.DiskThresholdPercent = 101
	configValue.Server.Maintenance.Cleanup.LogRetentionDays = 0
	configValue.Server.Maintenance.Cleanup.BackupKeepCount = 0
	configValue.Server.Contact.Admin.Email = ""
	configValue.Server.Contact.Security.Email = ""
	configValue.Server.Contact.General.Email = ""

	warnings := configValue.Validate()

	if configValue.Server.Address != "0.0.0.0" {
		t.Fatalf("expected invalid address to reset, got %q", configValue.Server.Address)
	}
	if configValue.Server.Port < 64000 || configValue.Server.Port > 64999 {
		t.Fatalf("expected invalid port to be replaced with high port, got %d", configValue.Server.Port)
	}
	if configValue.Server.BaseURL != "/" {
		t.Fatalf("expected invalid baseurl to reset, got %q", configValue.Server.BaseURL)
	}
	if configValue.Server.AdminPath != "admin" {
		t.Fatalf("expected invalid admin path to reset, got %q", configValue.Server.AdminPath)
	}
	if configValue.Server.Limits.MaxBodySizeBytes != 10*1024*1024 {
		t.Fatalf("expected body size default, got %d", configValue.Server.Limits.MaxBodySizeBytes)
	}
	if configValue.Server.Limits.ReadTimeoutSec != 30 {
		t.Fatalf("expected read timeout default, got %d", configValue.Server.Limits.ReadTimeoutSec)
	}
	if configValue.Server.Limits.WriteTimeoutSec != 30 {
		t.Fatalf("expected write timeout default, got %d", configValue.Server.Limits.WriteTimeoutSec)
	}
	if configValue.Server.Limits.IdleTimeoutSec != 120 {
		t.Fatalf("expected idle timeout default, got %d", configValue.Server.Limits.IdleTimeoutSec)
	}
	if configValue.Server.Compression.Level != 5 {
		t.Fatalf("expected compression default, got %d", configValue.Server.Compression.Level)
	}
	if len(configValue.Server.Compression.Types) == 0 {
		t.Fatalf("expected compression types default")
	}
	if configValue.Server.Session.Admin.CookieName != "admin_session" {
		t.Fatalf("expected admin cookie default, got %q", configValue.Server.Session.Admin.CookieName)
	}
	if configValue.Server.Session.Admin.MaxAgeHours != 30*24 {
		t.Fatalf("expected admin max age default, got %d", configValue.Server.Session.Admin.MaxAgeHours)
	}
	if configValue.Server.Session.Admin.IdleTimeoutHours != 24 {
		t.Fatalf("expected admin idle timeout default, got %d", configValue.Server.Session.Admin.IdleTimeoutHours)
	}
	if configValue.Server.Session.User.CookieName != "user_session" {
		t.Fatalf("expected user cookie default, got %q", configValue.Server.Session.User.CookieName)
	}
	if configValue.Server.Session.User.MaxAgeHours != 7*24 {
		t.Fatalf("expected user max age default, got %d", configValue.Server.Session.User.MaxAgeHours)
	}
	if configValue.Server.Session.User.IdleTimeoutHours != 24 {
		t.Fatalf("expected user idle timeout default, got %d", configValue.Server.Session.User.IdleTimeoutHours)
	}
	if configValue.Server.Session.Secure != "auto" {
		t.Fatalf("expected secure default, got %q", configValue.Server.Session.Secure)
	}
	if configValue.Server.Session.SameSite != "lax" {
		t.Fatalf("expected same_site default, got %q", configValue.Server.Session.SameSite)
	}
	if configValue.Server.RateLimit.Requests != 0 {
		t.Fatalf("expected rate limit requests default, got %d", configValue.Server.RateLimit.Requests)
	}
	if configValue.Server.RateLimit.WindowSec != 60 {
		t.Fatalf("expected rate limit window default, got %d", configValue.Server.RateLimit.WindowSec)
	}
	if configValue.Server.I18N.DefaultLanguage != "en" {
		t.Fatalf("expected i18n default language, got %q", configValue.Server.I18N.DefaultLanguage)
	}
	if !slices.Equal(configValue.Server.I18N.Supported, []string{"en"}) {
		t.Fatalf("expected i18n supported default, got %v", configValue.Server.I18N.Supported)
	}
	if configValue.Server.Tracking.Type != "" {
		t.Fatalf("expected tracking type reset, got %q", configValue.Server.Tracking.Type)
	}
	if configValue.Server.Maintenance.SelfHealing.RetryIntervalSec != 30 {
		t.Fatalf("expected retry interval default, got %d", configValue.Server.Maintenance.SelfHealing.RetryIntervalSec)
	}
	if configValue.Server.Maintenance.Cleanup.DiskThresholdPercent != 90 {
		t.Fatalf("expected disk threshold default, got %d", configValue.Server.Maintenance.Cleanup.DiskThresholdPercent)
	}
	if configValue.Server.Maintenance.Cleanup.LogRetentionDays != 7 {
		t.Fatalf("expected log retention default, got %d", configValue.Server.Maintenance.Cleanup.LogRetentionDays)
	}
	if configValue.Server.Maintenance.Cleanup.BackupKeepCount != 5 {
		t.Fatalf("expected backup keep count default, got %d", configValue.Server.Maintenance.Cleanup.BackupKeepCount)
	}
	if configValue.Server.Contact.Admin.Email != "admin@{fqdn}" {
		t.Fatalf("expected admin email default, got %q", configValue.Server.Contact.Admin.Email)
	}
	if configValue.Server.Contact.Security.Email != "security@{fqdn}" {
		t.Fatalf("expected security email default, got %q", configValue.Server.Contact.Security.Email)
	}
	if configValue.Server.Contact.General.Email != "admin@{fqdn}" {
		t.Fatalf("expected general email to follow admin, got %q", configValue.Server.Contact.General.Email)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "server.port invalid") {
		t.Fatalf("expected validation warnings, got %v", warnings)
	}
}

func TestDefaultConfigValidateAcceptedValues(t *testing.T) {
	configValue := DefaultConfig()
	configValue.Server.BaseURL = "/pbx"
	configValue.Server.AdminPath = "admin/server/asterisk"
	configValue.Server.Session.Secure = "true"
	configValue.Server.Session.SameSite = "strict"
	configValue.Server.Tracking.Type = "matomo"
	configValue.Server.Contact.Security.Email = "sec@example.invalid"
	configValue.Server.Contact.General.Email = "support@example.invalid"

	warnings := configValue.Validate()

	if len(warnings) != 0 {
		t.Fatalf("expected accepted config to validate cleanly, got %v", warnings)
	}
	if configValue.Server.BaseURL != "/pbx" {
		t.Fatalf("expected baseurl preserved, got %q", configValue.Server.BaseURL)
	}
	if configValue.Server.AdminPath != "admin/server/asterisk" {
		t.Fatalf("expected admin path preserved, got %q", configValue.Server.AdminPath)
	}
	if configValue.Server.Session.Secure != "true" {
		t.Fatalf("expected secure value preserved, got %q", configValue.Server.Session.Secure)
	}
	if configValue.Server.Session.SameSite != "strict" {
		t.Fatalf("expected same_site preserved, got %q", configValue.Server.Session.SameSite)
	}
	if configValue.Server.Tracking.Type != "matomo" {
		t.Fatalf("expected tracking value preserved, got %q", configValue.Server.Tracking.Type)
	}
}

func TestSanitizedConfig(t *testing.T) {
	configValue := DefaultConfig()
	configValue.Server.Contact.Admin.Webhooks.Telegram = "secret"
	configValue.Server.Contact.Security.Webhooks.Slack = "secret"
	configValue.Server.Contact.General.Webhooks.Generic = "secret"

	sanitizedConfig := configValue.Sanitized()

	if sanitizedConfig.Server.Contact.Admin.Webhooks.Telegram != "" {
		t.Fatalf("expected admin webhook to be redacted")
	}
	if sanitizedConfig.Server.Contact.Security.Webhooks.Slack != "" {
		t.Fatalf("expected security webhook to be redacted")
	}
	if sanitizedConfig.Server.Contact.General.Webhooks.Generic != "" {
		t.Fatalf("expected general webhook to be redacted")
	}
}

func TestConfigStringAndRandomFallback(t *testing.T) {
	configValue := DefaultConfig()
	configValue.Server.Mode = AppModeDevelopment
	configValue.Server.DebugEnabled = true
	configString := configValue.String()

	if !strings.Contains(configString, "mode=development") || !strings.Contains(configString, "debug=true") {
		t.Fatalf("unexpected config string %q", configString)
	}

	originalRandomReader := randomHighPortReader
	randomHighPortReader = failingRandomReader{}
	defer func() {
		randomHighPortReader = originalRandomReader
	}()

	if fallbackPort := randomHighPort(); fallbackPort != 64580 {
		t.Fatalf("expected fallback port, got %d", fallbackPort)
	}
}

func TestCustomDomainConfigDefaultsAndValidation(t *testing.T) {
	defaultConfig := DefaultConfig()
	if defaultConfig.Server.Features.CustomDomains.MaxDomainsPerUser != 5 || defaultConfig.Server.Features.CustomDomains.VerificationTTL != 24*time.Hour {
		t.Fatalf("unexpected custom domain defaults %+v", defaultConfig.Server.Features.CustomDomains)
	}

	configValue := DefaultConfig()
	configValue.Server.Features.CustomDomains.MaxDomainsPerUser = -1
	configValue.Server.Features.CustomDomains.MaxDomainsPerOrg = -1
	configValue.Server.Features.CustomDomains.VerificationTTL = 0
	configValue.Server.Features.CustomDomains.SSLRenewalDays = 0
	configValue.Server.Features.CustomDomains.Reserved = nil
	configValue.Server.Features.CustomDomains.BlockedPatterns = nil

	warnings := configValue.Validate()
	if len(warnings) < 6 {
		t.Fatalf("expected custom domain validation warnings, got %+v", warnings)
	}
	if configValue.Server.Features.CustomDomains.MaxDomainsPerUser != defaultConfig.Server.Features.CustomDomains.MaxDomainsPerUser {
		t.Fatalf("expected user limit reset to default")
	}
	if !slices.Equal(configValue.Server.Features.CustomDomains.Reserved, defaultConfig.Server.Features.CustomDomains.Reserved) {
		t.Fatalf("expected reserved domains reset to default")
	}
}

func TestAsteriskConfigDefaultsAndValidation(t *testing.T) {
	defaultConfig := DefaultConfig()
	if defaultConfig.Server.Asterisk.MinimumSupportedVersion != "12" || defaultConfig.Server.Asterisk.Subsystems.TTSEngine != "flite" {
		t.Fatalf("unexpected asterisk defaults %+v", defaultConfig.Server.Asterisk)
	}

	configValue := DefaultConfig()
	configValue.Server.Asterisk.MinimumSupportedVersion = ""
	configValue.Server.Asterisk.DetectionStatus = "broken"
	configValue.Server.Asterisk.HealthStatus = "bad"
	configValue.Server.Asterisk.ChannelDrivers = nil
	configValue.Server.Asterisk.EndpointStacks = nil
	configValue.Server.Asterisk.Codecs = nil
	configValue.Server.Asterisk.Subsystems.TTSEngine = ""
	configValue.Server.Asterisk.Subsystems.MusicOnHoldSources = "invalid"

	warnings := configValue.Validate()
	if len(warnings) < 5 {
		t.Fatalf("expected asterisk validation warnings, got %+v", warnings)
	}
	if configValue.Server.Asterisk.MinimumSupportedVersion != defaultConfig.Server.Asterisk.MinimumSupportedVersion {
		t.Fatalf("expected minimum supported version reset to default")
	}
	if configValue.Server.Asterisk.DetectionStatus != defaultConfig.Server.Asterisk.DetectionStatus || configValue.Server.Asterisk.HealthStatus != defaultConfig.Server.Asterisk.HealthStatus {
		t.Fatalf("expected asterisk statuses reset to defaults")
	}
	if configValue.Server.Asterisk.Subsystems.TTSEngine != "flite" || configValue.Server.Asterisk.Subsystems.MusicOnHoldSources != "local" {
		t.Fatalf("expected asterisk subsystem defaults restored")
	}
}

var _ io.Reader = failingRandomReader{}
