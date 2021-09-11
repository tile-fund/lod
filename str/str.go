package str

// CPadding is the current max caller padding, dynamically increased
var CPadding = 0

// Flags, log formats and miscellaneous strings
const (
	FDebugFlags      = "debug"
	FDebugFlagsUsage = "Comma separated debug flags [foo,bar,baz]"

	FConfigFile      = "c"
	FConfigFileUsage = "Path to TOML configuration file"

	InfoFormat  = "INFO  [%s] %s\n"
	DebugFormat = "DEBUG [%s] %s\n"
	ErrorFormat = "ERROR [%s] %s\n"
)

// (C) Log caller names
const (
	CMain = "LOD"
	CLog  = "Log"
	CTool = "Tool"
)

// (E) Error messages
const (
	ELogFail        = "failed to log error=%s msg=%+v"
	EConfig         = "failed to read config file error=%s"
	EConfigNotFound = "config file not found at path: '%s'"
	ERead           = "read err: %s"
	EWrite          = "write err: meta=%+v error=%s"
)

// (U) User-facing error messages and codes
const ()

// (M) Standard info log messages
const (
	MDevMode  = "!! DEVELOPER MODE !!"
	MInit     = "starting %s"
	MStarted  = "started in %s [env: %s][http: %s]"
	MShutdown = "shutting down"
	MExit     = "exit"
)

// (D) Debug log messages
const ()

// (T) Test messages
const ()
