package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	rc_lib "android/soong/cmd/release_config/release_config_lib"
	rc_proto "android/soong/cmd/release_config/release_config_proto"

	"google.golang.org/protobuf/proto"
)

type GlobalFlags struct {
	// The path to the top of the workspace.  Default: ".".
	top string

	// Pathlist of release config map textproto files.
	// If not specified, then the value is (if present):
	// - build/release/release_config_map.textproto
	// - vendor/google_shared/build/release/release_config_map.textproto
	// - vendor/google/release/release_config_map.textproto
	//
	// Additionally, any maps specified in the environment variable
	// `PRODUCT_RELEASE_CONFIG_MAPS` are used.
	maps rc_lib.StringList

	// File containing the list of maps to use, one file per line.
	// Cannot be used with --map.
	mapsFile string

	// Output directory (relative to `top`).
	outDir string

	// Which $TARGET_RELEASE(s) should we use.  Some commands will only
	// accept one value, others also accept `--release --all`.
	targetReleases rc_lib.StringList

	// The TARGET_BUILD_VARIANT to use
	targetBuildVariant string

	// Disable warning messages
	quiet bool

	// Show all release configs
	allReleases bool

	// Call get_build_var PRODUCT_RELEASE_CONFIG_MAPS to get the
	// product-specific map directories.
	useGetBuildVar bool

	// Panic on errors.
	debug bool

	// Allow missing release config.
	// If true, and we cannot find the named release config, values for
	// `trunk_staging` will be used.
	allowMissing bool

	// Only load flag declarations, do not load values.  The output
	// will have only values provided in the declaration files.
	declarationsOnly bool
}

type CommandFunc func(*rc_lib.ReleaseConfigs, GlobalFlags, ...string) error

type CommandInfo struct {
	// The function that executes this command.
	run CommandFunc

	// Helptext for usage.
	helpText string
}

var (
	releaseFlagSet   = ReleaseFlagsFactory()
	getCommandInfo   = CommandInfo{run: GetCommandFactory(), helpText: "Prints flag values"}
	setCommandInfo   = CommandInfo{run: SetCommandFactory(), helpText: "Generates changes to flag value files"}
	traceCommandInfo = CommandInfo{run: TraceCommandFactory(), helpText: "Shows flag override history"}

	commandMap map[string]CommandInfo = map[string]CommandInfo{
		"get":   getCommandInfo,
		"set":   setCommandInfo,
		"trace": traceCommandInfo,
	}

	setArgRegexp = regexp.MustCompile(`^((?P<dir>[^:]+):)?(?P<flag>[0-9A-Z_]+)(=(?P<value>.*)|:(?P<redacted>redacted))$`)
)

// Find the top of the release config contribution directory.
// Returns the parent of the flag_declarations and flag_values directories.
func GetMapDir(path string) (string, error) {
	for p := path; p != "."; p = filepath.Dir(p) {
		switch filepath.Base(p) {
		case "flag_declarations":
			return filepath.Dir(p), nil
		case "flag_values":
			return filepath.Dir(p), nil
		}
	}
	return "", fmt.Errorf("Could not determine directory from %s", path)
}

func MarshalFlagDefaultValue(config *rc_lib.ReleaseConfig, name string) (ret string, err error) {
	fa, ok := config.FlagArtifacts[name]
	if !ok {
		return "", fmt.Errorf("%s not found in %s", name, config.Name)
	}
	return rc_lib.MarshalValue(fa.Traces[0].Value), nil
}

func MarshalFlagValue(config *rc_lib.ReleaseConfig, name string) (val, typ string, err error) {
	fa, ok := config.FlagArtifacts[name]
	if !ok {
		return "", "unspecified", fmt.Errorf("%s not found in %s", name, config.Name)
	}
	if fa.Redacted {
		return "==REDACTED==", "unspecified", nil
	}
	return rc_lib.MarshalValue(fa.Value), rc_lib.ValueType(fa.Value), nil
}

