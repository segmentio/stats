package grpcstats

import (
	"net/http"
	"time"

	"google.golang.org/grpc"
)

type metrics struct {
	grpc struct {
		// err struct {
		// 	count int `metric:"count" type:"counter"`
		// } `metric:"error"`

		// req struct {
		// 	msg struct {
		// 		count       int `metric:"count"        type:"counter"`
		// 		headerSize  int `metric:"header.size"  type:"histogram"`
		// 		headerBytes int `metric:"header.bytes" type:"histogram"`
		// 		bodyBytes   int `metric:"body.bytes"   type:"histogram"`
		// 	} `metric:"message"`
		// }

		// resp struct {
		// 	rtt time.Duration `metric:"rtt.seconds" type:"histogram"`

		// 	msg struct {
		// 		count       int `metric:"count"        type:"counter"`
		// 		headerSize  int `metric:"header.size"  type:"histogram"`
		// 		headerBytes int `metric:"header.bytes" type:"histogram"`
		// 		bodyBytes   int `metric:"body.bytes"   type:"histogram"`
		// 	} `metric:"message"`
		// }

		// host   string `tag:"grpc_req_host"`
		method string `tag:"grpc_req_method"`
	} `metric:"grpc"`
}

func (m *metrics) observeRequest(info *grpc.UnaryServerInfo) {
	m.grpc.method = info.FullMethod
}

func (m *metrics) observeResponse(res *http.Response, op string, bodyLen int, rtt time.Duration) {
	// contentType, charset := contentType(res.Header)
	// contentEncoding := contentEncoding(res.Header)
	// upgrade := headerValue(res.Header, "Upgrade")
	// server := headerValue(res.Header, "Server")
	// bucket := responseStatusBucket(res.StatusCode)
	// status := statusCode(res.StatusCode)
	// transferEncoding := transferEncoding(res.TransferEncoding)

	// m.http.res.msg.count = 1
	// m.http.res.msg.headerSize = len(res.Header)
	// m.http.res.msg.headerBytes = responseHeaderLength(res)
	// m.http.res.msg.bodyBytes = bodyLen
	// m.http.res.rtt = rtt

	// m.http.res.operation = op
	// m.http.res.msgtype = "response"

	// m.http.res.contentCharset = charset
	// m.http.res.contentEncoding = contentEncoding
	// m.http.res.contentType = contentType
	// m.http.res.protocol = res.Proto
	// m.http.res.server = server
	// m.http.res.statusBucket = bucket
	// m.http.res.status = status
	// m.http.res.transferEncoding = transferEncoding
	// m.http.res.upgrade = upgrade
}

func (m *metrics) observeError() {

}
