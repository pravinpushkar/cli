//go:build e2e
// +build e2e

/*
Copyright 2022 The Dapr Authors
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

package standalone_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/dapr/cli/utils"
)

// getSocketCases return different unix socket paths for testing across Dapr commands.
// If the tests are being run on Windows, it returns an empty array.
func getSocketCases() []string {
	if runtime.GOOS == "windows" {
		return []string{""}
	} else {
		return []string{"", "/tmp"}
	}
}

// must is a helper function that executes a function and expects it to succeed.
func must(t *testing.T, f func(args ...string) (string, error), message string, fArgs ...string) {
	_, err := f(fArgs...)
	require.NoError(t, err, message)
}

// checkAndWriteFile writes content to file if it does not exist.
func checkAndWriteFile(filePath string, b []byte) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// #nosec G306
		if err = os.WriteFile(filePath, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// isSlimMode returns true if DAPR_E2E_INIT_SLIM is set to true.
func isSlimMode() bool {
	return os.Getenv("DAPR_E2E_INIT_SLIM") == "true"
}

// createSlimComponents creates default state store and pubsub components in path.
func createSlimComponents(path string) error {
	components := map[string]string{
		"pubsub.yaml": `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: pubsub
spec:
    type: pubsub.in-memory
    version: v1
    metadata: []`,
		"statestore.yaml": `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore
spec:
    type: state.in-memory
    version: v1
    metadata: []`,
	}

	for fileName, content := range components {
		fullPath := filepath.Join(path, fileName)
		if err := checkAndWriteFile(fullPath, []byte(content)); err != nil {
			return err
		}
	}

	return nil
}

// executeAgainstRunningDapr runs a function against a running Dapr instance.
// If Dapr or the App throws an error, the test is marked as failed.
func executeAgainstRunningDapr(t *testing.T, f func(), daprArgs ...string) {
	daprPath := common.GetDaprPath()

	cmd := exec.Command(daprPath, daprArgs...)
	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)

	cmd.Start()

	daprOutput := ""
	for scanner.Scan() {
		outputChunk := scanner.Text()
		t.Log(outputChunk)
		if strings.Contains(outputChunk, "You're up and running!") {
			f()
		}
		daprOutput += outputChunk
	}

	err := cmd.Wait()
	require.NoError(t, err, "dapr didn't exit cleanly")
	assert.NotContains(t, daprOutput, "The App process exited with error code: exit status", "Stop command should have been called before the app had a chance to exit")
	assert.Contains(t, daprOutput, "Exited Dapr successfully")
}

// ensureDaprInstallation ensures that Dapr is installed.
// If Dapr is not installed, a new installation is attempted.
func ensureDaprInstallation(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "failed to get user home directory")

	daprPath := filepath.Join(homeDir, ".dapr")
	if _, err = os.Stat(daprPath); err != nil {
		if os.IsNotExist(err) {
			installDapr(t)
		} else {
			// Some other error occurred.
			require.NoError(t, err, "failed to stat dapr installation")
		}
	}
	daprBinPath := filepath.Join(daprPath, "bin")
	if _, err = os.Stat(daprBinPath); err != nil {
		if os.IsNotExist(err) {
			installDapr(t)
		} else {
			// Some other error occurred.
			require.NoError(t, err, "failed to stat dapr installation")
		}
	}
	// Slim mode does not have any resources by default.
	// Install the resources required by the tests.
	if isSlimMode() {
		err = createSlimComponents(filepath.Join(daprPath, utils.DefaultResourcesDirName))
		require.NoError(t, err, "failed to create resources directory for slim mode")
	}
}

func containerRuntime() string {
	if daprContainerRuntime, ok := os.LookupEnv("CONTAINER_RUNTIME"); ok {
		return daprContainerRuntime
	}
	return ""
}

func installDapr(t *testing.T) {
	daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)
	args := []string{
		"--runtime-version", daprRuntimeVersion,
	}
	output, err := cmdInit(args...)
	require.NoError(t, err, "failed to install dapr:%v", output)
}

func uninstallDapr(uninstallArgs ...string) (string, error) {
	daprContainerRuntime := containerRuntime()

	// Add --container-runtime flag only if daprContainerRuntime is not empty, or overridden via args.
	// This is only valid for non-slim mode.
	if !isSlimMode() && daprContainerRuntime != "" && !utils.Contains(uninstallArgs, "--container-runtime") {
		uninstallArgs = append(uninstallArgs, "--container-runtime", daprContainerRuntime)
	}
	return spawn.Command(common.GetDaprPath(), uninstallArgs...)
}
