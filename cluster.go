// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pluton

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/mantle/platform"
	"github.com/coreos/mantle/util"
)

// Cluster represents an interface to test kubernetes clusters The creation is
// usually implemented by a function that builds the Cluster from a kola
// TestCluster from the 'spawn' subpackage. Tests may be aware of the
// implementor function since not all clusters are expected to have the same
// components nor properties.
type Cluster struct {
	Masters []platform.Machine
	Workers []platform.Machine

	m Manager
}

func NewCluster(m Manager, masters, workers []platform.Machine) *Cluster {
	return &Cluster{
		Masters: masters,
		Workers: workers,
		m:       m,
	}
}

// Kubectl will run kubectl from /home/core on the Master Machine
func (c *Cluster) Kubectl(cmd string) (string, error) {
	client, err := c.Masters[0].SSHClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout = bytes.NewBuffer(nil)
	var stderr = bytes.NewBuffer(nil)
	session.Stderr = stderr
	session.Stdout = stdout

	err = session.Run("sudo ./kubectl --kubeconfig=/etc/kubernetes/kubeconfig " + cmd)
	if err != nil {
		return "", fmt.Errorf("kubectl: %s", stderr)
	}
	return stdout.String(), nil
}

// AddMasters creates new master nodes for a Cluster and blocks until ready.
func (c *Cluster) AddMasters(n int) error {
	nodes, err := c.m.AddMasters(n)
	if err != nil {
		return err
	}

	c.Masters = append(c.Masters, nodes...)

	if err := c.NodeCheck(12); err != nil {
		return err
	}
	return nil
}

// NodeCheck will parse kubectl output to ensure all nodes in Cluster are
// registered. Set retry for max amount of retries to attempt before erroring.
func (c *Cluster) NodeCheck(retryAttempts int) error {
	f := func() error {
		b, err := c.Masters[0].SSH("./kubectl get nodes")
		if err != nil {
			return err
		}

		// parse kubectl output for IPs
		addrMap := map[string]struct{}{}
		for _, line := range strings.Split(string(b), "\n")[1:] {
			addrMap[strings.SplitN(line, " ", 2)[0]] = struct{}{}
		}

		nodes := append(c.Workers, c.Masters...)

		if len(addrMap) != len(nodes) {
			return fmt.Errorf("cannot detect all nodes in kubectl output \n%v\n%v", addrMap, nodes)
		}
		for _, node := range nodes {
			if _, ok := addrMap[node.PrivateIP()]; !ok {
				return fmt.Errorf("node IP missing from kubectl get nodes")
			}
		}
		return nil
	}

	if err := util.Retry(retryAttempts, 10*time.Second, f); err != nil {
		return err
	}
	return nil
}

// SSH is just a convenience function for running SSH commands when you don't
// care which machine the command runs on. The current implementation chooses
// the first master node. The signature is slightly different then the machine
// SSH command and doesn't automatically print stderr. I expect in the future
// that this will be more unified with the Machine.SSH signature, but for now
// this is useful to silence all the retry loops from clogging up the test
// results while giving the option to deal with stderr.
func (c *Cluster) SSH(cmd string) (stdout, stderr []byte, err error) {
	client, err := c.Masters[0].SSHClient()
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer session.Close()

	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)
	session.Stdout = outBuf
	session.Stderr = errBuf

	err = session.Run(cmd)

	stdout = bytes.TrimSpace(outBuf.Bytes())
	stderr = bytes.TrimSpace(errBuf.Bytes())

	return stdout, stderr, err
}
