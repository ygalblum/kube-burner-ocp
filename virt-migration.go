// Copyright 2024 The Kube-burner Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ocp

import (
	"fmt"
	"os"

	"github.com/cloud-bulldozer/go-commons/v2/ssh"
	"github.com/cloud-bulldozer/go-commons/v2/virtctl"
	"github.com/kube-burner/kube-burner/pkg/workloads"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const (
	virtMigrationSSHKeyFileName = "ssh"
	virtMigrationTmpDirPattern  = "kube-burner-virt-migration-*"
	virtMigrationTestName       = "virt-migration"
)

// Returns virt-density workload
func NewVirtMigration(wh *workloads.WorkloadHelper) *cobra.Command {
	var storageClassName string
	var sshKeyPairPath string
	var iterations int
	var vmsPerIteration int
	var testNamespace string
	var dataVolumeCount int
	var metricsProfiles []string

	var rc int
	cmd := &cobra.Command{
		Use:          virtMigrationTestName,
		Short:        fmt.Sprintf("Runs %s workload", virtMigrationTestName),
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			if !virtctl.IsInstalled() {
				log.Fatalf("Failed to run virtctl. Check that it is installed, in PATH and working")
			}

			storageClassName, _ = getStorageAndSnapshotClasses(storageClassName, false, true)
		},
		Run: func(cmd *cobra.Command, args []string) {
			privateKeyPath, publicKeyPath, err := ssh.GenerateSSHKeyPair(sshKeyPairPath, virtMigrationTmpDirPattern, virtMigrationSSHKeyFileName)
			if err != nil {
				log.Fatalf("Failed to generate SSH keys for the test - %v", err)
			}

			additionalVars := map[string]any{
				"privateKey":           privateKeyPath,
				"publicKey":            publicKeyPath,
				"storageClassName":     storageClassName,
				"testNamespace":        testNamespace,
				"vmCreateIterations":   iterations,
				"vmCreatePerIteration": vmsPerIteration,
				"dataVolumeCounters":   generateLoopCounterSlice(dataVolumeCount, 1),
				"workerNodeName":       "worker-0",
			}

			setMetrics(cmd, metricsProfiles)
			rc = wh.RunWithAdditionalVars(cmd.Name(), additionalVars, nil)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			os.Exit(rc)
		},
	}
	cmd.Flags().StringVar(&storageClassName, "storage-class", "", "Name of the Storage Class to test")
	cmd.Flags().StringVar(&sshKeyPairPath, "ssh-key-path", "", "Path to save the generarated SSH keys")
	cmd.Flags().IntVar(&iterations, "iterations", 2, "Number of start iterations. The total number of VMs is iterations*iteration-vms")
	cmd.Flags().IntVar(&vmsPerIteration, "iteration-vms", 10, "How many VMs to start simultaneously. The total number of VMs is iterations*iteration-vms")
	cmd.Flags().StringVarP(&testNamespace, "namespace", "n", virtMigrationTestName, "Base name for the namespace to run the test in")
	cmd.Flags().IntVar(&dataVolumeCount, "data-volume-count", 9, "Number of data volumes per VM")
	cmd.Flags().StringSliceVar(&metricsProfiles, "metrics-profile", []string{"metrics.yml"}, "Comma separated list of metrics profiles to use")
	return cmd
}
