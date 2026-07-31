package cmd

import (
	"fmt"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Alpha version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[Alpha version] " + constant.Version + " (" + constant.BuildTime + ")")
	},
}
