package registry

import (
	"github.com/elecbug/crawlp/internal/ieee"
	"github.com/elecbug/crawlp/internal/provider"
)

func NewDefaultRouter() *provider.Router {
	return provider.NewRouter(
		ieee.NewDownloader(),
	)
}
