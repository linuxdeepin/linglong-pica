package base

import (
	. "ll-pica/utils/log"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}
