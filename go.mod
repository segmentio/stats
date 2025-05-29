module github.com/segmentio/stats/v5

go 1.23.0

require (
	github.com/mdlayher/taskstats v0.0.0-20241219020249-a291fa5f5a69
	github.com/segmentio/encoding v0.4.1
	github.com/segmentio/fasthash v1.0.3
	github.com/segmentio/vpcinfo v0.2.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/sync v0.14.0
	golang.org/x/sys v0.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	golang.org/x/net v0.40.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// this version contains an error that truncates metric, tag, and field names
// once a buffer exceeds 250 characters
retract v5.6.0
