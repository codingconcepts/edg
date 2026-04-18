package ttllogger

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "ttllogger", Short: "Run the TTL logger workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "ttllogger"))
	return cmd
}
