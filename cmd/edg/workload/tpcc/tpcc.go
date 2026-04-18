package tpcc

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "tpcc", Short: "Run the TPC-C workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "tpcc"))
	return cmd
}
