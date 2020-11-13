package updater

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/hashicorp/go-multierror"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/sirupsen/logrus"

	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
)

type Updater struct {
	cache   cache.Cache
	matcher matcher.Matcher
	logger  *logrus.Entry
	url     string
	token   string
	stopped int32
}

func New(cache cache.Cache, matcher matcher.Matcher, logger *logrus.Entry, url, token string) *Updater {
	u := &Updater{
		cache:   cache,
		matcher: matcher,
		logger:  logger,
		url:     url,
		token:   token,
	}
	return u
}

func FromConfig(conf *config.Config, cache cache.Cache, matcher matcher.Matcher, logger *logrus.Entry) (*Updater, error) {
	token, err := auth.NewJWT(conf.JWTSecret, conf.JWTAlgorithm, []string{"admin"})
	if err != nil {
		return nil, err
	}
	logger.Infof("Proxy token: %s", string(token))
	return New(cache, matcher, logger, conf.ProxyURL, string(token)), nil
}

func (u *Updater) Start(ctx context.Context, period int) {
	defer func() {
		u.logger.Info("Exiting cache updater...")
		atomic.AddInt32(&u.stopped, 1)
	}()

	ticker := time.NewTicker(time.Second * time.Duration(period))

	if err := u.update(); err != nil {
		u.logger.Errorf("cannot update cached requests: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := u.update(); err != nil {
				u.logger.Errorf("cannot update cached requests: %v", err)
			}
		}
	}
}

func (u *Updater) StopWithTimeout(ctx context.Context) bool {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if atomic.LoadInt32(&u.stopped) == 1 {
				return true
			}
		}
	}
}

func (u *Updater) setResponseCache(req requests.RpcRequest, resp requests.RpcResponse) error {
	key := u.matcher.Key(req.Method, req.Params)
	if key == "" {
		return nil
	}
	return u.cache.Set(key, resp)
}

func (u *Updater) update() error {
	reqs := requests.RpcRequests{}
	counter := 1
	for _, method := range u.matcher.Methods() {
		reqs = append(reqs, requests.RpcRequest{
			JSONRPC: "2.0",
			ID:      counter,
			Method:  method.Name,
			Params:  method.Params,
		})
		counter++
	}
	if reqs.IsEmpty() {
		return nil
	}
	responses, _, err := requests.Request(u.url, u.token, reqs)
	if err != nil {
		return err
	}

	multiErr := &multierror.Error{}

	for _, resp := range responses {
		req, ok := reqs.FindByID(resp.ID)
		if ok {
			err := u.setResponseCache(req, resp)
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
			}
		}
	}

	return multiErr.ErrorOrNil()
}
