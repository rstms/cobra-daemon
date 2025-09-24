/*
Copyright © 2024 Matt Krueger <mkrueger@rstms.net>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package daemon

import (
	"github.com/rstms/go-common"
	"os/user"
	"runtime"
)

const Version = "0.0.13"

type CobraDaemon interface {
	Install() error
	Delete() error
	Start() error
	Stop() error
	GetConfig() (string, error)
	Query() (bool, error)
}

func NewDaemon(name, username, dir, command string, args ...string) (CobraDaemon, error) {

	taskUser, err := user.Current()
	if err != nil {
		return nil, common.Fatal(err)
	}
	if username != "" {
		taskUser, err = user.Lookup(username)
		if err != nil {
			return nil, common.Fatal(err)
		}
	}

	taskDir := dir
	if taskDir == "" {
		taskDir = taskUser.HomeDir
	}

	if !common.IsDir(taskDir) {
		return nil, common.Fatalf("not directory: %s", taskDir)
	}

	var daemon CobraDaemon
	switch runtime.GOOS {
	case "windows":
		daemon, err = NewWindowsTask(name, taskUser, taskDir, command, args...)
		if err != nil {
			return nil, common.Fatal(err)
		}
	case "openbsd":
		daemon, err = NewRCDaemon(name, taskUser, taskDir, command, args...)
		if err != nil {
			return nil, common.Fatal(err)
		}
	case "linux":
		daemon, err = NewDaemontools(name, taskUser, taskDir, command, args...)
		if err != nil {
			return nil, common.Fatal(err)
		}
	default:
		return nil, common.Fatalf("unsuported os: %s", runtime.GOOS)
	}
	return daemon, nil
}
