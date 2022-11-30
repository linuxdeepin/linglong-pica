/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */
package log

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var _callOnce sync.Once

var disableLogDebug string
var Logger *zap.SugaredLogger
var LogLevel zap.AtomicLevel

func InitLog() *zap.SugaredLogger {
	_callOnce.Do(func() {
		// writer, err := syslog.New(syslog.LOG_ERR|syslog.LOG_LOCAL0, "ll-pica")
		// if err != nil {
		// 	log.Fatalf("error creating syslog writer: %v", err)
		// }
		cfg := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		LogLevel = zap.NewAtomicLevel()
		// go build -ldflags '-X ll-pica/utils/log.disableLogDebug=yes'
		// fmt.Printf("disableLogDebug:%v", disableLogDebug)
		if disableLogDebug != "" {
			core := zapcore.NewCore(zapcore.NewJSONEncoder(cfg), os.Stdout, LogLevel)
			logger = zap.New(core)
			zap.ReplaceGlobals(logger)
		} else {
			LogLevel.SetLevel(zap.DebugLevel)
			core := zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), os.Stdout, LogLevel)
			logger = zap.New(core, zap.AddStacktrace(LogLevel))
			zap.ReplaceGlobals(logger)
		}

	})
	return logger.Sugar()
}

func LogSetLevel(l string) {
	LogLevel.SetLevel(zap.DebugLevel)
}

func init() {
	Logger = InitLog()
}
