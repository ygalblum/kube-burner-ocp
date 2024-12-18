// Copyright 2022 The Kube-burner Authors.
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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kube-burner/kube-burner/pkg/workloads"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const (
	sshKeyFileName = "ssh"
)

// NewCapacityBenchmark holds the capacity-benchmark workload
func NewCapacityBenchmark(wh *workloads.WorkloadHelper) *cobra.Command {
	var sshKeyPairPath string
	var maxIterations int
	var vmsPerIteration int
	// var metricsProfiles []string
	var rc int
	cmd := &cobra.Command{
		Use:          "capacity-benchmark",
		Short:        "Runs capacity-benchmark workload",
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			if sshKeyPairPath == "" {
				tempDir, err := os.MkdirTemp("", "kube-burner-capacity-benchmark-*")
				if err != nil {
					log.Fatalln("Error creating temporary directory:", err)
				}
				sshKeyPairPath = tempDir
			}
			privateKeyPath := path.Join(sshKeyPairPath, sshKeyFileName)
			publicKeyPath := path.Join(sshKeyPairPath, strings.Join([]string{sshKeyFileName, "pub"}, "."))
			log.Infof("Saving SSH keys to [%s]", sshKeyPairPath)
			if err := generateSSHKeyPair(privateKeyPath, publicKeyPath); err != nil {
				log.Fatalf("Failed to generate SSH keys for the test - %v", err)
			}
			os.Setenv("privateKey", privateKeyPath)
			os.Setenv("publicKey", publicKeyPath)
			os.Setenv("vmCount", fmt.Sprint(vmsPerIteration))
		},
		Run: func(cmd *cobra.Command, args []string) {
			// setMetrics(cmd, metricsProfiles)

			counter := 0
			for {
				os.Setenv("counter", fmt.Sprint(counter))
				rc = wh.Run(cmd.Name())
				if rc != 0 {
					log.Infof("Capacity failed in loop #%d", counter)
					break
				}
				counter += 1
				if maxIterations > 0 && counter >= maxIterations {
					log.Infof("Reached maxIterations [%d]", maxIterations)
					break
				}
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			os.Exit(rc)
		},
	}
	cmd.Flags().StringVar(&sshKeyPairPath, "ssh-key-path", "", "Path to save the generarated SSH keys - default to a temporary location")
	cmd.Flags().IntVar(&maxIterations, "max-iterations", 0, "Maximum times to run the test sequence. Default - run until failure (0)")
	cmd.Flags().IntVar(&vmsPerIteration, "vms", 1, "Number of VMs to test in each iteration")
	// cmd.Flags().StringSliceVar(&metricsProfiles, "metrics-profile", []string{"metrics-aggregated.yml"}, "Comma separated list of metrics profiles to use")
	// cmd.MarkFlagRequired("iterations")
	return cmd
}

// GenerateSSHKeyPair generates an SSH key pair and saves them to the specified files
func generateSSHKeyPair(privateKeyPath, publicKeyPath string) error {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode the private key to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Write the private key to a file
	err = os.WriteFile(privateKeyPath, privateKeyPEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	// Generate the public key in OpenSSH authorized_keys format
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	// Write the public key to a file
	err = os.WriteFile(publicKeyPath, publicKeyBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write public key to file: %w", err)
	}

	return nil
}
