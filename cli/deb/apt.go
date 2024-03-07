/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"fmt"
	"strings"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/log"
)

// 调用 apt-cache show 命令

func AptShow(path string) (string, error) {
	ret, msg, err := comm.ExecAndWait(10, "sh", "-c", fmt.Sprintf("apt-cache show %s", path))
	if err != nil {
		log.Logger.Warnf("apt-cache show error: msg: %s err: %s", msg, err)
		return ret, err
	}
	return ret, nil
}

func AptDownload(name string) string {
	ret, _, err := comm.ExecAndWait(10, "apt", "download", name, "-y", "--print-uris")
	if err != nil {
		log.Logger.Errorf("apt download error %s", err)
		return ""
	}
	url := strings.Split(ret, " ")[0]
	url = strings.Replace(url, "'", "", 2)
	return url
}
