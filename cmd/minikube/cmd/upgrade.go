/*
Copyright 2024 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/blang/semver/v4"
	"github.com/spf13/cobra"
	"k8s.io/minikube/pkg/minikube/download"
	"k8s.io/minikube/pkg/minikube/exit"
	"k8s.io/minikube/pkg/minikube/notify"
	"k8s.io/minikube/pkg/minikube/out"
	"k8s.io/minikube/pkg/minikube/reason"
	"k8s.io/minikube/pkg/minikube/style"
	"k8s.io/minikube/pkg/util"
	"k8s.io/minikube/pkg/version"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade minikube to the latest version",
	Long:  `Upgrade minikube to the latest version`,
	Run: func(_ *cobra.Command, _ []string) {
		currentVersion := version.GetVersion()
		out.Step(style.Usage, "You're currently using version: {{.version}}", out.V{"version": currentVersion})
		currentSemVer, err := semver.ParseTolerant(version.GetVersion())
		if err != nil {
			exit.Error(reason.InetVersionUnavailable, "Unable to parse current version", err)
		}

		out.Step(style.HealthCheck, "Checking for the latest version ...")
		latestVersion, err := getLatestMinikubeVersion()
		if err != nil {
			exit.Error(reason.InetVersionUnavailable, "Unable to fetch latest version info", err)
		}
		latestSemVer, err := semver.ParseTolerant(latestVersion)
		if err != nil {
			exit.Error(reason.InetVersionUnavailable, "Unable to parse latest version", err)
		}

		if currentSemVer.GTE(latestSemVer) {
			out.Styled(style.Verifying, "You're already using the latest version of minikube.")
			return
		}
		out.Styled(style.Fileserver, "A new version of minikube is available: {{.latestVersion}}\n",
			out.V{"latestVersion": latestVersion})

		newBinary, err := downloadMinikubeToTemp(latestVersion)
		if err != nil {
			exit.Error(reason.UpgradeMinikubeFailed, "Unable to download minikube", err)
		}

		if err := replaceMinikubeBinary(newBinary); err != nil {
			exit.Error(reason.UpgradeMinikubeFailed, "Unable to replace old minikube", err)
		}
		out.Styled(style.Success, "upgraded successfully!")
	},
}

func downloadMinikubeToTemp(version string) (string, error) {
	dst := filepath.Join(os.TempDir(), "minikube")
	if err := download.Minikube(version, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func replaceMinikubeBinary(new string) error {
	old, err := util.GetBinaryExecutePath()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		baseName := filepath.Base(old)
		// On Windows, we cannot directly overwrite a running executable.
		// Therefore, we rename the old binary to a temporary name before updating it.
		err = os.Rename(old, baseName+".old")
		if err != nil {
			return err
		}
		return os.Rename(new, old)
	case "linux", "darwin":
		oldFileInfo, err := os.Stat(old)
		if err != nil {
			return err
		}
		err = os.Chmod(new, oldFileInfo.Mode())
		if err != nil {
			return err
		}
		err = os.Rename(new, old)
		if err != nil && os.IsPermission(err) {
			out.Styled(style.Warning, "Please provide root privileges to continue.")
			cmd := exec.Command("sudo", "mv", new, old)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			return cmd.Run()
		}
		return err
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// getLatestMinikubeVersion returns the latest minikube version
func getLatestMinikubeVersion() (string, error) {
	r, err := notify.AllVersionsFromURL(notify.GithubMinikubeReleasesURL)
	if err != nil {
		return "", err
	}
	if len(r.Releases) < 1 {
		return "", fmt.Errorf("update server returned an empty list")
	}
	return r.Releases[0].Name, nil
}
