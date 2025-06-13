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

package console

import (
	"fmt"
	"os"
	"runtime"

	"github.com/containerd/containerd/v2/pkg/stdio"
)

// NewPlatform returns the appropriate platform implementation
// On Linux: returns the real platform (imported from existing implementation)
// On non-Linux: returns the simple proxy platform
func NewPlatform() (stdio.Platform, error) {
	if runtime.GOOS == "linux" {
		// Use the real Linux platform implementation
		// This would be imported from the existing implementation
		// For now, return an error indicating Linux implementation needed
		return nil, fmt.Errorf("linux platform implementation not yet integrated")
	}

	// Use simple proxy for non-Linux systems
	address := os.Getenv("RUNV_SIMPLE_CONSOLE_ADDRESS")
	if address == "" {
		address = "unix:///var/run/runv-simple-console.sock"
	}

	return NewSimplePlatform(address)
}

// NewSimplePlatformWithAddress creates a simple proxy platform with specific address
func NewSimplePlatformWithAddress(address string) (stdio.Platform, error) {
	return NewSimplePlatform(address)
}