// Returns a list of ReleaseConfig objects for which to process flags.
func GetReleaseArgs(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags) ([]*rc_lib.ReleaseConfig, error) {
	releaseFlagSet.flagSet.Parse(globalFlags.targetReleases)
	var ret []*rc_lib.ReleaseConfig
	if releaseFlagSet.all || globalFlags.allReleases {
		sortMap := map[string]int{
			"trunk_staging": 0,
			"trunk_food":    10,
			"trunk":         20,
			// Anything not listed above, uses this for key 1 in the sort.
			"-default": 100,
		}

		if err := configs.GenerateAllReleaseConfigs(globalFlags.targetReleases[0]); err != nil {
			return nil, err
		}
		for _, config := range configs.ReleaseConfigs {
			ret = append(ret, config)
		}
		slices.SortFunc(ret, func(a, b *rc_lib.ReleaseConfig) int {
			mapValue := func(v *rc_lib.ReleaseConfig) int {
				if v, ok := sortMap[v.Name]; ok {
					return v
				}
				return sortMap["-default"]
			}
			if n := cmp.Compare(mapValue(a), mapValue(b)); n != 0 {
				return n
			}
			return cmp.Compare(a.Name, b.Name)
		})
		return ret, nil
	}
	for _, arg := range releaseFlagSet.flagSet.Args() {
		// Return releases in the order that they were given.
		config, err := configs.GetReleaseConfig(arg)
		if err != nil {
			return nil, err
		}
		ret = append(ret, config)
	}
	return ret, nil
}

type ReleaseFlags struct {
	flagSet *flag.FlagSet

	// Display all releases
	all bool
}

func ReleaseFlagsFactory() *ReleaseFlags {
	flags := &ReleaseFlags{
		flagSet: flag.NewFlagSet("releaseFlags", flag.ExitOnError),
	}
	flags.flagSet.BoolVar(&flags.all, "all", false, "Display all releases")
	return flags
}

type GetFlags struct {
	flagSet *flag.FlagSet

	// Display all flags
	all bool

	// Output flag as json object
	json bool

	// Show all distinct values in all releases
	distinctValues bool

	// Hide flag name
	hideName bool
}

func GetCommandFactory() CommandFunc {
	flags := &GetFlags{
		flagSet: flag.NewFlagSet("get", flag.ExitOnError),
	}
	flags.flagSet.BoolVar(&flags.all, "all", false, "Display all flags")
	flags.flagSet.BoolVar(&flags.json, "json", false, "Output flag as json object")
	flags.flagSet.BoolVar(&flags.hideName, "hide-name", false, "Hide build flag names. (True when only one flag name is given)")
	flags.flagSet.BoolVar(&flags.distinctValues, "distinct-values", false, "Show all distinct values in all releases")
	return func(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags, args ...string) error {
		return GetCommand(configs, globalFlags, flags, args...)
	}
}

func TraceCommandFactory() CommandFunc {
	// The `trace` command is also handled by GetCommand.
	flags := &GetFlags{
		flagSet: flag.NewFlagSet("trace", flag.ExitOnError),
	}
	flags.flagSet.BoolVar(&flags.all, "all", false, "Display all flags")
	return func(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags, args ...string) error {
		return GetCommand(configs, globalFlags, flags, args...)
	}
}

type SetFlags struct {
	flagSet *flag.FlagSet

	// Directory in which to place the value
	dir string

	// Whether the flag should be redacted
	redacted bool
}

func SetCommandFactory() CommandFunc {
	flags := &SetFlags{
		flagSet: flag.NewFlagSet("set", flag.ExitOnError),
	}
	flags.flagSet.StringVar(&flags.dir, "dir", "", "Directory in which to place the value")
	flags.flagSet.BoolVar(&flags.redacted, "redacted", false, "Whether the flag should be redacted")
	return func(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags, args ...string) error {
		return SetCommand(configs, globalFlags, flags, args...)
	}
}

