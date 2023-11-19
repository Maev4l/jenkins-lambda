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
)

var (
	TMPDIR          = "/tmp"
	WORKDIR         = fmt.Sprintf("%s/workspace", TMPDIR)
	JENKINSHOMEDIR  = fmt.Sprintf("%s/jenkinshome", TMPDIR)
	RUNWORKSPACEDIR = fmt.Sprintf("%s/temp", TMPDIR)
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
		return errors.New("'respository' field not present in payload")
	}

	cloneUrl := gjson.Get(githubPayload, "repository.clone_url")
	commit := gjson.Get(githubPayload, "after")

	if cloneUrl.Type == gjson.Null || commit.Type == gjson.Null || cloneUrl.String() == "" || commit.String() == "" {
		return errors.New("'repository.clone_url' or after fields not present in payload")
	}

	log.Infof("Cloning %s@%s ...", cloneUrl, commit)

	err := gitClone(cloneUrl.String(), commit.String())
	if err != nil {
		return err
	}

	log.Info("Clone succeeded.")

	log.Info("Running Jenkinsfile ...")
	err = runJenkinsFile()
	if err != nil {
		return err
	}

	log.Info("Job completed.")
	return nil
}

func gitClone(cloneUrl string, commit string) error {
	_ = os.RemoveAll(WORKDIR)

	err := os.Mkdir(WORKDIR, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to created dir: %s", WORKDIR)
	}

	cmd := exec.Command("git", "clone", cloneUrl, WORKDIR)
	cmd.Dir = WORKDIR
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to clone repository '%s' into %s: %s", cloneUrl, WORKDIR, err.Error())
	}

	cmd = exec.Command("git", "checkout", commit)
	cmd.Dir = WORKDIR
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to checkout commit '%s': %s", commit, err.Error())
	}

	return nil
}

func runJenkinsFile() error {
	// TODO: Investigate on how to add custom plugins and check if
	// they have to be extracted into the /tmp/plugins folder

	_ = os.RemoveAll(JENKINSHOMEDIR)
	err := os.Mkdir(JENKINSHOMEDIR, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to created dir: %s", JENKINSHOMEDIR)
	}

	cmd := exec.Command("jenkinsfile-runner",
		"--jenkins-war",
		"/app/jenkins",
		"--plugins",
		"/usr/share/jenkins/ref/plugins",
		"--file",
		fmt.Sprintf("%s/Jenkinsfile", WORKDIR),
		"--runWorkspace",
		RUNWORKSPACEDIR,
		"--jenkinsHome",
		JENKINSHOMEDIR,
	)
	cmd.Dir = WORKDIR

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
