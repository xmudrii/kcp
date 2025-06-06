// Copyright 2013-2022 The Cobra Authors
// Copyright 2022 The KCP Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This program is a hacked version of the cobra cli-doc generator to generate
// cli-docs according to Hugo's conventions. The original program can be
// found at https://github.com/spf13/cobra/tree/main/doc

package doc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("### Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("### Options inherited from parent commands\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}
	return nil
}

// GenMarkdown creates markdown output.
func GenMarkdown(cmd *cobra.Command, w io.Writer) error {
	return GenMarkdownCustom(cmd, w, func(s string) string { return s })
}

// GenMarkdownCustom creates custom markdown output.
func GenMarkdownCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	buf.WriteString("---\ntitle: \"" + name + "\"\nlinkTitle: \"" + name + "\"\nweight: 1\ndescription: >\n  " + cmd.Short + "\n---\n\n")
	if len(cmd.Long) > 0 {
		buf.WriteString("### Synopsis\n\n")
		buf.WriteString(cmd.Long + "\n\n")
	}

	if cmd.Runnable() {
		fmt.Fprintf(buf, "```\n%s\n```\n\n", cmd.UseLine())
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("### Examples\n\n")
		fmt.Fprintf(buf, "```\n%s\n```\n\n", strings.TrimSpace(cmd.Example))
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		var links []string

		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()

			// skip "create" plugins
			if pname != "create" {
				link := pname + ".md"
				link = strings.ReplaceAll(link, " ", "_")
				links = append(links, fmt.Sprintf("* [%s](%s) – %s\n", pname, linkHandler(link), parent.Short))

				cmd.VisitParents(func(c *cobra.Command) {
					if c.DisableAutoGenTag {
						cmd.DisableAutoGenTag = c.DisableAutoGenTag
					}
				})
			}
		}

		children := cmd.Commands()
		sort.Sort(byName(children))

		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			cname := name + " " + child.Name()
			link := cname + ".md"
			link = strings.ReplaceAll(link, " ", "_")
			links = append(links, fmt.Sprintf("* [%s](%s) – %s\n", cname, linkHandler(link), child.Short))
		}

		if len(links) > 0 {
			buf.WriteString("### See Also\n\n")
			for _, link := range links {
				buf.WriteString(link)
			}
			buf.WriteString("\n")
		}
	}

	_, err := buf.WriteTo(w)
	return err
}

// GenMarkdownTree will generate a markdown page for this command and all
// descendants in the directory given. The header may be nil.
// This function may not work correctly if your command names have `-` in them.
// If you have `cmd` with two subcmds, `sub` and `sub-third`,
// and `sub` has a subcommand called `third`, it is undefined which
// help output will be in the file `cmd-sub-third.1`.
func GenMarkdownTree(cmd *cobra.Command, dir string) error {
	identity := func(s string) string { return "./" + s }
	emptyStr := func(s string) string { return "" }
	return GenMarkdownTreeCustom(cmd, dir, emptyStr, identity)
}

// GenMarkdownTreeCustom is the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
func GenMarkdownTreeCustom(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenMarkdownTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.ReplaceAll(cmd.CommandPath(), " ", "_") + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(filePrepender(filename)); err != nil {
		return err
	}
	return GenMarkdownCustom(cmd, f, linkHandler)
}
