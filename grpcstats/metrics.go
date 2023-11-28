package grpcstats

type metrics struct {
	grpc struct {
		err struct {
		} `metric:"error"`

		req struct {
		}

		resp struct {
		}

		host     string `tag:"grpc_req_host"`
		method   string `tag:"grpc_req_method"`
		path     string `tag:"grpc_req_path"`
		protocol string `tag:"grpc_req_protocol"`
	} `metric:"grpc"`
}

func (m *metrics) observeRequest() {

}

func (m *metrics) observeResponse() {

}

func (m *metrics) observeError() {

}
