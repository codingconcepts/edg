package sysbench_update_index

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "sysbench-update-index", Short: "Run the sysbench oltp_update_index workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "sysbench-update-index"))
	return cmd
}
