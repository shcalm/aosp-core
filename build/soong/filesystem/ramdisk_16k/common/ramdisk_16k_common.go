package common

import "encoding/json"

type Ramdisk16kImgPropertiesJSON struct {
	// List or filegroup of prebuilt kernel module files. Should have .ko suffix.
	Srcs []string `json:",omitempty"`

	// The zip of kernel modules from system_dlkm. Use `:module_name{.modules.zip}` here.
	// Modules in this zip will not be stripped, as stripping would remove the signature
	// of the kernel modules, and GKI modules must be signed for the kernel to load them.
	// https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/Makefile;l=1124;drc=a951ebf0198006f7fd38073a05c442d0eb92f97b
	System_dep *string `json:",omitempty"`

	// List specifying load order of kernel modules.
	Load []string `json:",omitempty"`

	// Path to the prebuilt 16KB kernel
	Kernel *string `json:",omitempty"`

	// properties for getting the kernel modules from a zip file.
	// Useful if the exact list of kernel modules to install isn't known at analysis time.
	Zip ZipProperties
}

type ZipProperties struct {
	// The zip file. Kernel modules will be looked up under the 16kb/ folder.
	Src *string `json:",omitempty"`

	// Names of modules that will be removed from the load file.
	Extra_blocked_modules []string `json:",omitempty"`
}

func (p *Ramdisk16kImgPropertiesJSON) ToJSON() string {
	result, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(result)
}
