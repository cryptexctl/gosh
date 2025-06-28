package config

type Config struct {
	Command    string
	ScriptFile string
	ScriptArgs []string
	ReadStdin  bool

	NoRC        bool
	NoProfile   bool
	POSIX       bool
	Debug       bool
	Interactive bool
	Login       bool

	HistorySize    int
	HistoryFile    string
	MaxJobHistory  int
	CommandTimeout int

	PS1 string
	PS2 string
	PS3 string
	PS4 string

	EnableColors     bool
	EnableCompletion bool
}

func New() *Config {
	return &Config{
		HistorySize:    1000,
		HistoryFile:    "~/.gosh_history",
		MaxJobHistory:  100,
		CommandTimeout: 0,

		PS1: "\\u@\\h:\\w\\$ ",
		PS2: "> ",
		PS3: "#? ",
		PS4: "+ ",

		EnableColors:     true,
		EnableCompletion: true,
	}
}
