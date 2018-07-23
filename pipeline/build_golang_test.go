package pipeline

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gaia-pipeline/gaia"
	"github.com/gaia-pipeline/gaia/store"
	hclog "github.com/hashicorp/go-hclog"
)

func TestPrepareEnvironmentGo(t *testing.T) {
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	err := b.PrepareEnvironment(p)
	if err != nil {
		t.Fatal("error was not expected when preparing environment: ", err)
	}
	var expectedDest = regexp.MustCompile(`^/.*/tmp/golang/src/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !expectedDest.MatchString(p.Pipeline.Repo.LocalDest) {
		t.Fatalf("expected destination is '%s', but was '%s'", expectedDest, p.Pipeline.Repo.LocalDest)
	}
}

func TestPrepareEnvironmentInvalidPathForMkdirGo(t *testing.T) {
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = "/notexists"
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	err := b.PrepareEnvironment(p)
	if err == nil {
		t.Fatal("error was expected but none occurred")
	}
}

func TestExecuteBuildGo(t *testing.T) {
	execCommandContext = fakeExecCommandContext
	defer func() {
		execCommandContext = exec.CommandContext
	}()
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	err := b.ExecuteBuild(p)
	if err != nil {
		t.Fatal("error while running executebuild. none was expected")
	}
	expectedDepArgs := "get,-d,./..."
	expectedBuildArgs := "build,-o,_"
	actualArgs := os.Getenv("CMD_ARGS")
	if !strings.Contains(actualArgs, expectedBuildArgs) && !strings.Contains(actualArgs, expectedDepArgs) {
		t.Fatalf("expected args '%s, %s' actual args '%s'", expectedDepArgs, expectedBuildArgs, actualArgs)
	}
}

func TestExecuteBuildFailPipelineBuildGo(t *testing.T) {
	os.Mkdir("tmp", 0744)
	ioutil.WriteFile(filepath.Join("tmp", "main.go"), []byte(`package main
		import "os"
		func main() {
			os.Exit(1
		}`), 0766)
	wd, _ := os.Getwd()
	tmp := filepath.Join(wd, "tmp")
	defer func() {
		os.RemoveAll(tmp)
	}()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	buf := new(bytes.Buffer)
	gaia.Cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: buf,
		Name:   "Gaia",
	})
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	p.Pipeline.Repo.LocalDest = tmp
	err := b.ExecuteBuild(p)
	if err == nil {
		t.Fatal("error while running executebuild. none was expected")
	}
	expected := "syntax error: unexpected newline, expecting comma or )"
	if !strings.Contains(p.Output, expected) {
		t.Fatal("got a different output than expected: ", p.Output)
	}
}

func TestExecuteBuildContextTimeoutGo(t *testing.T) {
	execCommandContext = fakeExecCommandContext
	killContext = true
	defer func() {
		execCommandContext = exec.CommandContext
	}()
	defer func() { killContext = false }()
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	// Initialize shared logger
	buf := new(bytes.Buffer)
	gaia.Cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: buf,
		Name:   "Gaia",
	})
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	err := b.ExecuteBuild(p)
	if err == nil {
		t.Fatal("no error found while expecting error.")
	}
	if err.Error() != "context deadline exceeded" {
		t.Fatal("context deadline should have been exceeded. was instead: ", err)
	}
}

func TestExecuteBuildBinaryNotFoundErrorGo(t *testing.T) {
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	// Initialize shared logger
	buf := new(bytes.Buffer)
	gaia.Cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: buf,
		Name:   "Gaia",
	})
	currentPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", currentPath) }()
	os.Setenv("PATH", "")
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	err := b.ExecuteBuild(p)
	if err == nil {
		t.Fatal("no error found while expecting error.")
	}
	if err.Error() != "exec: \"go\": executable file not found in $PATH" {
		t.Fatal("the error wasn't what we expected. instead it was: ", err)
	}
}

func TestCopyBinaryGo(t *testing.T) {
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	// Initialize shared logger
	buf := new(bytes.Buffer)
	gaia.Cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: buf,
		Name:   "Gaia",
	})
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	p.Pipeline.Name = "main"
	p.Pipeline.Type = "go"
	p.Pipeline.Repo.LocalDest = tmp
	src := filepath.Join(tmp, appendTypeToName(p.Pipeline.Name, p.Pipeline.Type))
	dst := appendTypeToName(p.Pipeline.Name, p.Pipeline.Type)
	f, _ := os.Create(src)
	defer f.Close()
	defer os.Remove(dst)
	ioutil.WriteFile(src, []byte("testcontent"), 0666)
	err := b.CopyBinary(p)
	if err != nil {
		t.Fatal("error was not expected when copying binary: ", err)
	}
	content, err := ioutil.ReadFile(dst)
	if err != nil {
		t.Fatal("error encountered while reading destination file: ", err)
	}
	if string(content) != "testcontent" {
		t.Fatal("file content did not equal src content. was: ", string(content))
	}
}

func TestCopyBinarySrcDoesNotExistGo(t *testing.T) {
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	// Initialize shared logger
	buf := new(bytes.Buffer)
	gaia.Cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: buf,
		Name:   "Gaia",
	})
	b := new(BuildPipelineGolang)
	p := new(gaia.CreatePipeline)
	p.Pipeline.Name = "main"
	p.Pipeline.Type = "go"
	p.Pipeline.Repo.LocalDest = "/noneexistent"
	err := b.CopyBinary(p)
	if err == nil {
		t.Fatal("error was expected when copying binary but none occurred ")
	}
	if err.Error() != "open /noneexistent/main_go: no such file or directory" {
		t.Fatal("a different error occurred then expected: ", err)
	}
}

func TestSavePipelineGo(t *testing.T) {
	tmp := os.TempDir()
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = tmp
	defer os.Remove(tmp)
	s := store.NewStore()
	s.Init()
	storeService = s
	defer os.Remove("gaia.db")
	gaia.Cfg = new(gaia.Config)
	gaia.Cfg.HomePath = "/tmp"
	gaia.Cfg.PipelinePath = "/tmp/pipelines/"
	// Initialize shared logger
	p := new(gaia.Pipeline)
	p.Name = "main"
	p.Type = gaia.PTypeGolang
	b := new(BuildPipelineGolang)
	err := b.SavePipeline(p)
	if err != nil {
		t.Fatal("something went wrong. wasn't supposed to get error: ", err)
	}
	if p.Name != "main" {
		t.Fatal("name of pipeline didn't equal expected 'main'. was instead: ", p.Name)
	}
	if p.Type != gaia.PTypeGolang {
		t.Fatal("type of pipeline was not go. instead was: ", p.Type)
	}
}
