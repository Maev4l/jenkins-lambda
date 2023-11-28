package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

var (
	TMPDIR = "/tmp"
)

func handler(ctx context.Context, event events.SNSEvent) {
	record := event.Records[0]
	message := record.SNS.Message

	valid := gjson.Valid(message)
	if !valid {
		log.Error("Invalid Github payload")
		return
	}

	err := handleRequest(message)
	if err != nil {
		log.Error(err.Error())
		return
	}
}

func handleRequest(githubPayload string) error {

	repository := gjson.Get(githubPayload, "repository")
	if repository.Type == gjson.Null {
		return errors.New("respository field not present in payload")
	}

	cloneUrl := gjson.Get(githubPayload, "repository.clone_url")
	ref := gjson.Get(githubPayload, "ref")

	if cloneUrl.Type == gjson.Null || ref.Type == gjson.Null || cloneUrl.String() == "" || ref.String() == "" {
		return errors.New("repository.clone_url or ref fields not present in payload")
	}

	workDir := fmt.Sprintf("%s/workspace", TMPDIR)
	jenkinsHomeDir := fmt.Sprintf("%s/jenkinshome", TMPDIR)

	err := createFolderIfNotExists(jenkinsHomeDir)
	if err != nil {
		return err
	}

	err = createFolderIfNotExists(workDir)
	if err != nil {
		return err
	}

	log.Infof("Cloning %s ...", cloneUrl)

	err = gitClone(cloneUrl.String(), workDir)
	if err != nil {
		return err
	}

	log.Info("Clone succeeded.")

	log.Infof("Generating SCM configuration file ...")
	configFilename, err := generateScmConfigFile(cloneUrl.String(), ref.String(), workDir)
	if err != nil {
		return err
	}
	log.Infof("SCM configuration file (%s) generated.", configFilename)

	log.Info("Running Jenkinsfile ...")
	err = runJenkinsFile(workDir, jenkinsHomeDir, configFilename)
	if err != nil {
		return err
	}

	log.Info("Job completed.")
	return nil
}

func generateScmConfigFile(url string, ref string, jenkinsHomeDir string) (string, error) {
	c := RemoteConfig{Url: url}
	b := Branch{Name: ref}
	r := RootConfig{
		Scm: ScmConfig{
			Git: GitConfig{
				UserRemoteConfigs: make([]RemoteConfig, 0),
				Branches:          make([]Branch, 0),
				GitTool:           "/usr/bin/git",
			},
		},
	}
	r.Scm.Git.UserRemoteConfigs = append(r.Scm.Git.UserRemoteConfigs, c)
	r.Scm.Git.Branches = append(r.Scm.Git.Branches, b)
	data, err := yaml.Marshal(&r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Scm configuration to YAML: %s", err.Error())
	}

	scmConfigFileName := fmt.Sprintf("%s/scm.yaml", jenkinsHomeDir)

	err = os.WriteFile(scmConfigFileName, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write %s file: %s", scmConfigFileName, err.Error())
	}

	return scmConfigFileName, nil
}

func gitClone(cloneUrl string, folder string) error {

	cmd := exec.Command("git", "clone", cloneUrl, folder)
	cmd.Dir = folder
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to clone repository '%s' into %s: %s", cloneUrl, folder, err.Error())
	}

	return nil
}

func createFolderIfNotExists(folder string) error {
	_ = os.RemoveAll(folder)
	err := os.Mkdir(folder, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to created dir: %s", folder)
	}
	return nil
}

func runJenkinsFile(workDir string, jenkinsHomeDir string, scmConfigFile string) error {
	// TODO: Investigate on how to add custom plugins and check if
	// they have to be extracted into the /tmp/plugins folder

	runWorkspaceDir := fmt.Sprintf("%s/temp", TMPDIR)
	cmd := exec.Command("jenkinsfile-runner",
		"--jenkins-war",
		"/app/jenkins",
		"--plugins",
		"/usr/share/jenkins/ref/plugins",
		"--file",
		fmt.Sprintf("%s/Jenkinsfile", workDir),
		"--runWorkspace",
		runWorkspaceDir,
		"--jenkinsHome",
		jenkinsHomeDir,
		"--scm",
		scmConfigFile,
	)
	cmd.Dir = workDir

	log.Infof("Command: %s", cmd.String())

	out, err := cmd.CombinedOutput()
	fmt.Printf("%s", string(out))

	if err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
