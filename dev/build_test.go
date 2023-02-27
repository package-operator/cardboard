package dev

import (
	"github.com/magefile/mage/mg"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

type buildImgTestCase struct {
	name      string
	buildInfo ImageBuildInfo
	commands  [][]string
}

type buildPkgTestCase struct {
	name      string
	buildInfo PackageBuildInfo
	commands  [][]string
}

type pushTestCase struct {
	name     string
	pushInfo ImagePushInfo
	commands [][]string
}

var (
	defaultBuildImgCase = buildImgTestCase{
		name: "default",
		buildInfo: ImageBuildInfo{
			ImageTag:      "test_ImageTag",
			CacheDir:      "",
			ContainerFile: "test_ContainerFile",
			ContextDir:    "test_ContextDir",
			Runtime:       "test_Runtime",
		},
		commands: [][]string{
			{"test_Runtime", "build", "-t", "test_ImageTag", "-f", "test_ContainerFile", "test_ContextDir"},
			{"test_Runtime", "image", "save", "-o", ".tar", "test_ImageTag"},
		},
	}

	noConFileBuildImgCase = buildImgTestCase{
		name: "no-container-file",
		buildInfo: ImageBuildInfo{
			ImageTag:      "test_ImageTag",
			CacheDir:      "",
			ContainerFile: "",
			ContextDir:    "test_ContextDir",
			Runtime:       "test_Runtime",
		},
		commands: [][]string{
			{"test_Runtime", "build", "-t", "test_ImageTag", "test_ContextDir"},
			{"test_Runtime", "image", "save", "-o", ".tar", "test_ImageTag"},
		},
	}

	defaultBuildPkgCase = buildPkgTestCase{
		name: "default",
		buildInfo: PackageBuildInfo{
			ImageTag:   "test_ImageTag",
			CacheDir:   "",
			SourcePath: "test_SourcePath",
			OutputPath: "test_OutputPath",
			Runtime:    "test_Runtime",
		},
		commands: [][]string{
			{"kubectl-package", "build", "--tag", "test_ImageTag", "--output", "test_OutputPath", "test_SourcePath"},
			{"test_Runtime", "load", "--input", "test_OutputPath"},
		},
	}

	defaultPushCase = pushTestCase{
		name: "default",
		pushInfo: ImagePushInfo{
			ImageTag:   "test_ImageTag",
			CacheDir:   "",
			Runtime:    "test_Runtime",
			DigestFile: "test_DigestFile",
		},
		commands: [][]string{
			{"test_Runtime", "push", "test_ImageTag"},
			{"test_Runtime", "login", "-u=" + os.Getenv("QUAY_USER"), "-p=" + os.Getenv("QUAY_TOKEN"), "quay.io"},
		},
	}

	podmanPushCase = pushTestCase{
		name: "podman",
		pushInfo: ImagePushInfo{
			ImageTag:   "test_ImageTag",
			CacheDir:   "",
			Runtime:    string(ContainerRuntimePodman),
			DigestFile: "test_DigestFile",
		},
		commands: [][]string{
			{string(ContainerRuntimePodman), "push", "--digestfile=test_DigestFile", "test_ImageTag"},
			{string(ContainerRuntimePodman), "login", "-u=" + os.Getenv("QUAY_USER"), "-p=" + os.Getenv("QUAY_TOKEN"), "quay.io"},
		},
	}

	buildImgTestCases = map[string]*buildImgTestCase{
		"default":           &defaultBuildImgCase,
		"no-container-file": &noConFileBuildImgCase,
	}

	buildPkgTestCases = map[string]*buildPkgTestCase{
		"default": &defaultBuildPkgCase,
	}

	pushTestCases = map[string]*pushTestCase{
		"default": &defaultPushCase,
		"podman":  &podmanPushCase,
	}

	// currentTestCase is used in TestXXXX_HelperProcess to identify which test ran it
	currentTestCase string

	// helperProcess is used by mockExecCommand to determine which helper process to run
	helperProcess string
)

const (
	buildImgHelper = "TestBuildImage_HelperProcess"
	buildPkgHelper = "TestBuildPackage_HelperProcess"
	pushHelper     = "TestPushImage_HelperProcess"
)

func matchCommands(cmd1, cmd2 []string) bool {
	if len(cmd1) != len(cmd2) {
		return false
	}
	for i, arg := range cmd1 {
		if arg != cmd2[i] {
			return false
		}
	}
	return true
}

func matchAtLeastOne(cmd []string, cmds [][]string) bool {
	for _, c := range cmds {
		if matchCommands(c, cmd) {
			return true
		}
	}
	return false
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=" + helperProcess, "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{
		"GO_WANT_HELPER_PROCESS=1",
		"GO_TEST_CASE_NAME=" + currentTestCase,
	}
	return cmd
}

func TestBuildImage_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	tc := buildImgTestCases[os.Getenv("GO_TEST_CASE_NAME")]
	command := os.Args[3:]
	if !matchAtLeastOne(command, tc.commands) {
		t.Fatalf("invalid command: %v", command)
	}
	os.Exit(0)
}

func TestBuildImage(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	helperProcess = buildImgHelper

	for _, tc := range buildImgTestCases {
		currentTestCase = tc.name
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, BuildImage(&tc.buildInfo, []interface{}{}))
		})
	}
}

func TestBuildPackage_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	tc := buildPkgTestCases[os.Getenv("GO_TEST_CASE_NAME")]
	command := os.Args[3:]
	if !matchAtLeastOne(command, tc.commands) {
		t.Fatalf("invalid command: %v", command)
	}
	os.Exit(0)
}

func TestBuildPackage(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	helperProcess = buildPkgHelper

	for _, tc := range buildPkgTestCases {
		currentTestCase = tc.name
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, BuildPackage(&tc.buildInfo, []interface{}{}))
		})
	}
}

func TestPushImage_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	tc := pushTestCases[os.Getenv("GO_TEST_CASE_NAME")]
	command := os.Args[3:]
	if !matchAtLeastOne(command, tc.commands) {
		t.Fatalf("invalid command: %v", command)
	}
	os.Exit(0)
}

func TestPushImage(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	helperProcess = pushHelper

	for _, tc := range pushTestCases {
		currentTestCase = tc.name
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, PushImage(&tc.pushInfo, mg.F(func() {})))
		})
	}
}