func GetCommand(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags, getFlags *GetFlags, args ...string) error {
	if len(args) < 1 {
		panic("missing command")
	}
	cmd := args[0]
	if cmd == "printdefaults" {
		getFlags.flagSet.PrintDefaults()
		return nil
	}
	getFlags.flagSet.Parse(args[1:])
	args = getFlags.flagSet.Args()
	isTrace := cmd == "trace"
	isSet := cmd == "set"

	if isSet || getFlags.distinctValues {
		globalFlags.allReleases = true
	}
	releaseConfigList, err := GetReleaseArgs(configs, globalFlags)
	if err != nil {
		return err
	}

	if len(releaseConfigList) > 1 {
		switch {
		case isTrace:
			return fmt.Errorf("trace command only allows one --release argument.  Got: %s", strings.Join(globalFlags.targetReleases, " "))
		case getFlags.json:
			return fmt.Errorf("--json only allows one --release argument.  Got: %s", strings.Join(globalFlags.targetReleases, " "))
		}
	}

	if getFlags.all {
		args = []string{}
		for _, fa := range configs.FlagArtifacts {
			args = append(args, *fa.FlagDeclaration.Name)
		}
		slices.Sort(args)
	}

	var maxVariableNameLen, maxReleaseNameLen int
	var releaseNameFormat, variableNameFormat string
	valueFormat := "%s"
	showReleaseName := len(releaseConfigList) > 1 && !getFlags.distinctValues
	showVariableName := (len(args) > 1 && !getFlags.hideName) || getFlags.json
	if getFlags.json {
		variableNameFormat = `    "%s": `
		valueFormat = `"%s"`
	} else if showVariableName {
		for _, arg := range args {
			maxVariableNameLen = max(len(arg), maxVariableNameLen)
		}
		variableNameFormat = fmt.Sprintf("%%-%ds ", maxVariableNameLen)
		valueFormat = "'%s'"
		if getFlags.distinctValues {
			variableNameFormat = "%s:"
			valueFormat = "%s"
		}
	}
	if showReleaseName {
		for _, config := range releaseConfigList {
			maxReleaseNameLen = max(len(config.Name), maxReleaseNameLen)
		}
		releaseNameFormat = fmt.Sprintf("%%-%ds ", maxReleaseNameLen)
		valueFormat = "'%s'"
	}

	outputOneLine := func(variable, release, value, valueFormat string, last bool) {
		var outStr string
		if showVariableName {
			outStr += fmt.Sprintf(variableNameFormat, variable)
		}
		if showReleaseName {
			outStr += fmt.Sprintf(releaseNameFormat, release)
		}
		outStr += fmt.Sprintf(valueFormat, value)
		if getFlags.json && !last {
			outStr += ","
		}
		fmt.Println(outStr)
	}

	newArgs := []string{}
	for _, arg := range args {
		if configs.IgnoredFlags[arg] {
			fmt.Fprintf(os.Stderr, "%s is a deleted flag\n", arg)
			continue
		}
		newArgs = append(newArgs, arg)
		if _, ok := configs.FlagArtifacts[arg]; !ok {
			return fmt.Errorf("%s is not a defined build flag", arg)
		}
	}
	args = newArgs

	if getFlags.distinctValues {
		values := map[string]bool{}
		for _, arg := range args {
			for _, config := range releaseConfigList {
				val, _, err := MarshalFlagValue(config, arg)
				if err == nil && val != "" {
					values[val] = true
				}
			}
			sortedValues := rc_lib.SortedKeys(values)
			numValues := len(sortedValues)
			for idx, val := range sortedValues {
				outputOneLine(arg, "various", val, valueFormat, idx == numValues-1)
			}
		}
		return nil
	}

	if getFlags.json {
		fmt.Println(`"BuildFlags": {`)
	}

	numArgs := len(args)
	flagTypes := make(map[string]string)
	for argIdx, arg := range args {
		for _, config := range releaseConfigList {
			if isSet {
				// If this is from the set command, format the output as:
				// <default>           ""
				// trunk_staging       ""
				// trunk               ""
				//
				// ap1a                ""
				// ...
				switch {
				case config.Name == "trunk_staging":
					defaultValue, err := MarshalFlagDefaultValue(config, arg)
					if err != nil {
						return err
					}
					outputOneLine(arg, "<default>", defaultValue, valueFormat, argIdx == numArgs-1)
				case config.AconfigFlagsOnly:
					continue
				case config.Name == "trunk":
					fmt.Println()
				}
			}
			val, typ, err := MarshalFlagValue(config, arg)
			if err == nil {
				outputOneLine(arg, config.Name, val, valueFormat, argIdx == numArgs-1)
				flagTypes[arg] = typ
			} else if !getFlags.json {
				outputOneLine(arg, config.Name, "REDACTED", "%s", argIdx == numArgs-1)
			}
			if err == nil && isTrace {
				for _, trace := range config.FlagArtifacts[arg].Traces {
					fmt.Printf("  => \"%s\" in %s\n", rc_lib.MarshalValue(trace.Value), *trace.Source)
				}
			}
		}
	}
	if getFlags.json {
		fmt.Println("},")
		fmt.Println(`"BuildFlagTypes": {`)
		for argIdx, arg := range args {
			for _, config := range releaseConfigList {
				outputOneLine(arg, config.Name, flagTypes[arg], valueFormat, argIdx == numArgs-1)
			}
		}
		fmt.Println("}")
	}
	return nil
}

