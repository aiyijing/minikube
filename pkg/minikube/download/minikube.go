package download

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/minikube/pkg/minikube/notify"
	"k8s.io/minikube/pkg/minikube/out"
	"k8s.io/minikube/pkg/minikube/style"
)

func minikubeWithOSArchAndChecksumURL(version string) string {
	base := notify.DownloadURL(version, runtime.GOOS, runtime.GOARCH)
	return fmt.Sprintf("%s?checksum=file:%s.sha256", base, base)
}

func Minikube(version, destination string) error {
	out.Step(style.FileDownload, "Downloading minikube {{.version}}:", out.V{"version": version})
	archURL := minikubeWithOSArchAndChecksumURL(version)
	if err := download(archURL, destination); err != nil {
		klog.Infof("failed to download minikube: %v.", err)
		return errors.Wrap(err, "download")
	}
	return nil
}
