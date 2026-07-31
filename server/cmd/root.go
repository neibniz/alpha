package cmd

import (
	"os"

	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

var rootCmd = &cobra.Command{
	Use:   "Alpha",
	Short: "Alpha is a modern multi-user agent panel.",
	Long: `[ Alpha is a pure, professional, and perfect open-source proxy panel tool, designed to be your ideal choice for learning and practical use.]
[ Simple and easy to operate.]`,
	Run: func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorw("[Command] Execute failed", logger.Field("error", err.Error()))
		os.Exit(1)
	}
}
