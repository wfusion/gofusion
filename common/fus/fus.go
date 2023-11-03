package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/fus/asynq"
	"github.com/wfusion/gofusion/common/fus/debug"
	"github.com/wfusion/gofusion/common/fus/encode"
	"github.com/wfusion/gofusion/common/fus/gorm"
	"github.com/wfusion/gofusion/common/fus/mill"
	"github.com/wfusion/gofusion/common/fus/rnd"
)

const ver = "v0.0.4"

var buildTime = time.Now()

func main() {
	ctx := context.Background()

	// Execute adds all child commands to the root command and sets flags appropriately.
	// This is called by main.main(). It only needs to happen once to the rootCmd.
	if err := command().ExecuteContext(ctx); err != nil {
		fmt.Println(err)
	}
}

func command() *cobra.Command {
	v := version()
	rootCmd := &cobra.Command{
		Use:   "fus",
		Short: "Gofusion CLI",
		Long: fmt.Sprintf(`Gofusion CLI (%s)

Capability:
  asynq client integrated
  watermill client with pubsub kafka, ampq, and io enabled integerate
  gorm gentool integerate
  encoder&decoder with cipher, compress, and print encoding
  random bytes generater
`,
			v),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			DisableNoDescFlag:   false,
			DisableDescriptions: false,
			HiddenDefaultCmd:    false,
		},
		Version:          version(),
		TraverseChildren: true,
	}

	// watermill
	rootCmd.AddCommand(mill.Command())

	// asynq
	rootCmd.AddCommand(asynq.Command())

	// gorm gen-tool
	rootCmd.AddCommand(gorm.Command())

	// encode, decode
	rootCmd.AddCommand(encode.EncCommand(), encode.DecCommand())

	// random
	rootCmd.AddCommand(rnd.Command())

	// debug
	rootCmd.PersistentFlags().BoolVarP(&debug.Debug, "debug", "", false, "print debug info")

	// init
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultHelpFlag()
	rootCmd.InitDefaultVersionFlag()

	return rootCmd
}

func version() string {
	commitInf := gitTag
	if len(commitInf) == 0 {
		commitInf = gitCommit
	}
	if len(commitInf) > 8 {
		commitInf = commitInf[:8]
	}
	return fmt.Sprintf("%s built with %s %s/%s from %s on %s",
		ver,
		constant.GoVersion, constant.OS, constant.Arch,
		commitInf, buildTime.Format(time.UnixDate),
	)
}
