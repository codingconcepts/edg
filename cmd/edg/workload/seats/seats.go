package seats

import (
	"embed"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/spf13/cobra"
)

//go:embed *.yaml
var yamlFS embed.FS

func Cmd(deps workload.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "seats", Short: "Run the SEATS workload"}
	workload.AddSubcommands(cmd, workload.WithWorkload(deps, yamlFS, "seats"))
	return cmd
}
