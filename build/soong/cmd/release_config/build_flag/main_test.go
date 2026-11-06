package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	rc_lib "android/soong/cmd/release_config/release_config_lib"
	rc_proto "android/soong/cmd/release_config/release_config_proto"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type testFileData struct {
	// The path components for the file.
	pathComps []string

	// The contents of the file.
	content string

	// The mode for the file.  If zero, then the file will be mode 644.
	mode os.FileMode
}

func getGlobalFlags(rcNames ...string) GlobalFlags {
	var targetReleases rc_lib.StringList
	if len(rcNames) == 0 {
		targetReleases.Set("trunk_staging")
	} else {
		for _, rcName := range rcNames {
			targetReleases.Set(rcName)
		}
	}
	return GlobalFlags{
		maps:               rc_lib.StringList{filepath.Join("build", "release", "release_config_map.textproto")},
		targetReleases:     targetReleases,
		targetBuildVariant: "eng",
		useGetBuildVar:     false,
	}
}

func createTestFiles(t *testing.T, tmpDir string, files []testFileData) {
	for _, f := range files {
		comps := []string{tmpDir}
		filePath := filepath.Join(append(comps, f.pathComps...)...)
		fileDir := filepath.Dir(filePath)
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			t.Fatalf("Failed to create %q: %v", fileDir, err)
		}
		mode := f.mode
		if mode == 0 {
			mode = 0644
		}
		if err := os.WriteFile(filePath, []byte(f.content), mode); err != nil {
			t.Fatalf("Failed to write %q: %v", filePath, err)
		}
	}
}

// setupTestEnv creates a temporary directory structure for testing.
// It returns the path to the temp directory and a cleanup function.
func setupTestEnv(t *testing.T, files []testFileData) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "build_flag_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	baseFiles := []testFileData{
		{
			pathComps: []string{"build", "release", "release_config_map.textproto"},
			content:   `default_containers: "system"`,
			mode:      0444,
		},
		{
			pathComps: []string{"build", "release", "flag_declarations", "RELEASE_TEST_FLAG1.textproto"},
			content:   `name: "RELEASE_TEST_FLAG1" namespace: "acme" workflow: MANUAL description: "A test flag." value: { string_value: "default_value" }`,
			mode:      0444,
		},
		{
			pathComps: []string{"build", "release", "flag_declarations", "RELEASE_TEST_FLAG2.textproto"},
			content:   `name: "RELEASE_TEST_FLAG2" namespace: "acme" workflow: MANUAL description: "Another test flag." value: { string_value: "default_value" }`,
			mode:      0444,
		},
		{
			pathComps: []string{"build", "release", "release_configs", "trunk_staging.textproto"},
			content:   `name: "trunk_staging" release_config_type: RELEASE_CONFIG`,
			mode:      0444,
		},
		{
			pathComps: []string{"build", "release", "release_configs", "trunk.textproto"},
			content:   `name: "trunk" release_config_type: RELEASE_CONFIG`,
			mode:      0444,
		},
		{
			pathComps: []string{"build", "release", "flag_values", "trunk_staging", "RELEASE_TEST_FLAG1.textproto"},
			content:   `name: "RELEASE_TEST_FLAG1" value: { string_value: "staging_override"}`,
			mode:      0644,
		},
	}

	createTestFiles(t, tmpDir, append(baseFiles, files...))

	// Change working directory to the temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	cleanup := func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func captureStdout(f func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return strings.TrimSpace(buf.String()), err
}

