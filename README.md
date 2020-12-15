Overview
-----------------------------

`filecoin-rpc-proxy` is a caching solution for the [Filecoin JSON RPC API](https://docs.filecoin.io/reference/lotus-api/) to reduce endpoint load used by the [Glif Filecoin Hosted Endpoints service](https://docs.filecoin.io/build/hosted-lotus/).

The `config.yaml` is a configuration reference where you can specify any methods to cache and the caching parameters, such as cache TTL, auto-update, key caching parameters and other custom parameters.

(Current methods we cache include `ChainGetTipsetByHeight`, `ClientQueryAsk`, `StateCirculatingSupply`, `StateMarketDeals` which can have large responses and/or often requested by our users.)


JSON RPC Proxy with a cache
-----------------------------

#### Build and install

    make clean check test build
    make install

#### Docker

    make docker

#### Start

    ./proxy --help

#### Prometheus metrics

    proxy_request_duration_sum 1269
    proxy_request_duration_count 3
    proxy_requests 10
    proxy_requests_cached 7
    proxy_requests_error 3
    proxy_requests_method{method="Filecoin.StateCirculatingSupply"} 10
    proxy_requests_method_cached{method="Filecoin.StateCirculatingSupply"} 7
    proxy_requests_method_error{method="Filecoin.StateCirculatingSupply"} 3
