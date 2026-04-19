package sysbench_point_select

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "sysbench-point-select", Short: "Run the sysbench oltp_point_select workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "sysbench-point-select"))
	return cmd
}
