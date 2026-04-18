package ttlbench

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "ttlbench", Short: "Run the TTL bench workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "ttlbench"))
	return cmd
}
