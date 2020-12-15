Json RPC proxy with a cache
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
