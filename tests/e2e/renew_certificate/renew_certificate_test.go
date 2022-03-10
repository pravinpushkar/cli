/*
Copyright 2021 The Dapr Authors
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

// pre -
// 	1. init kubernetes cluster with a particular version

// test flow -1 -
// 	1. check dapr status -k to have no warning mesg and check the dapr expiry
// 	2. trigger renew command to generate and insert a fresh expiring cert
// 	3. check dapr status -k to have warning for expiring cert and check dapr expiry
// 	4. Check dapr control plane healthy

// test flow -2 -
// 	1. check dapr status -k to have no warning mesg and check the dapr expiry
// 	2. trigger renew command to insert a provided expiring cert
// 	3. check dapr status -k to have warning for expiring cert and check dapr expiry
// 	4. Check dapr control plane healthy

// test flow -2 -
// 	1. check dapr status -k to have no warning mesg and check the dapr expiry
// 	2. trigger renew command to generate a expiring cert with given root.key
// 	3. check dapr status -k to have warning for expiring cert and check dapr expiry
// 	4. Check dapr control plane healthy

package renew_certificate

import (
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
)

var currentVersionDetails = common.VersionDetails{
	RuntimeVersion:      "1.5.1",
	DashboardVersion:    "0.9.0",
	CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io"},
	ClusterRoles:        []string{"dapr-operator-admin", "dashboard-reader"},
	ClusterRoleBindings: []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"},
}

var installOpts = common.TestOptions{
	HAEnabled:             false,
	MTLSEnabled:           true,
	ApplyComponentChanges: true,
	CheckResourceExists: map[common.Resource]bool{
		common.CustomResourceDefs:  true,
		common.ClusterRoles:        true,
		common.ClusterRoleBindings: true,
	},
}

func TestCertificateRenewalWithGeneratedCertificate(t *testing.T) {
	common.EnsureUninstall(true)

	tests := []common.TestCase{}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in less than 30 days", common.RenewCertificate(currentVersionDetails)},
		{"crds exist " + currentVersionDetails.RuntimeVersion, common.CRDTest(currentVersionDetails, installOpts)},
		{"clusterroles exist " + currentVersionDetails.RuntimeVersion, common.ClusterRolesTest(currentVersionDetails, installOpts)},
		{"clusterrolebindings exist " + currentVersionDetails.RuntimeVersion, common.ClusterRoleBindingsTest(currentVersionDetails, installOpts)},
		{"check mtls " + currentVersionDetails.RuntimeVersion, common.MTLSTestOnInstallUpgrade(installOpts)},
		{"status check " + currentVersionDetails.RuntimeVersion, common.StatusTestOnInstallUpgrade(currentVersionDetails, installOpts)},
		{"warning message check " + currentVersionDetails.RuntimeVersion, common.CheckWarningMessageForCertExpiry(currentVersionDetails, installOpts)},
	}...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}
