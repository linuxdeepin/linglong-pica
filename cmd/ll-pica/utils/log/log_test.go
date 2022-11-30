/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package log

import (
	"testing"
)

func TestNormal(t *testing.T) {
	Logger.Debugf("abc")
	Logger.Info("abc")
	LogSetLevel("debug")
	Logger.Debugf("abc")
	Logger.Info("abc")
}
