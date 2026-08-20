package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type RuntimeMode string

const (
	RuntimeShadow RuntimeMode = "shadow"
	RuntimeWriter RuntimeMode = "writer"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Token    TokenConfig
	Runtime  RuntimeConfig
	Jobs     JobConfig
	Outbound OutboundConfig
}

type OutboundConfig struct {
	AllowedCIDRs []*net.IPNet
	AllowedIPs   []net.IP
}

type ServerConfig struct {
	Host              string
	Port              int
	Mode              string
	FrontendURL       string
	AllowedOrigin     string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	Charset         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type TokenConfig struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

type RuntimeConfig struct {
	Mode RuntimeMode
}

type JobConfig struct {
	NotificationPollInterval   time.Duration
	NotificationAttemptTimeout time.Duration
	NotificationLeaseDuration  time.Duration
	NotificationBatchSize      int
	LifecyclePollInterval      time.Duration
	LifecycleAttemptTimeout    time.Duration
	LifecycleBatchSize         int
	MonitorHeartbeatTimeout    time.Duration
	MonitorSignTTL             time.Duration
}

func (c RuntimeConfig) CanWrite() bool {
	return c.Mode == RuntimeWriter
}

func Load() (Config, error) {
	loadDotEnv()

	serverPort, err := integer("VMQ_SERVER_PORT", 8080)
	if err != nil || serverPort < 1 || serverPort > 65535 {
		return Config{}, fmt.Errorf("VMQ_SERVER_PORT 配置无效")
	}

	databasePort, err := integerAny([]string{"VMQ_DB_PORT", "DB_HOSTPORT"}, 3306)
	if err != nil || databasePort < 1 || databasePort > 65535 {
		return Config{}, fmt.Errorf("数据库端口配置无效")
	}

	maxOpen, err := integer("VMQ_DB_MAX_OPEN_CONNS", 50)
	if err != nil || maxOpen < 1 {
		return Config{}, fmt.Errorf("VMQ_DB_MAX_OPEN_CONNS 配置无效")
	}
	maxIdle, err := integer("VMQ_DB_MAX_IDLE_CONNS", 10)
	if err != nil || maxIdle < 0 || maxIdle > maxOpen {
		return Config{}, fmt.Errorf("VMQ_DB_MAX_IDLE_CONNS 配置无效")
	}

	frontendURL := first([]string{"VMQ_FRONTEND_URL", "APP_FRONTEND_URL"}, "")
	allowedOrigin := first([]string{"VMQ_ALLOWED_ORIGIN"}, frontendURL)
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	mode := RuntimeMode(strings.ToLower(first([]string{"VMQ_RUNTIME_MODE", "VMQ_WRITE_MODE", "WRITE_MODE"}, string(RuntimeWriter))))
	if mode != RuntimeShadow && mode != RuntimeWriter {
		return Config{}, fmt.Errorf("VMQ_RUNTIME_MODE 只允许 %q 或 %q", RuntimeShadow, RuntimeWriter)
	}

	tokenSecret, ok := lookup("VMQ_TOKEN_SECRET", "TOKEN_SECRET")
	if !ok || len(strings.TrimSpace(tokenSecret)) < 32 {
		return Config{}, errors.New("VMQ_TOKEN_SECRET 必须显式配置且不少于 32 个字符")
	}
	tokenIssuer, ok := lookup("VMQ_TOKEN_ISSUER", "TOKEN_ISSUER")
	if !ok || strings.TrimSpace(tokenIssuer) == "" {
		return Config{}, errors.New("VMQ_TOKEN_ISSUER 必须显式配置")
	}
	tokenTTL, err := requiredDuration([]string{"VMQ_TOKEN_TTL", "TOKEN_TTL"})
	if err != nil || tokenTTL <= 0 {
		return Config{}, errors.New("VMQ_TOKEN_TTL 必须显式配置为正时长")
	}

	readHeaderTimeout, err := duration("VMQ_SERVER_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := duration("VMQ_SERVER_READ_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := duration("VMQ_SERVER_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := duration("VMQ_SERVER_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := duration("VMQ_SERVER_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	connectionLifetime, err := duration("VMQ_DB_CONN_MAX_LIFETIME", time.Hour)
	if err != nil {
		return Config{}, err
	}
	notificationPollInterval, err := duration("VMQ_NOTIFICATION_POLL_INTERVAL", 2*time.Second)
	if err != nil {
		return Config{}, err
	}
	notificationAttemptTimeout, err := duration("VMQ_NOTIFICATION_ATTEMPT_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	notificationLeaseDuration, err := duration("VMQ_NOTIFICATION_LEASE_DURATION", 45*time.Second)
	if err != nil {
		return Config{}, err
	}
	if notificationLeaseDuration <= notificationAttemptTimeout {
		return Config{}, errors.New("VMQ_NOTIFICATION_LEASE_DURATION 必须大于单次通知超时")
	}
	notificationBatchSize, err := integer("VMQ_NOTIFICATION_BATCH_SIZE", 20)
	if err != nil || notificationBatchSize < 1 || notificationBatchSize > 100 {
		return Config{}, errors.New("VMQ_NOTIFICATION_BATCH_SIZE 必须在 1 到 100 之间")
	}
	lifecyclePollInterval, err := duration("VMQ_LIFECYCLE_POLL_INTERVAL", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	lifecycleAttemptTimeout, err := duration("VMQ_LIFECYCLE_ATTEMPT_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	lifecycleBatchSize, err := integer("VMQ_LIFECYCLE_BATCH_SIZE", 100)
	if err != nil || lifecycleBatchSize < 1 || lifecycleBatchSize > 500 {
		return Config{}, errors.New("VMQ_LIFECYCLE_BATCH_SIZE 必须在 1 到 500 之间")
	}
	monitorHeartbeatTimeout, err := duration("VMQ_MONITOR_HEARTBEAT_TIMEOUT", 3*time.Minute)
	if err != nil {
		return Config{}, err
	}
	monitorSignTTL, err := duration("VMQ_MONITOR_SIGN_TTL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}

	outboundAllowedCIDRs, outboundAllowedIPs := parseAllowCIDR(first([]string{"VMQ_NOTIFY_ALLOW_CIDR", "NOTIFY_ALLOW_CIDR"}, ""))

	return Config{
		Server: ServerConfig{
			Host:              first([]string{"VMQ_SERVER_HOST"}, "0.0.0.0"),
			Port:              serverPort,
			Mode:              first([]string{"VMQ_SERVER_MODE"}, "release"),
			FrontendURL:       frontendURL,
			AllowedOrigin:     allowedOrigin,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
		},
		Database: DatabaseConfig{
			Host:            first([]string{"VMQ_DB_HOST", "DB_HOSTNAME"}, "127.0.0.1"),
			Port:            databasePort,
			Name:            first([]string{"VMQ_DB_NAME", "DB_DATABASE"}, "vmq"),
			User:            first([]string{"VMQ_DB_USER", "DB_USERNAME"}, "root"),
			Password:        first([]string{"VMQ_DB_PASSWORD", "DB_PASSWORD"}, ""),
			Charset:         first([]string{"VMQ_DB_CHARSET", "DB_CHARSET"}, "utf8mb4"),
			MaxOpenConns:    maxOpen,
			MaxIdleConns:    maxIdle,
			ConnMaxLifetime: connectionLifetime,
		},
		Token: TokenConfig{
			Secret: strings.TrimSpace(tokenSecret),
			Issuer: strings.TrimSpace(tokenIssuer),
			TTL:    tokenTTL,
		},
		Runtime: RuntimeConfig{Mode: mode},
		Jobs: JobConfig{
			NotificationPollInterval:   notificationPollInterval,
			NotificationAttemptTimeout: notificationAttemptTimeout,
			NotificationLeaseDuration:  notificationLeaseDuration,
			NotificationBatchSize:      notificationBatchSize,
			LifecyclePollInterval:      lifecyclePollInterval,
			LifecycleAttemptTimeout:    lifecycleAttemptTimeout,
			LifecycleBatchSize:         lifecycleBatchSize,
			MonitorHeartbeatTimeout:    monitorHeartbeatTimeout,
			MonitorSignTTL:             monitorSignTTL,
		},
		Outbound: OutboundConfig{
			AllowedCIDRs: outboundAllowedCIDRs,
			AllowedIPs:   outboundAllowedIPs,
		},
	}, nil
}

func lookup(names ...string) (string, bool) {
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			return value, true
		}
	}
	return "", false
}

func first(names []string, fallback string) string {
	if value, ok := lookup(names...); ok {
		return value
	}
	return fallback
}

func integer(name string, fallback int) (int, error) {
	return integerAny([]string{name}, fallback)
}

func integerAny(names []string, fallback int) (int, error) {
	value, ok := lookup(names...)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s 配置无效", name)
	}
	return parsed, nil
}

func requiredDuration(names []string) (time.Duration, error) {
	value, ok := lookup(names...)
	if !ok || strings.TrimSpace(value) == "" {
		return 0, errors.New("缺少时长配置")
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed, nil
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseAllowCIDR(raw string) ([]*net.IPNet, []net.IP) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var cidrs []*net.IPNet
	var ips []net.IP
	for _, item := range strings.Split(raw, ",") {
		token := strings.TrimSpace(item)
		if token == "" {
			continue
		}
		if strings.Contains(token, "/") {
			_, ipNet, err := net.ParseCIDR(token)
			if err == nil && ipNet != nil {
				cidrs = append(cidrs, ipNet)
			}
		} else {
			ip := net.ParseIP(token)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}
	return cidrs, ips
}

func loadDotEnv() {
	candidates := []string{".env", "../.env"}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
				(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				if len(val) >= 2 {
					val = val[1 : len(val)-1]
				}
			}
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
		break
	}
}
