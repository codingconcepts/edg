package sysbench_read_write

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "sysbench-read-write", Short: "Run the sysbench oltp_read_write workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "sysbench-read-write"))
	return cmd
}
