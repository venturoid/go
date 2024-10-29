package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Zap *zap.Logger

func New(logLevel string) *zap.Logger {
	if Zap != nil {
		return Zap
	}
	w := zapcore.AddSync(os.Stdout)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		ToZapLogLevel(logLevel),
	)

	mainLogger := zap.New(core)
	defer mainLogger.Sync()
	undo := zap.ReplaceGlobals(mainLogger)
	defer undo()

	Zap = mainLogger

	return Zap
}

func ToZapLogLevel(logLevel string) zapcore.Level {
	switch logLevel {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
