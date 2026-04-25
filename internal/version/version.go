package version

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X 'github.com/iucario/bili-danmu-go/internal/version.Version=v1.2.3'"
var Version = "dev"
