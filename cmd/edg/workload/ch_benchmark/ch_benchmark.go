package ch_benchmark

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "ch-benchmark", Short: "Run the CH-benCHmark workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "ch-benchmark"))
	return cmd
}
