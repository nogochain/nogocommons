// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"fmt"
)

// Mode specifies the PoW engine operation mode.
type Mode uint

const (
	ModeNormal Mode = iota
	ModeFake
	ModeTest
)

// ConsensusParams holds consensus-critical configuration parameters
// for the NogoPow engine. These are derived from chaincfg.Params.
type ConsensusParams struct {
	ChainID                      int
	DifficultyEnable             bool
	BlockTimeTargetSeconds       int
	MinDifficulty                int
	MaxDifficultyChangePercent   int
	DifficultyAdjustmentInterval int
}

// Config holds the NogoPow engine configuration.
type Config struct {
	powMode         Mode // unexported - immutable after construction (prevents runtime bypass to Fake)
	CacheDir        string
	Log             Logger
	ConsensusParams *ConsensusParams
	UseSIMD         bool
	UseBitShift     bool
	ReuseObjects    bool
}

// PowMode returns the engine's PoW mode (immutable after construction).
func (c *Config) PowMode() Mode {
	return c.powMode
}

// powModeForTest is an internal setter ONLY for test code in this package.
func (c *Config) powModeForTest(m Mode) {
	c.powMode = m
}

// powModeForInit sets the PoW mode during initialization (genesis creation).
// Internal use only; external callers must use NewConfigForGenesis.
// After initialization, mode is immutable.
func (c *Config) powModeForInit(m Mode) {
	c.powMode = m
}

// NewConfigForGenesis creates a Config suitable for genesis block creation.
// Accepts the PoW mode to use during genesis (typically ModeFake for fast creation).
// After genesis is sealed, the engine should be recreated with DefaultConfig.
func NewConfigForGenesis(mode Mode) *Config {
	cfg := DefaultConfig()
	cfg.powMode = mode
	return cfg
}

// DefaultConfig returns a Config with safe defaults.
// ConsensusParams is intentionally nil - callers MUST set it before use.
// All production callers already override it with Chain.c.consensus.
// NewDifficultyAdjuster() has internal nil-check fallback for standalone mode.
//
// For genesis creation, use NewConfigForGenesis() instead.
func DefaultConfig() *Config {
	return &Config{
		powMode:      ModeNormal,
		CacheDir:     "",
		Log:          &defaultLogger{},
		UseSIMD:      false,
		UseBitShift:  false,
		ReuseObjects: true,
		// ConsensusParams deliberately nil — must be configured by caller.
		// Runtime nil-guards exist in NewDifficultyAdjuster() and CalcDifficulty().
	}
}

// ValidateConsensusParams returns an error if ConsensusParams is nil.
// Call before any operation that requires consensus parameters.
func (c *Config) ValidateConsensusParams() error {
	if c.ConsensusParams == nil {
		return fmt.Errorf("ConsensusParams is nil: must be set before using the engine")
	}
	return nil
}

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// GlobalLogLevel controls the minimum log level to display
// Production environments should set this to LogLevelInfo or higher
var GlobalLogLevel = LogLevelInfo

type Logger interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	if GlobalLogLevel <= LogLevelInfo {
		println(formatLog("INFO", msg, args...))
	}
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	if GlobalLogLevel <= LogLevelDebug {
		println(formatLog("DEBUG", msg, args...))
	}
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	if GlobalLogLevel <= LogLevelError {
		println(formatLog("ERROR", msg, args...))
	}
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	if GlobalLogLevel <= LogLevelWarn {
		println(formatLog("WARN", msg, args...))
	}
}

func formatLog(level, msg string, args ...interface{}) string {
	result := "[" + level + "] " + msg
	if len(args) > 0 {
		result += " " + sprintArgs(args...)
	}
	return result
}

func sprintArgs(args ...interface{}) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%v", arg)
	}
	return result
}
