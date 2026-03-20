package observability

import (
	"bug-report-service/internal/adapters/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, fields ...any)
	With(fields ...any) Logger
	Sync() error
}

type zapLogger struct {
	l *zap.SugaredLogger
}

func NewLogger(cfg config.Config) (Logger, error) {
	level := zapcore.InfoLevel
	_ = level.Set(cfg.Log.Level)

	zcfg := zap.NewProductionConfig()
	zcfg.Level = zap.NewAtomicLevelAt(level)
	zcfg.EncoderConfig.TimeKey = "ts"
	zcfg.EncoderConfig.MessageKey = "msg"
	zcfg.EncoderConfig.LevelKey = "level"

	base, err := zcfg.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{l: base.Sugar()}, nil
}

func (z *zapLogger) Info(msg string, fields ...any)  { z.l.Infow(msg, fields...) }
func (z *zapLogger) Error(msg string, fields ...any) { z.l.Errorw(msg, fields...) }
func (z *zapLogger) With(fields ...any) Logger       { return &zapLogger{l: z.l.With(fields...)} }
func (z *zapLogger) Sync() error                     { return z.l.Sync() }
