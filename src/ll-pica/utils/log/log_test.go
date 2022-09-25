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
	"testing"
)

// var logger *zap.SugaredLogger

// func init() {
// 	logger = InitLog()
// }

func TestNormal(t *testing.T) {
	Logger.Debugf("abc")
	Logger.Info("abc")
	LogSetLevel("debug")
	Logger.Debugf("abc")
	Logger.Info("abc")
}
