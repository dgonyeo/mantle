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

package misc

import (
	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/kola/register"
	"github.com/coreos/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         RootOnRaid,
		ClusterSize: 1,
		Name:        "coreos.misc.root-on-raid",
		// This will only work on qemu, since it's overwriting
		// /usr/share/oem/grub.cfg
		Platforms: []string{"qemu"},
		UserData: conf.ContainerLinuxConfig(`storage:
  raid:
    - name: "ROOT"
      level: "raid1"
      devices:
        - "/dev/disk/by-partlabel/ROOT"
        - "/dev/disk/by-partlabel/USR-B"
  filesystems:
    - name: "ROOT"
      mount:
        device: "/dev/md127"
        format: "ext4"
        create:
          options:
            - "-L"
            - "ROOT"
    - name: "OEM"
      mount:
        device: "/dev/vda6"
        format: "ext4"
  files:
    - filesystem: "OEM"
      path: "/grub.cfg"
      contents:
        inline: |
            set linux_append="rd.auto"`),
	})
}

func RootOnRaid(c cluster.TestCluster) {
	m := c.Machines()[0]

	// make sure things in /etc exist
	_, err := m.SSH("cat /etc/hosts")
	if err != nil {
		c.Fatalf("could not get hosts file: %v", err)
	}

	// reboot it to make sure it comes up again
	err = m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	// make sure things in /etc exist
	_, err = m.SSH("cat /etc/hosts")
	if err != nil {
		c.Fatalf("could not get hosts file: %v", err)
	}
}
