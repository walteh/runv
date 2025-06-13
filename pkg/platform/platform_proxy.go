//go:build !linux
// +build !linux

/*
   Copyright The containerd Authors.

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

package platform

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/v2/pkg/stdio"
)

// NewPlatform returns a proxy platform for use with I/O operations on non-Linux systems
// This proxies all operations to a remote Linux server via ttrpc
func NewPlatform() (stdio.Platform, error) {
	// Get the proxy server address from environment variable
	address := os.Getenv("RUNV_CONSOLE_PROXY_ADDRESS")
	if address == "" {
		address = "/var/run/runv-console.sock" // Default socket path
	}

	return NewLinuxProxyPlatform(address)
}

// NewPlatformWithProxy creates a platform with a specific proxy address
func NewPlatformWithProxy(address string) (stdio.Platform, error) {
	if address == "" {
		return nil, fmt.Errorf("proxy address is required")
	}

	return NewLinuxProxyPlatform(address)
}
