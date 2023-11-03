package rnd

import (
	"crypto/rand"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"github.com/wfusion/gofusion/common/fus/util"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:     "rnd [flags] [number of bytes]",
		Short:   "Generate cryptographically secure random bytes",
		Example: "  fus rnd 16",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			size, err := cast.ToIntE(args[0])
			if err != nil {
				return
			}
			bs := make([]byte, size)
			if _, err = rand.Read(bs); err != nil {
				return
			}
			util.PrintOutput(bs)
			return
		},
	}
}
