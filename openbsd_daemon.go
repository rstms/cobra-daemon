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
	_ "embed"
	"fmt"
	"github.com/rstms/go-common"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

//go:embed template/rcfile
var rcTemplate string

type RCDaemon struct {
	Name       string
	Username   string
	Uid        string
	Executable string
	Args       string
	Dir        string
	LogFile    string
	serviceBin string
}

func NewRCDaemon(name string, daemonUser *user.User, runDir string, command string, args ...string) (CobraDaemon, error) {

	logFile := filepath.Join("/var/log", name)
	_, basename := filepath.Split(command)

	t := RCDaemon{
		Name:       name,
		Username:   daemonUser.Username,
		Uid:        daemonUser.Uid,
		Executable: command,
		Args:       strings.Join(append(args, "-L", logFile), " "),
		Dir:        runDir,
		LogFile:    logFile,
		serviceBin: filepath.Join("/usr/local/bin", basename),
	}

	return &t, nil
}

func (d *RCDaemon) Install() error {

	rcData := os.Expand(rcTemplate, func(key string) string {
		switch key {
		case "TASK_USER":
			return d.Username
		case "TASK_UID":
			return d.Uid
		case "TASK_BIN":
			return d.serviceBin
		case "TASK_ARGS":
			return d.Args
		case "TASK_DIR":
			return d.Dir
		}
		return "${" + key + "}"
	})
	if d.Executable != d.serviceBin {
		src, err := os.Open(d.Executable)
		if err != nil {
			return common.Fatal(err)
		}
		defer src.Close()
		dst, err := os.OpenFile(d.serviceBin, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			return common.Fatal(err)
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		if err != nil {
			return common.Fatal(err)
		}
	}
	rcFile := filepath.Join("/etc/rc.d", d.Name)
	err := os.WriteFile(rcFile, []byte(rcData), 0700)
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (d *RCDaemon) rcctl(command string) error {
	cmd := exec.Command("rcctl", command, d.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("%v\n", cmd)
	return cmd.Run()
}

func (d *RCDaemon) Delete() error {
	err := d.rcctl("stop")
	if err != nil {
		return common.Fatal(err)
	}
	err = d.rcctl("disable")
	if err != nil {
		return common.Fatal(err)
	}
	err = os.Remove(filepath.Join("/etc/rc.d", d.Name))
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (d *RCDaemon) Start() error {
	err := d.rcctl("enable")
	if err != nil {
		return common.Fatal(err)
	}
	err = d.rcctl("start")
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (d *RCDaemon) Stop() error {
	err := d.rcctl("stop")
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (d *RCDaemon) GetConfig() (string, error) {
	config, err := exec.Command("rcctl", "get", d.Name).Output()
	if err != nil {
		return "", common.Fatal(err)
	}
	return string(config), nil
}

func (d *RCDaemon) Query() (bool, error) {
	cmd := exec.Command("rcctl", "check", d.Name)
	err := cmd.Run()
	switch err.(type) {
	case nil:
	case *exec.ExitError:
	default:
		return false, common.Fatal(err)
	}
	exitCode := cmd.ProcessState.ExitCode()
	return exitCode == 0, nil
}
