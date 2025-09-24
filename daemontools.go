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
	"github.com/rstms/go-common"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed template/daemontools_run
var runTemplate string

//go:embed template/daemontools_log
var logTemplate string

type Daemontools struct {
	Name       string
	Username   string
	Uid        string
	Gid        string
	Executable string
	Args       string
	Dir        string
	LogFile    string
	service    string
	serviceBin string
}

func NewDaemontools(name string, serviceUser *user.User, runDir string, command string, args ...string) (CobraDaemon, error) {

	serviceDir := filepath.Join("/etc/service", name)
	_, basename := filepath.Split(command)
	args = append(args, "-L-")
	t := Daemontools{
		Name:       name,
		Username:   serviceUser.Username,
		Uid:        serviceUser.Uid,
		Gid:        serviceUser.Gid,
		Executable: command,
		Args:       strings.Join(args, " "),
		Dir:        runDir,
		service:    serviceDir,
		serviceBin: filepath.Join("/usr/local/bin", basename),
	}

	return &t, nil
}

func (d *Daemontools) templateData(template string) []byte {
	data := os.Expand(template, func(key string) string {
		switch key {
		case "TASK_NAME":
			return d.Name
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
	return []byte(data)
}

func (d *Daemontools) enable() error {
	downFile := filepath.Join(d.service, "down")
	if common.IsFile(downFile) {
		err := os.Remove(downFile)
		if err != nil {
			return fatal(err)
		}
	}
	return nil
}

func (d *Daemontools) disable() error {
	downFile := filepath.Join(d.service, "down")
	if !common.IsFile(downFile) {
		err := os.WriteFile(downFile, []byte{}, 0600)
		if err != nil {
			return fatal(err)
		}
	}
	return nil
}

func (d *Daemontools) Install() error {

	gid, err := strconv.Atoi(d.Gid)
	if err != nil {
		return fatal(err)
	}
	err = os.MkdirAll("/var/svc.d", 0755)
	if err != nil {
		return err
	}
	dir := filepath.Join("/var/svc.d", d.Name)
	err = os.MkdirAll(filepath.Join(dir, "log"), 0750)
	if err != nil {
		return err
	}
	err = os.Chown(dir, -1, gid)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(dir, "run"), d.templateData(runTemplate), 0700)
	if err != nil {
		return fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "log", "run"), d.templateData(logTemplate), 0700)
	if err != nil {
		return fatal(err)
	}
	ifp, err := os.Open(d.Executable)
	if err != nil {
		return fatal(err)
	}
	defer ifp.Close()
	ofp, err := os.OpenFile(d.serviceBin, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return fatal(err)
	}
	_, err = io.Copy(ofp, ifp)
	if err != nil {
		return fatal(err)
	}

	logdir := filepath.Join("/var/log", d.Name)
	if !common.IsDir(logdir) {
		err = os.Mkdir(logdir, 0770)
		if err != nil {
			return fatal(err)
		}
	}
	err = os.WriteFile(filepath.Join(dir, "down"), []byte{}, 0600)
	if err != nil {
		return fatal(err)
	}
	err = os.Symlink(dir, d.service)
	if err != nil {
		return fatal(err)
	}
	return nil
}

func (d *Daemontools) Delete() error {
	running, err := d.svstat(d.service)
	if err != nil {
		return fatal(err)
	}
	if running {
		err := d.Stop()
		if err != nil {
			return fatal(err)
		}
	}
	logRunning, err := d.svstat(filepath.Join(d.service, "log"))
	if err != nil {
		return fatal(err)
	}
	if logRunning {
		err = exec.Command("svc", "-d", filepath.Join(d.service, "log")).Run()
		if err != nil {
			return fatal(err)
		}
	}
	err = os.RemoveAll(d.service)
	if err != nil {
		return fatal(err)
	}
	err = os.RemoveAll(filepath.Join("/var/svc.d", d.Name))
	if err != nil {
		return fatal(err)
	}
	return nil
}

func (d *Daemontools) Start() error {
	err := d.enable()
	if err != nil {
		return fatal(err)
	}
	err = exec.Command("svc", "-u", d.service).Run()
	if err != nil {
		return fatal(err)
	}
	return nil
}

func (d *Daemontools) Stop() error {
	err := exec.Command("svc", "-d", d.service).Run()
	if err != nil {
		return fatal(err)
	}
	return nil
}

func (d *Daemontools) GetConfig() (string, error) {
	runData, err := os.ReadFile(filepath.Join(d.service, "run"))
	if err != nil {
		return "", fatal(err)
	}
	return string(runData), nil
}

func (d *Daemontools) Query() (bool, error) {
	running, err := d.svstat(d.service)
	if err != nil {
		return false, fatal(err)
	}
	return running, nil
}

func (d *Daemontools) svstat(serviceDir string) (bool, error) {
	stdout, err := exec.Command("svstat", serviceDir).Output()
	if err != nil {
		return false, fatal(err)
	}
	status := strings.TrimSpace(string(stdout))
	fields := strings.Fields(status)
	if len(fields) < 2 {
		return false, fatalf("unexpected svstat output: %s", status)
	}
	if fields[0] != serviceDir+":" {
		return false, fatalf("unexpected svstat dir output: %s", fields[0])
	}
	running := false
	if fields[1] == "up" {
		running = true
	}
	return running, nil
}
