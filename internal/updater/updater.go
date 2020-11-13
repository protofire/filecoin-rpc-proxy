package updater

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/hashicorp/go-multierror"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/sirupsen/logrus"
)

type Updater struct {
	cacher  proxy.ResponseCacher
	logger  *logrus.Entry
	url     string
	token   string
	stopped int32
}

func New(cacher proxy.ResponseCacher, logger *logrus.Entry, url, token string) *Updater {
	u := &Updater{
		cacher: cacher,
		logger: logger,
		url:    url,
		token:  token,
	}
	return u
}

func FromConfig(conf *config.Config, cacher proxy.ResponseCacher, logger *logrus.Entry) (*Updater, error) {
	token, err := auth.NewJWT(conf.JWTSecret, conf.JWTAlgorithm, []string{"admin"})
	if err != nil {
		return nil, err
	}
	logger.Infof("Proxy token: %s", string(token))
	return New(cacher, logger, conf.ProxyURL, string(token)), nil
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

func (u *Updater) requests() requests.RPCRequests {
	reqs := requests.RPCRequests{}
	counter := float64(1)
	for _, method := range u.cacher.Matcher().Methods() {
		reqs = append(reqs, requests.RPCRequest{
			JSONRPC: "2.0",
			ID:      counter,
			Method:  method.Name,
			Params:  method.Params,
		})
		counter++
	}
	return reqs
}

func (u *Updater) update() error {
	reqs := u.requests()
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
			err := u.cacher.SetResponseCache(req, resp)
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
			}
		}
	}

	return multiErr.ErrorOrNil()
}