func TestGetCommand(t *testing.T) {
	_, cleanup := setupTestEnv(t, []testFileData{})
	defer cleanup()

	configs, err := rc_lib.ReadReleaseConfigMaps(getGlobalFlags().maps, "trunk_staging", "eng", false, false, false)
	if err != nil {
		t.Fatalf("Failed to read release configs: %v", err)
	}

	t.Run("Basic flag retrieval", func(t *testing.T) {
		getFlags := GetCommandFactory()
		globalFlags := getGlobalFlags()
		out, err := captureStdout(func() error {
			return getFlags(configs, globalFlags, "get", "RELEASE_TEST_FLAG1")
		})
		if err != nil {
			t.Fatalf("GetCommand failed: %v", err)
		}
		expected := "staging_override"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
	})

	t.Run("Multiple flag retrieval", func(t *testing.T) {
		getFlags := GetCommandFactory()
		globalFlags := getGlobalFlags()
		out, err := captureStdout(func() error {
			return getFlags(configs, globalFlags, "get", "RELEASE_TEST_FLAG1", "RELEASE_TEST_FLAG2")
		})
		if err != nil {
			t.Fatalf("GetCommand failed: %v", err)
		}
		expected := "RELEASE_TEST_FLAG1 'staging_override'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
		expected = "RELEASE_TEST_FLAG2 'default_value'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
	})

	t.Run("Multiple release config retrieval", func(t *testing.T) {
		getFlags := GetCommandFactory()
		globalFlags := getGlobalFlags("trunk_staging", "trunk")
		out, err := captureStdout(func() error {
			return getFlags(configs, globalFlags, "get", "RELEASE_TEST_FLAG1", "RELEASE_TEST_FLAG2")
		})
		if err != nil {
			t.Fatalf("GetCommand failed: %v", err)
		}
		expected := "RELEASE_TEST_FLAG1 trunk_staging 'staging_override'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
		expected = "RELEASE_TEST_FLAG1 trunk         'default_value'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
		expected = "RELEASE_TEST_FLAG2 trunk_staging 'default_value'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
		expected = "RELEASE_TEST_FLAG2 trunk         'default_value'"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
	})

	t.Run("Default value fallback", func(t *testing.T) {
		// Use a release without an override
		getFlags := GetCommandFactory()
		globalFlags := getGlobalFlags("trunk")
		out, err := captureStdout(func() error {
			return getFlags(configs, globalFlags, "get", "RELEASE_TEST_FLAG1")
		})
		if err != nil {
			t.Fatalf("GetCommand failed: %v", err)
		}
		expected := "default_value"
		if !strings.Contains(out, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, out)
		}
	})

	t.Run("Trace flag history", func(t *testing.T) {
		traceFlags := TraceCommandFactory()
		globalFlags := getGlobalFlags()
		out, err := captureStdout(func() error {
			return traceFlags(configs, globalFlags, "trace", "RELEASE_TEST_FLAG1")
		})
		if err != nil {
			t.Fatalf("TraceCommand failed: %v", err)
		}
		expectedOut := `staging_override
  => "default_value" in build/release/flag_declarations/RELEASE_TEST_FLAG1.textproto
  => "staging_override" in build/release/flag_values/trunk_staging/RELEASE_TEST_FLAG1.textproto`
		if expectedOut != out {
			t.Errorf("Bad trace: expected `%s` got `%s`", expectedOut, out)
		}
	})

}

func flagValueEqualOrError(t *testing.T, valuePath string, expectedValue *rc_proto.FlagValue) error {
	content, err := os.ReadFile(valuePath)
	if err != nil {
		return fmt.Errorf("Failed to read created flag file: %v", err)
	}

	actualValue := &rc_proto.FlagValue{}
	err = prototext.Unmarshal(content, actualValue)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal created flag file: %v", err)
	}

	if !proto.Equal(actualValue, expectedValue) {
		actual, _ := prototext.Marshal(actualValue)
		expected, _ := prototext.Marshal(expectedValue)
		return fmt.Errorf("File content mismatch: expected `%s` got `%s`", expected, actual)
	}
	return nil
}

