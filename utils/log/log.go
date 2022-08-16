package log

import (
	"sync"

	"go.uber.org/zap"
)

var _logger *zap.SugaredLogger
var _callOnce sync.Once

func InitLog() *zap.SugaredLogger {
	//_callOnce = sync.Once()
	_callOnce.Do(func() {
		logger2, _ := zap.NewDevelopment()
		//logger2, _ := zap.Config{Level: zap.NewAtomicLevelAt(zap.DebugLevel)}.Build()
		zap.ReplaceGlobals(logger2)
		_logger = zap.S()
	})

	return _logger

}

func init() {
	_logger = InitLog()
	//_logger.Debug("log init")
}