// Supported syntax:
//
// Legacy format to set exactly one flag:
//
//	set [--dir DIR] FLAG VALUE
//
// Multi-flag format:
//
//	set [--dir DIR] --redacted FLAG ...
//	set [[DIR:]FLAG:redacted] [[DIR:]FLAG=VALUE] ...
func SetCommand(configs *rc_lib.ReleaseConfigs, globalFlags GlobalFlags, setFlags *SetFlags, args ...string) error {
	if len(args) < 1 {
		panic("missing command")
	}
	cmd := args[0]
	if cmd == "printdefaults" {
		setFlags.flagSet.PrintDefaults()
		return nil
	}
	if len(globalFlags.targetReleases) > 1 {
		return fmt.Errorf("set command only allows one --release argument.  Got: %s", strings.Join(globalFlags.targetReleases, " "))
	}
	setFlags.flagSet.Parse(args[1:])
	targetRelease := globalFlags.targetReleases[0]
	release, err := configs.GetReleaseConfig(targetRelease)
	targetRelease = release.Name
	if err != nil {
		return err
	}
	if release.AconfigFlagsOnly {
		return fmt.Errorf("%s does not allow build flag overrides", targetRelease)
	}

	setArgs := setFlags.flagSet.Args()
	// Handle the legacy syntax where FLAG and VALUE were separate args.
	// - --redacted FLAG
	// - FLAG VALUE
	switch {
	case len(setArgs) == 0 && setFlags.redacted:
		// No flags given.
		return fmt.Errorf("set command expected '--redacted [DIR:]FLAG' or '[DIR:]FLAG:redacted'")
	case len(setArgs) == 0:
		fmt.Printf("Nothing to do: no flags given.")
		return nil
	case setFlags.redacted || strings.HasSuffix(setArgs[0], ":redacted") || strings.Contains(setArgs[0], "="):
		// No special handling required.
	case len(setArgs) == 1 && !setFlags.redacted:
		// A single argument is only valid if we are redacting.
		return fmt.Errorf("set command expected '[DIR:]FLAG VALUE' or '[DIR:]FLAG=VALUE'")
	case len(setArgs) == 2 && !strings.Contains(setArgs[0], "="):
		// Handle the `FLAG VALUE` case.
		// Convert `FLAG VALUE` to `FLAG=VALUE` to simplify processing.
		setArgs = []string{fmt.Sprintf("%s=%s", setArgs[0], setArgs[1])}
	default:
		// No special handling.  If there are syntactic errors, they will cause errors in the for loop below.
	}

	var updatedFiles []string
	getArgs := []string{cmd}
	for _, arg := range setArgs {
		match := setArgRegexp.FindStringSubmatch(arg)
		if match == nil {
			if setFlags.redacted && !strings.HasSuffix(arg, ":redacted") {
				// Keep the rest of the logic simpler by re-parsing the flag with `:redacted` appended.
				// Leave `arg` unchanged in case we generate an error.
				match = setArgRegexp.FindStringSubmatch(arg + ":redacted")
			}
			if match == nil {
				return fmt.Errorf("Expected %q or %q, got %q", "[DIR:]FLAG=VALUE", "[DIR:]FLAG:redacted", arg)
			}
		}
		reFields := make(map[string]string)
		for i, name := range setArgRegexp.SubexpNames() {
			if i != 0 && name != "" {
				reFields[name] = match[i]
			}
		}
		flagName := reFields["flag"]
		flagArtifact, ok := release.FlagArtifacts[flagName]
		if !ok {
			return fmt.Errorf("Unknown build flag %s", flagName)
		}

		flagValue := &rc_proto.FlagValue{
			Name: proto.String(flagName),
		}
		switch {
		case reFields["value"] != "":
			flagValue.Value = rc_lib.UnmarshalValue(reFields["value"])
		case reFields["redacted"] != "" || setFlags.redacted:
			flagValue.Redacted = proto.Bool(true)
		default:
			return fmt.Errorf("Expected either '=VALUE' or ':redacted' in %q", arg)
		}
		// Write the flag to:
		// - The `DIR:` location from the arg, or
		// - The path from `--dir PATH`, or
		// - The directory from GetFlagValueDirectory() for the flag.
		var mapDir string
		switch {
		case reFields["dir"] != "":
			mapDir = reFields["dir"]
		case setFlags.dir != "":
			mapDir = setFlags.dir
		default:
			mapDir, err = configs.GetFlagValueDirectory(release, flagArtifact)
			if err != nil {
				return err
			}
		}

		rcPath := filepath.Join(mapDir, "release_configs", fmt.Sprintf("%s.textproto", targetRelease))
		// Create the release config declaration only if necessary.
		if _, err = os.Stat(rcPath); err != nil {
			if err = os.MkdirAll(filepath.Dir(rcPath), 0775); err != nil {
				return err
			}
			rcValue := &rc_proto.ReleaseConfig{
				Name: proto.String(targetRelease),
			}
			err = rc_lib.WriteMessage(rcPath, rcValue)
			if err != nil {
				return err
			}
			updatedFiles = append(updatedFiles, rcPath)
		}

		flagPath := filepath.Join(mapDir, "flag_values", targetRelease, fmt.Sprintf("%s.textproto", flagName))
		err = rc_lib.WriteMessage(flagPath, flagValue)
		if err != nil {
			return err
		}
		updatedFiles = append(updatedFiles, flagPath)
		getArgs = append(getArgs, flagName)
	}

	// Reload the release configs.
	configs, err = rc_lib.ReadReleaseConfigMaps(globalFlags.maps, globalFlags.targetReleases[0], globalFlags.targetBuildVariant, globalFlags.useGetBuildVar, globalFlags.allowMissing, globalFlags.declarationsOnly)
	if err != nil {
		return err
	}
	err = getCommandInfo.run(configs, globalFlags, getArgs...)
	if err != nil {
		return err
	}
	fmt.Printf("\033[1mAdded/Updated: %s\033[0m\n", strings.Join(updatedFiles, " "))
	return nil
}

