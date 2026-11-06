load("@builtin//path.star", "path")
load("@builtin//struct.star", "module")

def __filegroups(ctx, vars):
    if vars.use_reclient:
        return {}
    version = vars.RELEASE_BUILD_CLANG_VERSION
    fg = {
        "prebuilts/clang/host/linux-x86/" + version + "/bin:bin": {
            "type": "glob",
            "includes": [
                "clang*",  # clang, clang++, clang-<ver> clang-real, clang++-real, clang++.real etc.
            ],
        },
        "prebuilts/clang/host/linux-x86/" + version + "/include:include": {
            "type": "glob",
            "includes": [
                "*",
            ],
        },
        "prebuilts/clang/host/linux-x86/" + version + "/lib:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "*_ignorelist.txt",
            ],
        },
        "prebuilts/clang/host/linux-x86/" + version + "/android_libc++/ndk:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "*/include/c++/v1/*",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include:include": {
            "type": "glob",
            "includes": [
                "*",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/lib:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "crtbegin.o",
            ],
        },
        "bionic/libc/include:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "bionic/libc/kernel/uapi:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "bionic/libc/kernel/android/uapi:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/include:include": {
            "type": "glob",
            "includes": ["*"],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib:lib": {
            "type": "glob",
            "includes": ["crtbegin.o"],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib32:lib32": {
            "type": "glob",
            "includes": ["crtbegin.o"],
        },
        "external/boringssl/src/include:include": {
            "type": "glob",
            "includes": ["*.h"],
        },
        # prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include includes e.g. linux/rtnetlink.h
        "external/libbpf/include/uapi:headers": {
            "type": "glob",
            "includes": ["*.h"],
        },
    }
    return fg

__handlers = {}

def __step_config(ctx, vars, step_config):
    if vars.use_reclient:
        step_config["rules"].extend([
            {
                "name": "g.cc.cc",
                "action": "g.cc.cc",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
                # "debug": True,
            },
        ])
        return step_config

    version = vars.RELEASE_BUILD_CLANG_VERSION
    step_config["rules"].extend([
        {
            "name": "g.cc.cc",
            "action": "g.cc.cc",
            "remote": True,
            "canonicalize_dir": True,
            "timeout": "2m",
            # "debug": True,
        },
    ])
    step_config["input_deps"].update({
        "prebuilts/clang/host/linux-x86/" + version + ":headers": [
            "prebuilts/clang/host/linux-x86/" + version + "/android_libc++/ndk:headers",
            "prebuilts/clang/host/linux-x86/" + version + "/bin:bin",
            "prebuilts/clang/host/linux-x86/" + version + "/include:include",
            "prebuilts/clang/host/linux-x86/" + version + "/lib:headers",
        ],
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot:headers": [
            "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include:include",
            "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/lib:headers",
        ],
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32:headers": [
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/include:include",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib:lib",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib32:lib32",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib64",
        ],
        # The NDK sysroot is generated mid-build, so its headers are not available
        # for dependency scanning at the start of the build. The dependency is
        # handled via a timestamp file instead. See b/433611110 for context.
        path.join(vars.OUT_DIR, "soong/ndk/sysroot:headers"): [
            path.join(vars.OUT_DIR, "soong/ndk_headers.timestamp") + ":inputs",
        ],

        # asm include
        "device/google/cuttlefish/host/libs/confui/fonts.S": [
            # .incbin
            "device/google/cuttlefish/host/libs/confui/Roboto-Medium.ttf",
            "device/google/cuttlefish/host/libs/confui/Roboto-Regular.ttf",
            "device/google/cuttlefish/host/libs/confui/Shield.ttf",
        ],
    })
    step_config["clang_scandeps"] = "unsupported-macro"
    step_config["inputs_requiring_clang_scandeps"].extend([
        # #define RELOC_TYPES STRINGIFIED_PASTE (BACKEND, reloc.def)
        # #include RELOC_TYPES
        "external/elfutils/backends/common-reloc.c",

        # #if __has_include(<android/binder_ibinder.h>)
        "frameworks/native/libs/binder/ndk/include_cpp/android/binder_to_string.h",
    ])
    return step_config

clang = module(
    "clang",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
)
