package daemon

import (
	"github.com/rstms/go-common"
	"github.com/spf13/cobra"
)

func fatal(err error) error {
	return common.Fatal(err)
}

func fatalf(template string, args ...interface{}) error {
	return common.Fatalf(template, args...)
}

func viperSetDefault(key string, value any) {
	common.ViperSetDefault(key, value)
}

func viperGetString(key string) string {
	return common.ViperGetString(key)
}

func viperGetBool(key string) bool {
	return common.ViperGetBool(key)
}

func cobraAddCommand(rootCmd, parentCmd, cmd *cobra.Command) {
	common.CobraAddCommand(rootCmd, parentCmd, cmd)
}