func main() {
	var globalFlags GlobalFlags
	var configs *rc_lib.ReleaseConfigs
	topDir, err := rc_lib.GetTopDir()

	// Handle the common arguments
	defaultVariant := os.Getenv("TARGET_BUILD_VARIANT")
	if defaultVariant == "" {
		defaultVariant = "eng"
	}

	flag.Usage = func() {
		cmdNames := rc_lib.SortedKeys(commandMap)
		helpHeader := "Usage:  build-flag [GLOBAL_OPTION...] COMMAND [COMMAND_OPTION...] FLAG_NAME [FLAG_NAME...]\n" +
			"Supported commands:\n" +
			func() string {
				var ret string
				for _, name := range cmdNames {
					ret += fmt.Sprintf("  %-6s %s\n", name, commandMap[name].helpText)
				}
				return ret
			}() +
			"\nSupported global options:\n"
		fmt.Fprintf(flag.CommandLine.Output(), helpHeader)
		flag.PrintDefaults()
		for _, cmdName := range cmdNames {
			cmd := commandMap[cmdName]
			fmt.Fprintf(flag.CommandLine.Output(), "\nSupported command options for command %q (%s):\n", cmdName, cmd.helpText)
			cmd.run(configs, globalFlags, "printdefaults")
		}
		// TODO: figure out if there is a good way to output releaseFlagSet.PrintDefaults().
	}
	flag.StringVar(&globalFlags.top, "top", topDir, "path to top of workspace")
	flag.BoolVar(&globalFlags.quiet, "quiet", false, "disable warning messages")
	flag.Var(&globalFlags.maps, "map", "path to a release_config_map.textproto. may be repeated")
	flag.StringVar(&globalFlags.mapsFile, "maps-file", "", "path to a file containing a list of release_config_map.textproto paths")
	flag.StringVar(&globalFlags.outDir, "out-dir", rc_lib.GetDefaultOutDir(), "basepath for the output. Multiple formats are created")
	flag.Var(&globalFlags.targetReleases, "release", "TARGET_RELEASE for this build")
	flag.StringVar(&globalFlags.targetBuildVariant, "variant", defaultVariant, "TARGET_BUILD_VARIANT for this build")
	flag.BoolVar(&globalFlags.allowMissing, "allow-missing", false, "Use trunk_staging values if release not found")
	flag.BoolVar(&globalFlags.allReleases, "all-releases", false, "operate on all releases. (Ignored for set command)")
	flag.BoolVar(&globalFlags.useGetBuildVar, "use-get-build-var", true, "use get_build_var PRODUCT_RELEASE_CONFIG_MAPS to get needed maps")
	flag.BoolVar(&globalFlags.debug, "debug", false, "turn on debugging output for errors")
	flag.BoolVar(&globalFlags.declarationsOnly, "declarations-only", false, "only process flag declarations")
	flag.Parse()

	if _, ok := commandMap[flag.Arg(0)]; !ok {
		fmt.Fprintf(os.Stderr, "Unsupported command %q\n", flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

	errorExit := func(err error) {
		if globalFlags.debug {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	if globalFlags.quiet {
		rc_lib.DisableWarnings()
	}

	if globalFlags.mapsFile != "" {
		if len(globalFlags.maps) > 0 {
			panic(fmt.Errorf("Cannot specify both --map and --maps-file"))
		}
		if err := globalFlags.maps.ReadFromFile(globalFlags.mapsFile); err != nil {
			panic(fmt.Errorf("Could not read %s", globalFlags.mapsFile))
		}
	}

	if len(globalFlags.targetReleases) == 0 {
		release, ok := os.LookupEnv("TARGET_RELEASE")
		if ok {
			globalFlags.targetReleases = rc_lib.StringList{release}
		} else {
			globalFlags.targetReleases = rc_lib.StringList{"trunk_staging"}
		}
	}

	if err = os.Chdir(globalFlags.top); err != nil {
		errorExit(err)
	}

	// Get the current state of flagging.
	relName := globalFlags.targetReleases[0]
	if relName == "--all" || relName == "-all" {
		globalFlags.allReleases = true
	}
	configs, err = rc_lib.ReadReleaseConfigMaps(globalFlags.maps, relName, globalFlags.targetBuildVariant, globalFlags.useGetBuildVar, globalFlags.allowMissing, globalFlags.declarationsOnly)
	if err != nil {
		errorExit(err)
	}

	if cmd, ok := commandMap[flag.Arg(0)]; ok {
		if err = cmd.run(configs, globalFlags, flag.Args()...); err != nil {
			errorExit(err)
		}
	}
}
