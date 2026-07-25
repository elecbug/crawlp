package registry

import (
	"github.com/elecbug/crawlp/internal/paper/acm"
	"github.com/elecbug/crawlp/internal/paper/ieee"
	"github.com/elecbug/crawlp/internal/provider"
)

func NewDefaultRouter() *provider.Router {
	return provider.NewRouter(
		ieee.NewDownloader(),
		acm.NewDownloader(),
	)
}
