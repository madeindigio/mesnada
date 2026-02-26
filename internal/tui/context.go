package tui

import (
	"github.com/sevir/mesnada/internal/config"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
)

type TUIContext = tuictx.Context

func newContext(cfg *config.Config) *TUIContext {
	return tuictx.New(cfg)
}
