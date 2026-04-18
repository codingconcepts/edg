package ycsb

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "ycsb", Short: "Run the YCSB workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "ycsb"))
	return cmd
}
