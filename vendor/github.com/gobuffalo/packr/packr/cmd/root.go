package cmd

import (
	"context"
	"os"

	"github.com/gobuffalo/packr/builder"
	"github.com/spf13/cobra"
)

var input string
var compress bool

var rootCmd = &cobra.Command{
	Use:   "packr",
	Short: "compiles static files into Go files",
	RunE: func(cmd *cobra.Command, args []string) error {
		b := builder.New(context.Background(), input)
		b.Compress = compress
		return b.Run()
	},
}

func init() {
	pwd, _ := os.Getwd()
	rootCmd.Flags().StringVarP(&input, "input", "i", pwd, "path to scan for packr Boxes")
	rootCmd.Flags().BoolVarP(&compress, "compress", "z", false, "compress box contents")
}

// Execute the commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
