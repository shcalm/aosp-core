package main

import (
	"android/soong/filesystem/ramdisk_16k/common"
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var (
	//soongZip  = flag.String("soong_zip", "", "path to soong_zip executable")
	//zipSync   = flag.String("zipsync", "", "path to zipsync executable")
	//mergeZips = flag.String("merge_zips", "", "path to merge_zips executable")
	depmod        = flag.String("depmod", "", "path to depmod executable")
	llvmStrip     = flag.String("llvm-strip", "", "path to llvm-strip executable")
	extractKernel = flag.String("extract_kernel", "", "path to extract_kernel executable")
	lz4           = flag.String("lz4", "", "path to lz4 executable")
	mkbootfs      = flag.String("mkbootfs", "", "path to mkbootfs executable")
	tempDir       string
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kernel_modules_builder <props.json> <temp dir> <out img>")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) != 3 {
		flag.Usage()
		os.Exit(1)
	}

	propsFile := flag.Arg(0)
	tempDir = flag.Arg(1)
	out := flag.Arg(2)

	must(os.RemoveAll(tempDir))
	must(os.MkdirAll(tempDir, 0o777))

	var props common.Ramdisk16kImgPropertiesJSON
	must(json.Unmarshal(must2(os.ReadFile(propsFile)), &props))

	kernelRelease := cmdOutput(*extractKernel, "--tools", "lz4:"+*lz4, "--input", *props.Kernel, "--output-release")
	kernelConfigs := cmdOutput(*extractKernel, "--tools", "lz4:"+*lz4, "--input", *props.Kernel, "--output-config")
	if strings.Contains(kernelConfigs, "CONFIG_ARM64_16K_PAGES=y") {
		kernelRelease += "_16k"
	}

	intermediatesDir := tempDir
	modulesDir := filepath.Join(intermediatesDir, "lib", "modules", kernelRelease)
	must(os.MkdirAll(modulesDir, 0o777))

	if props.Zip.Src != nil {
		zipLogic(&props, modulesDir)
	} else {
		// Copy the .ko files and modules.load to a staging directory.
		// Kernel version is one of the path components of the staging directory.
		for _, src := range props.Srcs {
			cp(src, filepath.Join(modulesDir, filepath.Base(src)))
		}
		createLoadFile(&props, filepath.Join(modulesDir, "modules.load"))
	}

	strip(&props, modulesDir)

	// Run depmod.
	// This implementation is slightly different than make, which first copies the .ko
	// files to lib/modules/0.0, runs depmod, and then does a recursive cp to the final
	// staging directory with kernel version as one of the path components.
	cmd(*depmod, "-b", intermediatesDir, kernelRelease)

	mkLz4Bootfs(intermediatesDir, out)
}

func mkLz4Bootfs(dir, out string) {
	mkbootfsCmd := exec.Command(*mkbootfs, dir)
	lz4Command := exec.Command(*lz4, "-l", "-12", "--favor-decSpeed", "-", out)

	mkbootfsCmd.Stderr = os.Stderr
	lz4Command.Stderr = os.Stderr
	lz4Command.Stdin = must2(mkbootfsCmd.StdoutPipe())
	lz4Command.Stdout = os.Stdout

	must(mkbootfsCmd.Start())
	must(lz4Command.Start())
	must(lz4Command.Wait())
	must(mkbootfsCmd.Wait())
}

func strip(props *common.Ramdisk16kImgPropertiesJSON, modulesDir string) {
	excluded := make(map[string]struct{})
	if props.System_dep != nil {
		f := must2(os.Open(*props.System_dep))
		defer f.Close()
		reader := must2(zip.NewReader(f, must2(f.Stat()).Size()))
		for _, f := range reader.File {
			// only look at files in the root of the zip
			if strings.Contains(f.Name, "/") {
				continue
			}
			excluded[f.Name] = struct{}{}
		}
	}
	for _, src := range props.Srcs {
		if _, ok := excluded[filepath.Base(src)]; ok {
			continue
		}

		cmd(*llvmStrip, "--strip-debug", filepath.Join(modulesDir, filepath.Base(src)))
	}
}

func zipLogic(props *common.Ramdisk16kImgPropertiesJSON, modulesDir string) {
	f := must2(os.Open(*props.Zip.Src))
	defer f.Close()
	reader := must2(zip.NewReader(f, must2(f.Stat()).Size()))

	var loads []string

	processLoadFile := func(file string) {
		for _, line := range strings.Split(string(readZipFile(reader, file)), "\n") {
			if line == "" {
				continue
			}
			loads = append(loads, filepath.Base(line))
		}
	}
	processLoadFile("16kb/vendor_kernel_boot.modules.load")
	processLoadFile("16kb/system_dlkm.modules.load")
	processLoadFile("16kb/vendor_dlkm.modules.load")

	blocks := make(map[string]struct{})
	processBlocklistFile := func(file string) {
		for _, blocked := range strings.Fields(string(readZipFile(reader, file))) {
			if blocked == "blocklist" {
				continue
			}
			// some entries have .ko, some don't.
			if !strings.HasSuffix(blocked, ".ko") {
				blocked += ".ko"
			}
			blocks[blocked] = struct{}{}
			blocks[strings.ReplaceAll(blocked, "_", "-")] = struct{}{}
		}
	}
	processBlocklistFile("16kb/system_dlkm.modules.blocklist")
	processBlocklistFile("16kb/vendor_dlkm.modules.blocklist")

	var filteredLoads []string
	for _, load := range loads {
		if _, ok := blocks[load]; ok {
			continue
		}
		if slices.Contains(props.Zip.Extra_blocked_modules, load) {
			continue
		}

		extractZipFile(reader, "16kb/"+load, filepath.Join(modulesDir, load))
		filteredLoads = append(filteredLoads, load)
	}

	os.WriteFile(filepath.Join(modulesDir, "modules.load"), []byte(strings.Join(filteredLoads, "\n")+"\n"), 0o666)
}

func createLoadFile(props *common.Ramdisk16kImgPropertiesJSON, out string) {
	var loadOrder []string
	if len(props.Load) > 0 {
		loadOrder = props.Load
	} else {
		for _, src := range props.Srcs {
			loadOrder = append(loadOrder, filepath.Base(src))
		}
	}

	os.WriteFile(out, []byte(strings.Join(loadOrder, "\n")+"\n"), 0o666)
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func must2[T any](x T, err error) T {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return x
}

func readZipFile(zipFile *zip.Reader, pathInZip string) []byte {
	in := must2(zipFile.Open(pathInZip))
	defer in.Close()
	return must2(io.ReadAll(in))
}

func extractZipFile(zipFile *zip.Reader, pathInZip string, outputPath string) {
	in := must2(zipFile.Open(pathInZip))
	defer in.Close()
	out := must2(os.Create(outputPath))
	defer out.Close()
	must2(io.Copy(out, in))
}

func cp(from string, to string) {
	cmd("cp", from, to)
}

func cmd(tool string, args ...string) {
	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
}

func cmdOutput(tool string, args ...string) string {
	cmd := exec.Command(tool, args...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	must(cmd.Run())
	return strings.TrimSpace(b.String())
}
