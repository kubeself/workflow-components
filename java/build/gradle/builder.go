package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	baseSpace  = "/root/src"
	cacheSpace = "/workflow-cache"
)

type Builder struct {
	// 用户提供参数, 通过环境变量传入
	GitCloneURL string
	GitRef      string
	EntryFile   string

	HubRepo      string
	HubUser      string
	HubToken     string
	ArtifactTag  string
	ArtifactPath string

	projectName string

	WorkflowCache bool
	workDir       string
	gitDir        string
}

// NewBuilder is
func NewBuilder(envs map[string]string) (*Builder, error) {
	b := &Builder{}

	if envs["GIT_CLONE_URL"] != "" {
		b.GitCloneURL = envs["GIT_CLONE_URL"]
		b.GitRef = envs["GIT_REF"]
	} else if envs["_WORKFLOW_GIT_CLONE_URL"] != "" {
		b.GitCloneURL = envs["_WORKFLOW_GIT_CLONE_URL"]
		b.GitRef = envs["_WORKFLOW_GIT_REF"]
	} else {
		return nil, fmt.Errorf("envionment variable GIT_CLONE_URL is required")
	}

	if b.GitRef == "" {
		b.GitRef = "master"
	}

	s := strings.TrimSuffix(strings.TrimSuffix(b.GitCloneURL, "/"), ".git")
	b.projectName = s[strings.LastIndex(s, "/")+1:]

	if b.GitRef = envs["GIT_REF"]; b.GitRef == "" {
		b.GitRef = "master"
	}

	if b.EntryFile = envs["ENTRY_FILE"]; b.EntryFile == "" {
		b.EntryFile = "./build.gradle"
	}

	b.HubUser = envs["HUB_USER"]
	b.HubToken = envs["HUB_TOKEN"]

	if b.HubUser == "" && b.HubToken == "" {
		b.HubUser = envs["_WORKFLOW_HUB_USER"]
		b.HubToken = envs["_WORKFLOW_HUB_TOKEN"]
	}
	if b.HubUser == "" || b.HubToken == "" {
		return nil, fmt.Errorf("envionment variable HUB_USER, HUB_TOKEN are required")
	}

	b.HubRepo = envs["HUB_REPO"]
	b.ArtifactPath = envs["ARTIFACT_PATH"]
	b.ArtifactTag = envs["ARTIFACT_TAG"]
	if b.ArtifactTag == "" {
		b.ArtifactTag = "latest"
	}

	b.WorkflowCache = strings.ToLower(envs["_WORKFLOW_FLAG_CACHE"]) == "true"

	if b.WorkflowCache {
		b.workDir = cacheSpace
	} else {
		b.workDir = baseSpace
	}
	b.gitDir = filepath.Join(b.workDir, b.projectName)

	return b, nil
}

func (b *Builder) run() error {
	if err := os.Chdir(b.workDir); err != nil {
		return fmt.Errorf("chdir to workdir (%s) failed:%v", b.workDir, err)
	}

	if _, err := os.Stat(b.gitDir); os.IsNotExist(err) {
		if err := b.gitPull(); err != nil {
			return err
		}

		if err := b.gitReset(); err != nil {
			return err
		}
	}

	if err := b.build(); err != nil {
		return err
	}

	if err := b.handleArtifacts(); err != nil {
		return err
	}
	return nil
}

func (b *Builder) build() error {
	var command = []string{"gradle", "jar"}

	command = append(command, "-b", b.EntryFile)

	cwd, _ := os.Getwd()
	if _, err := (CMD{command, b.gitDir}).Run(); err != nil {
		fmt.Println("Run gradle jar failed:", err)
		return err
	}
	fmt.Println("Run gradle jar succeed.")
	return nil
}

func (b *Builder) handleArtifacts() error {
	command := []string{"find", "./", "-name", "*.jar"}
	output, err := (CMD{command, b.gitDir}).Run()
	if err != nil {
		fmt.Println("Run find artifacts failed:", err)
		return err
	}
	output = strings.TrimSpace(output)
	if len(output) == 0 {
		return errors.New("no artifact")
	}

	artifactsSlice := strings.Split(output, "\n")
	fmt.Printf("[JOB_OUT] ARTIFACT = %s\n", strings.Join(artifactsSlice, ";"))

	if b.HubRepo == "" {
		fmt.Println("HUB_REPO is empty, no need upload artifacts")
		return nil
	}

	artifactsTar := fmt.Sprintf("%s.tar.bz", b.projectName)

	command = []string{"tar", "-cjf", artifactsTar}
	command = append(command, artifactsSlice...)
	if _, err := (CMD{command, b.gitDir}).Run(); err != nil {
		fmt.Println("Run tar artifacts failed:", err)
		return err
	}

	command = []string{
		"/.workflow/bin/thub", "push",
		fmt.Sprintf("--username=%s", b.HubUser), fmt.Sprintf("--password=%s", b.HubToken),
		fmt.Sprintf("--repo=%s", b.HubRepo),
		fmt.Sprintf("--localpath=%s", artifactsTar),
		fmt.Sprintf("--path=%s", filepath.Join(b.ArtifactPath, artifactsTar)),
		fmt.Sprintf("--tag=%s", b.ArtifactTag),
	}
	if _, err := (CMD{command, b.gitDir}).Run(); err != nil {
		fmt.Println("Run upload artifacts failed:", err)
		return err
	}

	// TODO
	fmt.Printf("[JOB_OUT] ARTIFACT_URL = %s\n", filepath.Join(b.HubRepo, b.ArtifactPath, artifactsTar))
	fmt.Println("Run upload artifacts succeed.")
	return nil
}

func (b *Builder) gitPull() error {
	var command = []string{"git", "clone", "--recurse-submodules", b.GitCloneURL, b.projectName}
	if _, err := (CMD{Command: command}).Run(); err != nil {
		fmt.Println("Clone project failed:", err)
		return err
	}
	fmt.Println("Clone project", b.GitCloneURL, "succeed.")
	return nil
}

func (b *Builder) gitReset() error {
	cwd, _ := os.Getwd()
	var command = []string{"git", "checkout", b.GitRef, "--"}
	if _, err := (CMD{command, b.gitDir}).Run(); err != nil {
		fmt.Println("Switch to commit", b.GitRef, "failed:", err)
		return err
	}
	fmt.Println("Switch to", b.GitRef, "succeed.")
	return nil
}

type CMD struct {
	Command []string // cmd with args
	WorkDir string
}

func (c CMD) Run() (string, error) {
	fmt.Println("Run CMD: ", strings.Join(c.Command, " "))

	cmd := exec.Command(c.Command[0], c.Command[1:]...)
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}

	data, err := cmd.CombinedOutput()
	result := string(data)
	if len(result) > 0 {
		fmt.Println(result)
	}

	return result, err
}
