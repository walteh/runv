//go:build darwin || freebsd || netbsd || openbsd

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

package kqueue

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/containerd/console"
)

// this test is a copy from the epoll test in containerd/console
// it does not seem to run on darwin correctly, saving here in case we need it later
func ReferenceTestKqueueConsole(t *testing.T) {
	console, slavePath, err := console.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()

	iteration := 10

	cmd := exec.Command("sh", "-c", fmt.Sprintf("for x in `seq 1 %d`; do echo -n test; done", iteration))
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave

	kqueuer, err := NewKqueuer()
	if err != nil {
		t.Fatal(err)
	}
	kqueueConsole, err := kqueuer.Add(console)
	if err != nil {
		t.Fatal(err)
	}
	go kqueuer.Wait()

	var (
		b  bytes.Buffer
		wg sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		io.Copy(&b, kqueueConsole)
		wg.Done()
	}()

	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	slave.Close()
	if err := kqueueConsole.Shutdown(kqueuer.CloseConsole); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if err := kqueueConsole.Close(); err != nil {
		t.Fatal(err)
	}

	expectedOutput := ""
	for i := 0; i < iteration; i++ {
		expectedOutput += "test"
	}
	if out := b.String(); out != expectedOutput {
		t.Errorf("unexpected output %q", out)
	}

	// make sure multiple Close calls return os.ErrClosed after the first
	if err := kqueuer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := kqueuer.Close(); err != os.ErrClosed {
		t.Fatalf("unexpected error returned from second call to kqueuer.Close(): %v", err)
	}
}
