package cmd

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type flagMeta struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type"`
	Usage      string `json:"usage"`
	Default    string `json:"default,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Inherited  bool   `json:"inherited,omitempty"`
}

type commandMeta struct {
	Command  string            `json:"command"`
	Use      string            `json:"use"`
	Short    string            `json:"short,omitempty"`
	Aliases  []string          `json:"aliases,omitempty"`
	Example  string            `json:"example,omitempty"`
	Endpoint map[string]string `json:"endpoint,omitempty"`
	Flags    []flagMeta        `json:"flags,omitempty"`
	Children []commandMeta     `json:"children,omitempty"`
}

func newCommandsCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commands",
		Short: "List commands with metadata for agents and automation",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}

			meta := buildCommandMeta(root, "af")
			if factory.Options.Output == "json" || factory.Options.Output == "jsonl" {
				return factory.WriteOutput(meta)
			}

			rows := flattenCommandMeta(meta)
			return factory.WriteOutput(rows)
		},
	}
	return cmd
}

func buildCommandMeta(cmd *cobra.Command, path string) commandMeta {
	meta := commandMeta{
		Command:  path,
		Use:      cmd.Use,
		Short:    cmd.Short,
		Aliases:  append([]string(nil), cmd.Aliases...),
		Example:  cmd.Example,
		Endpoint: endpointFor(cmd),
		Flags:    collectFlagMeta(cmd),
	}

	children := cmd.Commands()
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
	for _, child := range children {
		if child.Hidden {
			continue
		}
		childPath := strings.TrimSpace(path + " " + child.Name())
		meta.Children = append(meta.Children, buildCommandMeta(child, childPath))
	}
	return meta
}

func collectFlagMeta(cmd *cobra.Command) []flagMeta {
	flags := make([]flagMeta, 0)
	addFlags := func(set *pflag.FlagSet, inherited bool) {
		set.VisitAll(func(flag *pflag.Flag) {
			if flag.Hidden {
				return
			}
			flags = append(flags, flagMeta{
				Name:      flag.Name,
				Shorthand: flag.Shorthand,
				Type:      flag.Value.Type(),
				Usage:     flag.Usage,
				Default:   flag.DefValue,
				Required:  false,
				Inherited: inherited,
			})
		})
	}
	addFlags(cmd.LocalFlags(), false)
	addFlags(cmd.InheritedFlags(), true)
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Name < flags[j].Name
	})
	return flags
}

func flattenCommandMeta(root commandMeta) []map[string]any {
	rows := make([]map[string]any, 0)
	var walk func(commandMeta)
	walk = func(item commandMeta) {
		rows = append(rows, map[string]any{
			"command":  item.Command,
			"short":    item.Short,
			"endpoint": item.Endpoint,
			"flags":    item.Flags,
		})
		for _, child := range item.Children {
			walk(child)
		}
	}
	walk(root)
	return rows
}

func (m commandMeta) MarshalJSON() ([]byte, error) {
	type alias commandMeta
	return json.Marshal(alias(m))
}
