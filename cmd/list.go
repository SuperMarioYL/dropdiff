package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/SuperMarioYL/dropdiff/internal/spec"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the drops in the current registry with sources",
	Long:  `Prints every drop in the (synced or bundled) registry with its pricing, context, and source — the m2 done-check ("list shows >=8 drops with sources").`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, regSource, err := loadRegistry()
		if err != nil {
			return err
		}
		fmt.Printf("registry: %s  (%d drops, updated %s)\n\n", regSource, len(reg.Drops), reg.Updated)
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MODEL\tVENDOR\tDATE\tINPUT\tOUTPUT\tCACHE\tCONTEXT\tQUANTS\tSOURCE")
		for _, d := range reg.Drops {
			fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%.2f\t%.2f\t%d\t%s\t%s\n",
				d.Model, d.Vendor, d.DropDate,
				d.Pricing.Input, d.Pricing.Output, d.Pricing.Cache,
				d.Context, joinQuants(d.Quants), shortSource(d.Source))
		}
		return w.Flush()
	},
}

func joinQuants(q []string) string {
	out := ""
	for i, v := range q {
		if i > 0 {
			out += "/"
		}
		out += v
	}
	return out
}

func shortSource(s spec.SourceRef) string {
	switch {
	case s.SDKCommitSHA != "":
		return s.SDKCommitSHA
	case s.ModelScopeCard != "":
		return "modelscope"
	}
	return "—"
}

func init() {
	rootCmd.AddCommand(listCmd)
}
