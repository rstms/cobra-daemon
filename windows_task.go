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
	"bytes"
	_ "embed"
	"fmt"
	"github.com/rstms/go-common"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

//go:embed template/task.xml
var xmlTemplate string

type WindowsTask struct {
	Name       string
	Username   string
	Uid        string
	Executable string
	Args       string
	Dir        string
	LogFile    string
}

func NewWindowsTask(taskName string, taskUser *user.User, taskDir string, taskCommand string, taskArgs ...string) (CobraDaemon, error) {

	logDir := filepath.Join(taskUser.HomeDir, "logs")
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		return nil, common.Fatal(err)
	}
	logFile := filepath.Join(logDir, taskName+"-task.log")
	taskArgs = append(taskArgs, "-L", logFile)
	t := WindowsTask{
		Name:       taskName,
		Username:   taskUser.Username,
		Uid:        taskUser.Uid,
		Executable: taskCommand,
		Args:       strings.Join(taskArgs, " "),
		Dir:        taskDir,
		LogFile:    logFile,
	}

	return &t, nil
}

func (t *WindowsTask) taskScheduler(cmd string, args ...string) (int, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	taskArgs := append([]string{"/" + cmd, "/TN", t.Name}, args...)
	command := exec.Command("schtasks.exe", taskArgs...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := command.ProcessState.ExitCode()
	estr := strings.TrimSpace(stderr.String())
	ostr := strings.TrimSpace(stdout.String())
	if common.ViperGetBool("verbose") {
		fmt.Printf("%s\n", ostr)
	}
	if err != nil {
		if estr != "" {
			return exitCode, "", fmt.Errorf("%v", estr)
		}
		return exitCode, "", err
	}
	return exitCode, ostr, nil
}

func (t *WindowsTask) Install() error {

	xmlData := os.Expand(xmlTemplate, func(key string) string {
		switch key {
		case "TASK_USER":
			return t.Username
		case "TASK_UID":
			return t.Uid
		case "TASK_BIN":
			return t.Executable
		case "TASK_ARGS":
			return t.Args
		case "TASK_DIR":
			return t.Dir
		}
		return "UNEXPANDED_XML_PARAM_" + key
	})

	tempDir, err := os.MkdirTemp("", "task-create-*")
	if err != nil {
		return common.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	xmlFile := filepath.Join(tempDir, "task.xml")
	err = os.WriteFile(filepath.Join(tempDir, "task.xml"), []byte(xmlData), 0600)
	if err != nil {
		return common.Fatal(err)
	}
	createArgs := []string{
		"/XML", xmlFile,
	}
	_, _, err = t.taskScheduler("CREATE", createArgs...)
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (t *WindowsTask) Delete() error {
	_, _, err := t.taskScheduler("END")
	if err != nil {
		return common.Fatal(err)
	}
	_, _, err = t.taskScheduler("DELETE", "/F")
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (t *WindowsTask) Start() error {
	_, _, err := t.taskScheduler("RUN")
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (t *WindowsTask) Stop() error {
	_, _, err := t.taskScheduler("END")
	if err != nil {
		return common.Fatal(err)
	}
	return nil
}

func (t *WindowsTask) GetConfig() (string, error) {
	_, out, err := t.taskScheduler("QUERY", "/XML", "ONE")
	if err != nil {
		return "", common.Fatal(err)
	}
	return out, nil
}

func (t *WindowsTask) Query() (bool, error) {

	_, stdout, err := t.taskScheduler("QUERY", "/FO", "csv", "/NH")
	if err != nil {
		return false, common.Fatal(err)
	}
	fields := []string{}
	lines := strings.Split(stdout, "\n")
	if len(lines) == 1 {
		fields = strings.Split(lines[0], ",")
	}
	if len(lines) != 1 || len(fields) != 3 {
		return false, common.Fatalf("unexpected output: %v", stdout)
	}
	taskName := `"\` + t.Name + `"`
	if fields[0] != taskName {
		return false, common.Fatalf("unexpected task name: %s", fields[0])
	}
	if fields[2] == `"Running"` {
		return true, nil
	}
	return false, nil
}