func TestSetArg(t *testing.T) {
	testCases := []struct {
		Name     string
		Arg      string
		Expected map[string]string
	}{
		{
			Name:     "FLAG",
			Arg:      "FLAG",
			Expected: nil,
		},
		{
			Name: "FLAG_1=new_value",
			Arg:  "FLAG_1=new_value",
			Expected: map[string]string{
				"dir":      "",
				"flag":     "FLAG_1",
				"value":    "new_value",
				"redacted": "",
			},
		},
		{
			Name: "DIR/SUB:FLAG_2=VALUE",
			Arg:  "DIR/SUB:FLAG_2=VALUE",
			Expected: map[string]string{
				"dir":      "DIR/SUB",
				"flag":     "FLAG_2",
				"value":    "VALUE",
				"redacted": "",
			},
		},
		{
			Name: "DIR/35:FLAG_3:redacted",
			Arg:  "DIR/35:FLAG_3:redacted",
			Expected: map[string]string{
				"dir":      "DIR/35",
				"flag":     "FLAG_3",
				"value":    "",
				"redacted": "redacted",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			match := setArgRegexp.FindStringSubmatch(tc.Arg)
			if match == nil {
				if tc.Expected != nil {
					t.Errorf("Expected %q or %q, got %q", "[DIR:]FLAG=VALUE", "[DIR:]FLAG:redacted", tc.Arg)
				}
				return
			}
			reFields := make(map[string]string)
			for i, name := range setArgRegexp.SubexpNames() {
				if i != 0 && name != "" {
					reFields[name] = match[i]
				}
			}
			if !reflect.DeepEqual(reFields, tc.Expected) {
				t.Errorf("Expected %v, got %v", tc.Expected, reFields)
			}
		})
	}
}

func runSetCommand(t *testing.T, tmpDir string, rcName string, globFunc func(*GlobalFlags), args ...string) (string, error) {
	var myFileData []testFileData
	for _, f := range []testFileData{
		{
			pathComps: []string{"build", "release", "release_configs", rcName + ".textproto"},
			content:   `name: "` + rcName + `" release_config_type: RELEASE_CONFIG`,
			mode:      0444,
		},
	} {
		comps := []string{tmpDir}
		filePath := filepath.Join(append(comps, f.pathComps...)...)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			myFileData = append(myFileData, f)
		}
	}

	createTestFiles(t, tmpDir, myFileData)

	globalFlags := getGlobalFlags(rcName)
	if globFunc != nil {
		globFunc(&globalFlags)
	}
	configs, err := rc_lib.ReadReleaseConfigMaps(globalFlags.maps, rcName, "eng", true, false, false)
	if err != nil {
		return "", fmt.Errorf("Failed to read release configs: %v", err)
	}

	setCmd := SetCommandFactory()
	out, err := captureStdout(func() error {
		oldStderr := os.Stderr
		os.Stderr = nil
		defer func() { os.Stderr = oldStderr }()
		return setCmd(configs, globalFlags, args...)
	})

	if err != nil {
		return "", fmt.Errorf("SetCommand failed: %v", err)
	}
	return out, nil
}

