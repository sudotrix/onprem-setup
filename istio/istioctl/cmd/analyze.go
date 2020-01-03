// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"istio.io/pkg/env"

	"istio.io/istio/galley/pkg/config/analysis"
	"istio.io/istio/galley/pkg/config/analysis/analyzers"
	"istio.io/istio/galley/pkg/config/analysis/diag"
	"istio.io/istio/galley/pkg/config/analysis/local"
	"istio.io/istio/galley/pkg/config/meta/metadata"
	"istio.io/istio/galley/pkg/config/resource"
	cfgKube "istio.io/istio/galley/pkg/config/source/kube"
	"istio.io/istio/istioctl/pkg/util/handlers"
	"istio.io/istio/pkg/kube"
)

type AnalyzerFoundIssuesError struct{}
type FileParseError struct{}

const (
	NoIssuesString   = "\u2714 No validation issues found."
	FoundIssueString = "Analyzers found issues."
	FileParseString  = "Some files couldn't be parsed."
	LogOutput        = "log"
	JSONOutput       = "json"
	YamlOutput       = "yaml"
)

func (f AnalyzerFoundIssuesError) Error() string {
	return fmt.Sprintf("%s\nSee %s for more information about causes and resolutions.", FoundIssueString, diag.DocPrefix)
}

func (f FileParseError) Error() string {
	return FileParseString
}

var (
	listAnalyzers   bool
	useKube         bool
	failureLevel    = messageThreshold{diag.Warning} // messages at least this level will generate an error exit code
	outputLevel     = messageThreshold{diag.Info}    // messages at least this level will be included in the output
	colorize        bool
	msgOutputFormat string
	meshCfgFile     string
	allNamespaces   bool

	termEnvVar = env.RegisterStringVar("TERM", "", "Specifies terminal type.  Use 'dumb' to suppress color output")

	colorPrefixes = map[diag.Level]string{
		diag.Info:    "",           // no special color for info messages
		diag.Warning: "\033[33m",   // yellow
		diag.Error:   "\033[1;31m", // bold red
	}
)

