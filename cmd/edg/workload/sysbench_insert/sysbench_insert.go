package sysbench_insert

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "sysbench-insert", Short: "Run the sysbench oltp_insert workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "sysbench-insert"))
	return cmd
}
