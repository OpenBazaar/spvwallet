package astilog

import "flag"

// Flags
var (
	AppName  = flag.String("logger-app-name", "", "the logger's app name")
	Filename = flag.String("logger-filename", "", "the logger's filename")
	Verbose  = flag.Bool("logger-verbose", false, "if true, then log level is debug")
)

// Formats
const (
	FormatJSON = "json"
	FormatText = "text"
)

// Outs
const (
	OutFile   = "file"
	OutStdOut = "stdout"
	OutSyslog = "syslog"
)

// Configuration represents the configuration of the logger
type Configuration struct {
	AppName          string `toml:"app_name"`
	DisableColors    bool   `toml:"disable_colors"`
	DisableTimestamp bool   `toml:"disable_timestamp"`
	Filename         string `toml:"filename"`
	FullTimestamp    bool   `toml:"full_timestamp"`
	Format           string `toml:"format"`
	MessageKey       string `toml:"message_key"`
	Out              string `toml:"out"`
	TimestampFormat  string `toml:"timestamp_format"`
	Verbose          bool   `toml:"verbose"`
}

// SetHandyFlags sets handy flags
func SetHandyFlags() {
	Verbose = flag.Bool("v", false, "if true, then log level is debug")
}

// FlagConfig generates a Configuration based on flags
func FlagConfig() Configuration {
	return Configuration{
		AppName:  *AppName,
		Filename: *Filename,
		Verbose:  *Verbose,
	}
}
