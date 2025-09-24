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
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type DaemonProcess interface {
	Start() error
	Stop() error
	Run() error
}

var factory func() (DaemonProcess, error)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "daemon commands",
	Long: `

subcommands for netboot/winexec daemon management  

OS       | Utility      | Config File
-------- | ------------ | --------------------- 
OpenBSD  | rcctl        | /etc/rc.d/NAME
Linux    | daemontools  | /etc/service/NAME
Windows  | schtasks.exe | internal XML config

`,
}

func initDaemon() Daemon {
	var command string
	switch {
	case ViperGetBool("daemon.winexec"):
		command = "winexec"
	case ViperGetBool("daemon.netboot"):
		command = "netboot"
	}
	if command == "" {
		cobra.CheckErr(Fatalf("missing daemon command"))
	}
	ViperSetDefault("daemon.name", command)
	name := ViperGetString("daemon.name")
	user := ViperGetString("daemon.user")
	dir := ViperGetString("daemon.dir")
	executable, err := os.Executable()
	cobra.CheckErr(err)
	d, err := NewDaemon(name, user, dir, filepath.Clean(executable), command, "server")
	cobra.CheckErr(err)
	return d
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install daemon",
	Long: `
install netboot/winexec daemon config
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		_, err := d.GetConfig()
		if err == nil && ViperGetBool("force") {
			err := d.Delete()
			cobra.CheckErr(err)
		}
		err = d.Install()
		cobra.CheckErr(err)
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "start daemon",
	Long: `
start netboot/winexec daemon
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		err := d.Start()
		cobra.CheckErr(err)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop daemon",
	Long: `
stop netboot/winexec daemon
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		err := d.Stop()
		cobra.CheckErr(err)
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart daemon",
	Long: `
restart netboot/winexec daemon
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		err := d.Stop()
		cobra.CheckErr(err)
		err = d.Start()
		cobra.CheckErr(err)
	},
}

var daemonDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete daemon",
	Long: `
delete netboot/winexec daemon config
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		err := d.Delete()
		cobra.CheckErr(err)
	},
}

var daemonShowCmd = &cobra.Command{
	Use:   "show",
	Short: "show daemon config",
	Long: `
show netboot/winexec daemon config
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		out, err := d.GetConfig()
		cobra.CheckErr(err)
		fmt.Println(out)
	},
}

var daemonQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "query daemon status",
	Long: `
return 0 if netboot/winexec daemon is running, 1 if not
`,
	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		running, err := d.Query()
		if !ViperGetBool("quiet") {
			cobra.CheckErr(err)
		}
		if running {
			os.Exit(0)
		}
		os.Exit(1)
	},
}

func AddDaemonCommand(rootCmd *cobra.Command, init func() (DaemonProcess, error)) {
	factory = init
	CobraAddCommand(rootCmd, rootCmd, daemonCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonInstallCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonStartCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonStopCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonRestartCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonDeleteCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonShowCmd)
	CobraAddCommand(rootCmd, daemonCmd, daemonQueryCmd)
	OptionString(daemonCmd, "name", "", "", "daemon name")
	OptionString(daemonCmd, "user", "", "", "run as username")
	OptionString(daemonCmd, "dir", "", "", "run directory")
	OptionSwitch(daemonCmd, "winexec", "", "select winexec daemon")
	OptionSwitch(daemonCmd, "netboot", "", "select winexec daemon")
	daemonCmd.MarkFlagsMutuallyExclusive("netboot", "winexec")
	daemonCmd.MarkFlagsOneRequired("netboot", "winexec")

}
