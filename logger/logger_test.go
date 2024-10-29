package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venturoid/go/logger"
	"go.uber.org/zap/zapcore"
)

type (
	MapType   map[string]interface{}
	ArrayType []interface{}
)

func TestToZapLogLevel(t *testing.T) {
	testCases := []struct {
		TestName string

		logLevelInput string
		ExpectedLevel zapcore.Level
	}{
		{
			TestName:      "Return DebugLevel",
			logLevelInput: "debug",
			ExpectedLevel: zapcore.DebugLevel,
		},
		{
			TestName:      "Return WarnLevel",
			logLevelInput: "warn",
			ExpectedLevel: zapcore.WarnLevel,
		},
		{
			TestName:      "Return ErrorLevel",
			logLevelInput: "error",
			ExpectedLevel: zapcore.ErrorLevel,
		},
		{
			TestName:      "Return InfoLevel",
			logLevelInput: "info",
			ExpectedLevel: zapcore.InfoLevel,
		},
		{
			TestName:      "Return InfoLevel (default value)",
			logLevelInput: "asdas",
			ExpectedLevel: zapcore.InfoLevel,
		},
	}

	for _, c := range testCases {
		t.Run(c.TestName, func(t *testing.T) {
			assert.Equal(t, c.ExpectedLevel, logger.ToZapLogLevel(c.logLevelInput))
		})
	}
}
