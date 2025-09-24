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
	"github.com/rstms/cobra-daemon/common"
	"github.com/spf13/cobra"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var daemonArgs []string

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "daemon commands",
	Long: `

subcommands for daemon management  

OS       | Utility      | Config File
-------- | ------------ | --------------------- 
OpenBSD  | rcctl        | /etc/rc.d/NAME
Linux    | daemontools  | /etc/service/NAME
Windows  | schtasks.exe | internal XML config

`,
}

func daemonDefaults() (string, string) {
	binary, err := os.Executable()
	cobra.CheckErr(err)
	binary = filepath.Clean(binary)

	_, name := filepath.Split(binary)
	name, _, _ = strings.Cut(name, ".")

	return binary, name
}

func initDaemon() CobraDaemon {

	binary, defaultName := daemonDefaults()
	common.ViperSetDefault("daemon.name", defaultName)

	systemUser, err := user.Current()
	cobra.CheckErr(err)
	common.ViperSetDefault("daemon.user", systemUser.Username)

	daemonUser, err := user.Lookup(common.ViperGetString("daemon.user"))
	cobra.CheckErr(err)
	common.ViperSetDefault("daemon.dir", daemonUser.HomeDir)

	name := common.ViperGetString("daemon.name")
	user := common.ViperGetString("daemon.user")
	dir := common.ViperGetString("daemon.dir")
	d, err := NewDaemon(name, user, dir, binary, daemonArgs...)
	cobra.CheckErr(err)
	return d
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install daemon",
	Long: `
install daemon config
`,

	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		_, err := d.GetConfig()
		if err == nil && common.ViperGetBool("force") {
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
start daemon
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
stop daemon
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
restart daemon
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
delete daemon config
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
show daemon config
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
return 0 if daemon is running, 1 if not
`,
	Run: func(cmd *cobra.Command, args []string) {
		d := initDaemon()
		quiet := common.ViperGetBool("daemon.query.quiet")
		running, err := d.Query()
		if !quiet {
			cobra.CheckErr(err)
		}
		if running {
			if !quiet {
				fmt.Println("running")
			}
			os.Exit(0)
		}
		if !quiet {
			fmt.Println("stopped")
		}
		os.Exit(1)
	},
}

func AddDaemonCommands(rootCmd *cobra.Command, args ...string) {
	daemonArgs = args
	common.CobraAddCommand(rootCmd, rootCmd, daemonCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonInstallCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonStartCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonStopCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonRestartCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonDeleteCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonShowCmd)
	common.CobraAddCommand(rootCmd, daemonCmd, daemonQueryCmd)
	common.OptionString(daemonCmd, "name", "", "", "daemon name")
	common.OptionString(daemonCmd, "user", "", "", "run as username")
	common.OptionString(daemonCmd, "dir", "", "", "run directory")
	common.OptionSwitch(daemonQueryCmd, "quiet", "q", "suppress output")
}
