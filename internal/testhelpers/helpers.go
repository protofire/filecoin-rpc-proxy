package testhelpers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

var (
	paramInCacheID           = 1
	token                    = "token"
	configParamsByIDTemplate = `
jwt_token: %s
jwt_secret: %s
proxy_url: %s
log_level: DEBUG
log_pretty_print: true
cache_methods:
- name: %s
  cache_by_params: true
  params_in_cache_by_id:
    - %s
`
)

func GetConfig(url string, method string) (*config.Config, error) {
	template := fmt.Sprintf(configParamsByIDTemplate, token, token, url, method, strconv.Itoa(paramInCacheID))
	return config.NewConfig(strings.NewReader(template))
}