// Analyze command
func Analyze() *cobra.Command {
	// Validate the output format before doing potentially expensive work to fail earlier
	msgOutputFormats := map[string]bool{LogOutput: true, JSONOutput: true, YamlOutput: true}
	var msgOutputFormatKeys []string

	for k := range msgOutputFormats {
		msgOutputFormatKeys = append(msgOutputFormatKeys, k)
	}

	analysisCmd := &cobra.Command{
		Use:   "analyze <file>...",
		Short: "Analyze Istio configuration and print validation messages",
		Example: `
# Analyze the current live cluster
istioctl analyze

# Analyze the current live cluster, simulating the effect of applying additional yaml files
istioctl analyze a.yaml b.yaml

# Analyze yaml files without connecting to a live cluster
istioctl analyze --use-kube=false a.yaml b.yaml

# List available analyzers
istioctl analyze -L
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			msgOutputFormat = strings.ToLower(msgOutputFormat)
			_, ok := msgOutputFormats[msgOutputFormat]
			if !ok {
				return CommandParseError{
					fmt.Errorf("%s not a valid option for format. See istioctl analyze --help", msgOutputFormat),
				}
			}

			if listAnalyzers {
				fmt.Print(AnalyzersAsString(analyzers.All()))
				return nil
			}

			readers, err := gatherFiles(args)
			if err != nil {
				return err
			}
			cancel := make(chan struct{})

			// We use the "namespace" arg that's provided as part of root istioctl as a flag for specifying what namespace to use
			// for file resources that don't have one specified.
			selectedNamespace := handlers.HandleNamespace(namespace, defaultNamespace)

			var k cfgKube.Interfaces
			if useKube {
				// Set up the kube client
				config := kube.BuildClientCmd(kubeconfig, configContext)
				restConfig, err := config.ClientConfig()
				if err != nil {
					return err
				}
				k = cfgKube.NewInterfaces(restConfig)
			}

			// If we've explicitly asked for all namespaces, blank the selectedNamespace var out
			if allNamespaces {
				selectedNamespace = ""
			}

			sa := local.NewSourceAnalyzer(metadata.MustGet(), analyzers.AllCombined(),
				resource.Namespace(selectedNamespace), resource.Namespace(istioNamespace), nil, true)

			// If we're using kube, use that as a base source.
			if k != nil {
				sa.AddRunningKubeSource(k)
			}

			// If files are provided, treat them (collectively) as a source.
			parseErrors := 0
			if len(readers) > 0 {
				if err = sa.AddReaderKubeSource(readers); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Error(s) adding files: %v", err)
					parseErrors++
				}
			}

			// If we explicitly specify mesh config, use it.
			// This takes precedence over default mesh config or mesh config from a running Kube instance.
			if meshCfgFile != "" {
				_ = sa.AddFileKubeMeshConfigSource(meshCfgFile)
			}

			// Do the analysis
			result, err := sa.Analyze(cancel)
			if err != nil {
				return err
			}

			// Maybe output details about which analyzers ran
			if verbose {
				if allNamespaces {
					fmt.Fprintln(cmd.ErrOrStderr(), "Analyzed resources in all namespaces")
				} else {
					fmt.Fprintln(cmd.ErrOrStderr(), "Analyzed resources in namespace:", selectedNamespace)
				}

				if len(result.SkippedAnalyzers) > 0 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Skipped analyzers:")
					for _, a := range result.SkippedAnalyzers {
						fmt.Fprintln(cmd.ErrOrStderr(), "\t", a)
					}
				}
				if len(result.ExecutedAnalyzers) > 0 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Executed analyzers:")
					for _, a := range result.ExecutedAnalyzers {
						fmt.Fprintln(cmd.ErrOrStderr(), "\t", a)
					}
				}
				fmt.Fprintln(cmd.ErrOrStderr())
			}

			// Filter outputMessages by specified level, and append a ref arg to the doc URL
			var outputMessages diag.Messages
			for _, m := range result.Messages {
				if m.Type.Level().IsWorseThanOrEqualTo(outputLevel.Level) {
					m.DocRef = "istioctl-analyze"
					outputMessages = append(outputMessages, m)
				}
			}

			switch msgOutputFormat {
			case LogOutput:
				// Print validation message output, or a line indicating that none were found
				if len(outputMessages) == 0 {
					if parseErrors == 0 {
						fmt.Fprintln(cmd.ErrOrStderr(), NoIssuesString)
					} else {
						fileOrFiles := "files"
						if parseErrors == 1 {
							fileOrFiles = "file"
						}
						fmt.Fprintf(cmd.ErrOrStderr(),
							"No validation issues found (but %d %s could not be parsed)\n",
							parseErrors,
							fileOrFiles,
						)
					}
				} else {
					for _, m := range outputMessages {
						fmt.Fprintln(cmd.OutOrStdout(), renderMessage(m))
					}
				}
			case JSONOutput:
				jsonOutput, err := json.MarshalIndent(outputMessages, "", "\t")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(jsonOutput))
			case YamlOutput:
				yamlOutput, err := yaml.Marshal(outputMessages)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(yamlOutput))
			default: // This should never happen since we validate this already
				panic(fmt.Sprintf("%q not found in output format switch statement post validate?", msgOutputFormat))
			}

			// Return code is based on the unfiltered validation message list/parse errors
			// We're intentionally keeping failure threshold and output threshold decoupled for now
			returnError := errorIfMessagesExceedThreshold(result.Messages)
			if returnError == nil && parseErrors > 0 {
				returnError = FileParseError{}
			}
			return returnError
		},
	}

	analysisCmd.PersistentFlags().BoolVarP(&listAnalyzers, "list-analyzers", "L", false,
		"List the analyzers available to run. Suppresses normal execution.")
	analysisCmd.PersistentFlags().BoolVarP(&useKube, "use-kube", "k", true,
		"Use live Kubernetes cluster for analysis. Set --use-kube=false to analyze files only.")
	analysisCmd.PersistentFlags().BoolVar(&colorize, "color", istioctlColorDefault(analysisCmd),
		"Default true.  Disable with '=false' or set $TERM to dumb")
	analysisCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable verbose output")
	analysisCmd.PersistentFlags().Var(&failureLevel, "failure-threshold",
		fmt.Sprintf("The severity level of analysis at which to set a non-zero exit code. Valid values: %v", diag.GetAllLevelStrings()))
	analysisCmd.PersistentFlags().Var(&outputLevel, "output-threshold",
		fmt.Sprintf("The severity level of analysis at which to display messages. Valid values: %v", diag.GetAllLevelStrings()))
	analysisCmd.PersistentFlags().StringVarP(&msgOutputFormat, "output", "o", LogOutput,
		fmt.Sprintf("Output format: one of %v", msgOutputFormatKeys))
	analysisCmd.PersistentFlags().StringVar(&meshCfgFile, "meshConfigFile", "",
		"Overrides the mesh config values to use for analysis.")
	analysisCmd.PersistentFlags().BoolVarP(&allNamespaces, "all-namespaces", "A", false,
		"Analyze all namespaces")
	return analysisCmd
}

func gatherFiles(args []string) ([]io.Reader, error) {
	var readers []io.Reader
	var r *os.File
	var err error
	for _, f := range args {
		if f == "-" {
			if isatty.IsTerminal(os.Stdin.Fd()) {
				fmt.Fprint(os.Stderr, "Reading from stdin:\n")
			}
			r = os.Stdin
		} else {
			r, err = os.Open(f)
			if err != nil {
				return nil, err
			}
			runtime.SetFinalizer(r, func(x *os.File) { x.Close() })
		}
		readers = append(readers, r)
	}
	return readers, nil
}

func colorPrefix(m diag.Message) string {
	if !colorize {
		return ""
	}

	prefix, ok := colorPrefixes[m.Type.Level()]
	if !ok {
		return ""
	}

	return prefix
}

func colorSuffix() string {
	if !colorize {
		return ""
	}

	return "\033[0m"
}

func renderMessage(m diag.Message) string {
	origin := ""
	if m.Origin != nil {
		origin = " (" + m.Origin.FriendlyName() + ")"
	}
	return fmt.Sprintf(
		"%s%v%s [%v]%s %s", colorPrefix(m), m.Type.Level(), colorSuffix(), m.Type.Code(), origin, fmt.Sprintf(m.Type.Template(), m.Parameters...))
}

func istioctlColorDefault(cmd *cobra.Command) bool {
	if strings.EqualFold(termEnvVar.Get(), "dumb") {
		return false
	}

	file, ok := cmd.OutOrStdout().(*os.File)
	if ok {
		if !isatty.IsTerminal(file.Fd()) {
			return false
		}
	}

	return true
}

func errorIfMessagesExceedThreshold(messages []diag.Message) error {
	foundIssues := false
	for _, m := range messages {
		if m.Type.Level().IsWorseThanOrEqualTo(failureLevel.Level) {
			foundIssues = true
		}
	}

	if foundIssues {
		return AnalyzerFoundIssuesError{}
	}

	return nil
}

type messageThreshold struct {
	diag.Level
}

// String satisfies interface pflag.Value
func (m *messageThreshold) String() string {
	return m.Level.String()
}

// Type satisfies interface pflag.Value
func (m *messageThreshold) Type() string {
	return "Level"
}

// Set satisfies interface pflag.Value
func (m *messageThreshold) Set(s string) error {
	l, err := LevelFromString(s)
	if err != nil {
		return err
	}
	m.Level = l
	return nil
}

func LevelFromString(s string) (diag.Level, error) {
	val, ok := diag.GetUppercaseStringToLevelMap()[strings.ToUpper(s)]
	if !ok {
		return diag.Level{}, fmt.Errorf("%q not a valid option, please choose from: %v", s, diag.GetAllLevelStrings())
	}

	return val, nil
}

func AnalyzersAsString(analyzers []analysis.Analyzer) string {
	nameToAnalyzer := make(map[string]analysis.Analyzer)
	analyzerNames := make([]string, len(analyzers))
	for i, a := range analyzers {
		analyzerNames[i] = a.Metadata().Name
		nameToAnalyzer[a.Metadata().Name] = a
	}
	sort.Strings(analyzerNames)

	var b strings.Builder
	for _, aName := range analyzerNames {
		b.WriteString(fmt.Sprintf("* %s:\n", aName))
		a := nameToAnalyzer[aName]
		if a.Metadata().Description != "" {
			b.WriteString(fmt.Sprintf("    %s\n", a.Metadata().Description))
		}
	}
	return b.String()
}
