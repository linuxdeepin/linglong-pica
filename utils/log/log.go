/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
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