func TestSetCommand(t *testing.T) {
	setFiles := []testFileData{
		{
			// new_release needs to be declared somewhere.
			pathComps: []string{"build", "release", "release_configs", "new_release.textproto"},
			content:   `name: "new_release" release_config_type: RELEASE_CONFIG`,
			mode:      0444,
		},
		{
			// The set command will create anything else that we need in build/release2.
			pathComps: []string{"build", "release2", "release_config_map.textproto"},
			content:   `default_containers: "system"`,
			mode:      0444,
		},
	}

	tmpDir, cleanup := setupTestEnv(t, setFiles)
	defer cleanup()

	t.Run("Set new flag value", func(t *testing.T) {
		rcName := "new_release"
		flagFunc := func(g *GlobalFlags) {
			g.maps.Set(filepath.Join("build", "release2", "release_config_map.textproto"))
		}
		_, err := runSetCommand(t, tmpDir, rcName, flagFunc, "set", "build/release2:RELEASE_TEST_FLAG1=new_value")
		if err != nil {
			t.Fatalf("%s", err)
		}

		// Verify the file was created
		valuePath := filepath.Join(tmpDir, "build", "release2", "flag_values", "new_release", "RELEASE_TEST_FLAG1.textproto")
		if _, err := os.Stat(valuePath); os.IsNotExist(err) {
			t.Fatalf("Expected flag value file to be created at %s, but it was not", valuePath)
		}

		expectedValue := &rc_proto.FlagValue{
			Name:  proto.String("RELEASE_TEST_FLAG1"),
			Value: &rc_proto.Value{Val: &rc_proto.Value_StringValue{"new_value"}},
		}

		if err = flagValueEqualOrError(t, valuePath, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}

		// Also verify the release config was created
		rcPath := filepath.Join(tmpDir, "build", "release2", "release_configs", "new_release.textproto")
		if _, err := os.Stat(rcPath); os.IsNotExist(err) {
			t.Fatalf("Expected release config file to be created at %s, but it was not", rcPath)
		}
	})

	t.Run("Overwrite existing value", func(t *testing.T) {
		rcName := "trunk_staging"
		_, err := runSetCommand(t, tmpDir, rcName, nil, "set", "RELEASE_TEST_FLAG1=overwritten")
		if err != nil {
			t.Fatalf("%s", err)
		}
		valuePath := filepath.Join(tmpDir, "build", "release", "flag_values", "trunk_staging", "RELEASE_TEST_FLAG1.textproto")
		expectedValue := &rc_proto.FlagValue{
			Name:  proto.String("RELEASE_TEST_FLAG1"),
			Value: &rc_proto.Value{Val: &rc_proto.Value_StringValue{"overwritten"}},
		}
		if err = flagValueEqualOrError(t, valuePath, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}
	})

	t.Run("Redact flag", func(t *testing.T) {
		rcName := "redacted_release"
		_, err := runSetCommand(t, tmpDir, rcName, nil, "set", "RELEASE_TEST_FLAG1:redacted")
		if err != nil {
			t.Fatalf("%s", err)
		}
		valuePath := filepath.Join(tmpDir, "build", "release", "flag_values", rcName, "RELEASE_TEST_FLAG1.textproto")
		expectedValue := &rc_proto.FlagValue{
			Name:     proto.String("RELEASE_TEST_FLAG1"),
			Redacted: proto.Bool(true),
		}
		if err = flagValueEqualOrError(t, valuePath, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}
	})

	t.Run("Set multiple flags", func(t *testing.T) {
		rcName := "multi_flags"
		_, err := runSetCommand(t, tmpDir, rcName, nil, "set", "RELEASE_TEST_FLAG1:redacted", "RELEASE_TEST_FLAG2=multi flags")
		if err != nil {
			t.Fatalf("%s", err)
		}
		valueDir := filepath.Join(tmpDir, "build", "release", "flag_values", rcName)
		flag1Path := filepath.Join(valueDir, "RELEASE_TEST_FLAG1.textproto")
		expectedValue := &rc_proto.FlagValue{
			Name:     proto.String("RELEASE_TEST_FLAG1"),
			Redacted: proto.Bool(true),
		}
		if err = flagValueEqualOrError(t, flag1Path, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}
		flag2Path := filepath.Join(valueDir, "RELEASE_TEST_FLAG2.textproto")
		expectedValue = &rc_proto.FlagValue{
			Name:  proto.String("RELEASE_TEST_FLAG2"),
			Value: &rc_proto.Value{Val: &rc_proto.Value_StringValue{"multi flags"}},
		}
		if err = flagValueEqualOrError(t, flag2Path, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}

	})

	t.Run("Legacy syntax", func(t *testing.T) {
		rcName := "legacy_flags"
		_, err := runSetCommand(t, tmpDir, rcName, nil, "set", "RELEASE_TEST_FLAG1", "new value")
		if err != nil {
			t.Fatalf("%s", err)
		}
		valuePath := filepath.Join(tmpDir, "build", "release", "flag_values", rcName, "RELEASE_TEST_FLAG1.textproto")
		expectedValue := &rc_proto.FlagValue{
			Name:  proto.String("RELEASE_TEST_FLAG1"),
			Value: &rc_proto.Value{Val: &rc_proto.Value_StringValue{"new value"}},
		}
		if err = flagValueEqualOrError(t, valuePath, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}

		_, err = runSetCommand(t, tmpDir, rcName, nil, "set", "--redacted", "RELEASE_TEST_FLAG1")
		if err != nil {
			t.Fatalf("%s", err)
		}
		expectedValue = &rc_proto.FlagValue{
			Name:     proto.String("RELEASE_TEST_FLAG1"),
			Redacted: proto.Bool(true),
		}
		if err = flagValueEqualOrError(t, valuePath, expectedValue); err != nil {
			t.Fatalf("%s", err)
		}
	})
}
