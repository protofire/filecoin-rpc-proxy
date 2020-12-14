package updater

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/utils"

	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/hashicorp/go-multierror"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/sirupsen/logrus"
)

type Updater struct {
	cacher            proxy.ResponseCacher
	logger            *logrus.Entry
	url               string
	token             string
	stopped           int32
	debugHTTPRequest  bool
	debugHTTPResponse bool
	batchSize         int
	concurrency       int
}

func New(
	cacher proxy.ResponseCacher,
	logger *logrus.Entry,
	url, token string,
	batchSize int,
	concurrency int,
	debugHTTPRequest bool,
	debugHTTPResponse bool,
) *Updater {
	u := &Updater{
		cacher:            cacher,
		logger:            logger,
		url:               url,
		token:             token,
		batchSize:         batchSize,
		concurrency:       concurrency,
		debugHTTPRequest:  debugHTTPRequest,
		debugHTTPResponse: debugHTTPResponse,
	}
	return u
}

func FromConfig(conf *config.Config, cacher proxy.ResponseCacher, logger *logrus.Entry) (*Updater, error) {
	token, err := auth.NewJWT(conf.JWT(), conf.JWTAlgorithm, conf.JWTPermissions)
	if err != nil {
		return nil, err
	}
	logger.Infof("Proxy token: %s", string(token))
	return New(
		cacher,
		logger,
		conf.ProxyURL,
		string(token),
		conf.RequestsBatchSize,
		conf.RequestsConcurrency,
		conf.DebugHTTPRequest,
		conf.DebugHTTPResponse,
	), nil
}

func (u *Updater) start(ctx context.Context, update func() error, period int) {

	ticker := time.NewTicker(time.Second * time.Duration(period))

	if err := update(); err != nil {
		u.logger.Errorf("cannot update requests: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := update(); err != nil {
				u.logger.Errorf("cannot update requests: %v", err)
			}
		}
	}
}

func (u *Updater) StartMethodUpdater(ctx context.Context, period int) {
	defer func() {
		u.logger.Info("Exiting methods updater...")
		atomic.AddInt32(&u.stopped, 1)
	}()
	u.start(ctx, u.updateMethods, period)
}

func (u *Updater) StartCacheUpdater(ctx context.Context, period int) {
	defer func() {
		u.logger.Info("Exiting cache updater...")
		atomic.AddInt32(&u.stopped, 1)
	}()
	u.start(ctx, u.updateCache, period)
}

func (u *Updater) StopWithTimeout(ctx context.Context, waitFor int) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if atomic.LoadInt32(&u.stopped) == int32(waitFor) {
				return true
			}
		}
	}
}

func (u *Updater) methodRequests() requests.RPCRequests {
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

func (u *Updater) cacheRequests() requests.RPCRequests {
	reqs := requests.RPCRequests{}
	counter := float64(1)
	cacheReqs, err := u.cacher.Cacher().Requests()
	if err != nil {
		u.logger.Errorf("Cannot get cache requests: %v", err)
		return reqs
	}
	for _, req := range cacheReqs {
		if !u.cacher.Matcher().IsUpdatable(req.Method) {
			continue
		}
		req.ID = counter
		reqs = append(reqs, req)
		counter++
	}
	return reqs
}

func (u *Updater) updateMethods() error {
	if reqs := u.methodRequests(); !reqs.IsEmpty() {
		return u.update(reqs)
	}
	return nil
}

func (u *Updater) updateCache() error {
	if reqs := u.cacheRequests(); !reqs.IsEmpty() {
		return u.update(reqs)
	}
	return nil
}

func (u *Updater) update(reqs requests.RPCRequests) error {
	if reqs.IsEmpty() {
		return nil
	}

	ch := make(chan struct{}, u.concurrency)
	errs := make(chan error, u.concurrency)

	go func() {

		var wg sync.WaitGroup

		defer func() {
			wg.Wait()
			defer close(ch)
			defer close(errs)
		}()

		for i := 0; i < len(reqs); i += u.batchSize {

			ch <- struct{}{}
			end := utils.Min(i+u.batchSize, len(reqs))
			wg.Add(1)

			go func(reqs requests.RPCRequests) {

				defer func() {
					wg.Done()
					<-ch
				}()

				u.logger.Infof("Updating %d cache records...", len(reqs))
				responses, _, err := requests.Request(u.url, u.token, u.logger, u.debugHTTPRequest, u.debugHTTPResponse, reqs)
				u.logger.Infof("Got %d responses", len(responses))
				if err != nil {
					errs <- err
					return
				}

				multiErr := &multierror.Error{}

				for _, resp := range responses {
					if resp.Error != nil {
						errs <- resp.Error
						continue
					}
					req, ok := reqs.FindByID(resp.ID)
					u.logger.Infof("Processing response ID %v...", resp.ID)
					if ok {
						u.logger.Infof("Setting response cache for request: %#v", req)
						if err := u.cacher.SetResponseCache(req, resp); err != nil {
							multiErr = multierror.Append(multiErr, err)
						}
					}
				}

				err = multiErr.ErrorOrNil()
				if err != nil {
					errs <- err
				}

			}(reqs[i:end])
		}

	}()

	multiErr := &multierror.Error{}

	for err := range errs {
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr.ErrorOrNil()

}
