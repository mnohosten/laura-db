package connstring

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidConnString is returned when the connection string is invalid
	ErrInvalidConnString = errors.New("invalid connection string")
	// ErrInvalidScheme is returned when the connection string scheme is not supported
	ErrInvalidScheme = errors.New("invalid scheme: must be 'laura://' or 'mongodb://'")
	// ErrNoHosts is returned when no hosts are specified
	ErrNoHosts = errors.New("no hosts specified in connection string")
)

// ConnString represents a parsed connection string
type ConnString struct {
	// Scheme is the connection protocol (laura or mongodb)
	Scheme string
	// Hosts is the list of host:port pairs
	Hosts []Host
	// Database is the optional database name
	Database string
	// Options contains connection options
	Options Options
}

// Host represents a host:port pair
type Host struct {
	Host string
	Port int
}

// Options contains connection string options
type Options struct {
	// Connection options
	Timeout         time.Duration
	MaxConnections  int
	MinConnections  int
	MaxIdleTime     time.Duration
	ConnectTimeout  time.Duration
	SocketTimeout   time.Duration

	// TLS/SSL options
	TLS         bool
	TLSInsecure bool
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	// Read/Write options
	ReadPreference string
	WriteConcern   string

	// Application options
	AppName string

	// Authentication
	Username string
	Password string
	AuthDB   string
	AuthMech string

	// Miscellaneous
	ReplicaSet string
	RetryWrites bool
	RetryReads  bool
}

// DefaultOptions returns default connection options
func DefaultOptions() Options {
	return Options{
		Timeout:        30 * time.Second,
		MaxConnections: 100,
		MinConnections: 1,
		MaxIdleTime:    10 * time.Minute,
		ConnectTimeout: 10 * time.Second,
		SocketTimeout:  30 * time.Second,
		ReadPreference: "primary",
		WriteConcern:   "majority",
		RetryWrites:    true,
		RetryReads:     true,
	}
}

// Parse parses a connection string into a ConnString struct
// Supported formats:
//   - laura://host:port/database?options
//   - laura://host1:port1,host2:port2/database?options
//   - mongodb://host:port/database?options (alias for laura://)
//   - mongodb://username:password@host:port/database?options
func Parse(connStr string) (*ConnString, error) {
	if connStr == "" {
		return nil, fmt.Errorf("%w: empty connection string", ErrInvalidConnString)
	}

	// Parse URL
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConnString, err)
	}

	// Validate scheme
	scheme := strings.ToLower(u.Scheme)
	if scheme != "laura" && scheme != "mongodb" {
		return nil, ErrInvalidScheme
	}

	cs := &ConnString{
		Scheme:  scheme,
		Options: DefaultOptions(),
	}

	// Extract authentication from userinfo
	if u.User != nil {
		cs.Options.Username = u.User.Username()
		if password, ok := u.User.Password(); ok {
			cs.Options.Password = password
		}
	}

	// Parse hosts
	hosts := u.Host
	if hosts == "" {
		return nil, ErrNoHosts
	}

	cs.Hosts, err = parseHosts(hosts)
	if err != nil {
		return nil, err
	}

	// Extract database name from path
	if u.Path != "" && u.Path != "/" {
		cs.Database = strings.TrimPrefix(u.Path, "/")
	}

	// Parse query options
	if u.RawQuery != "" {
		if err := parseOptions(&cs.Options, u.Query()); err != nil {
			return nil, err
		}
	}

	return cs, nil
}

// parseHosts parses a comma-separated list of host:port pairs
func parseHosts(hostStr string) ([]Host, error) {
	parts := strings.Split(hostStr, ",")
	hosts := make([]Host, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		host, portStr, hasPort := strings.Cut(part, ":")

		port := 8080 // default port
		if hasPort {
			var err error
			port, err = strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				return nil, fmt.Errorf("%w: invalid port '%s'", ErrInvalidConnString, portStr)
			}
		}

		hosts = append(hosts, Host{
			Host: host,
			Port: port,
		})
	}

	if len(hosts) == 0 {
		return nil, ErrNoHosts
	}

	return hosts, nil
}

// parseOptions parses query parameters into Options
func parseOptions(opts *Options, values url.Values) error {
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		val := vals[0] // use first value if multiple provided

		switch strings.ToLower(key) {
		// Connection options
		case "timeout", "timeoutms":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid timeout value: %v", err)
			}
			opts.Timeout = time.Duration(ms) * time.Millisecond

		case "maxconnections", "maxpoolsize":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid maxConnections value: %v", err)
			}
			opts.MaxConnections = n

		case "minconnections", "minpoolsize":
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid minConnections value: %v", err)
			}
			opts.MinConnections = n

		case "maxidletime", "maxidletimems":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid maxIdleTime value: %v", err)
			}
			opts.MaxIdleTime = time.Duration(ms) * time.Millisecond

		case "connecttimeout", "connecttimeoutms":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid connectTimeout value: %v", err)
			}
			opts.ConnectTimeout = time.Duration(ms) * time.Millisecond

		case "sockettimeout", "sockettimeoutms":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid socketTimeout value: %v", err)
			}
			opts.SocketTimeout = time.Duration(ms) * time.Millisecond

		// TLS options
		case "tls", "ssl":
			opts.TLS = parseBool(val)

		case "tlsinsecure", "tlsinsecureskipverify":
			opts.TLSInsecure = parseBool(val)

		case "tlscertfile", "tlscertificatekeyfile":
			opts.TLSCertFile = val

		case "tlskeyfile":
			opts.TLSKeyFile = val

		case "tlscafile":
			opts.TLSCAFile = val

		// Read/Write options
		case "readpreference":
			opts.ReadPreference = val

		case "writeconcern", "w":
			opts.WriteConcern = val

		// Application options
		case "appname":
			opts.AppName = val

		// Authentication
		case "authsource":
			opts.AuthDB = val

		case "authmechanism":
			opts.AuthMech = val

		// Miscellaneous
		case "replicaset":
			opts.ReplicaSet = val

		case "retrywrites":
			opts.RetryWrites = parseBool(val)

		case "retryreads":
			opts.RetryReads = parseBool(val)
		}
	}

	return nil
}

// parseBool parses a boolean value from string
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes"
}

// String returns the connection string representation
func (cs *ConnString) String() string {
	var sb strings.Builder

	// Scheme
	sb.WriteString(cs.Scheme)
	sb.WriteString("://")

	// Authentication
	if cs.Options.Username != "" {
		sb.WriteString(url.QueryEscape(cs.Options.Username))
		if cs.Options.Password != "" {
			sb.WriteString(":")
			sb.WriteString(url.QueryEscape(cs.Options.Password))
		}
		sb.WriteString("@")
	}

	// Hosts
	for i, host := range cs.Hosts {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(host.Host)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(host.Port))
	}

	// Database
	if cs.Database != "" {
		sb.WriteString("/")
		sb.WriteString(cs.Database)
	}

	return sb.String()
}

// GetFirstHost returns the first host in the connection string
func (cs *ConnString) GetFirstHost() Host {
	if len(cs.Hosts) == 0 {
		return Host{Host: "localhost", Port: 8080}
	}
	return cs.Hosts[0]
}

// HasAuthentication returns true if username is specified
func (cs *ConnString) HasAuthentication() bool {
	return cs.Options.Username != ""
}
