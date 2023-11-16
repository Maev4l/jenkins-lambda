package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var (
	TMPDIR         = "/tmp"
	WORKDIR        = fmt.Sprintf("%s/workspace", TMPDIR)
	JENKINSHOMEDIR = fmt.Sprintf("%s/jenkinshome", TMPDIR)
)

type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

func apiResponse(status int, body interface{}) *events.APIGatewayProxyResponse {
	resp := events.APIGatewayProxyResponse{Headers: map[string]string{
		"Content-Type": "application/json",
	}}

	resp.StatusCode = status
	if body != nil {
		stringBody, _ := json.Marshal(body)

		resp.Body = string(stringBody)
	}

	return &resp
}

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	valid := gjson.Valid(event.Body)
	if !valid {
		message := "Invalid Github payload"
		log.Error(message)
		return *apiResponse(500, Error{Message: message}), nil
	}

	err := handleRequest(event.Body)
	if err != nil {
		log.Error(err.Message)
		return *apiResponse(500, err), nil
	}

	log.Info("So far so good !!")

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "\"Hello from Lambda!\"",
	}
	return response, nil
}

func handleRequest(githubPayload string) *Error {

	repository := gjson.Get(githubPayload, "repository")
	if repository.Type == gjson.Null {
		return &Error{Message: "'respository' field not present in payload"}
	}

	cloneUrl := gjson.Get(githubPayload, "repository.clone_url")
	commit := gjson.Get(githubPayload, "after")

	if cloneUrl.Type == gjson.Null || commit.Type == gjson.Null || cloneUrl.String() == "" || commit.String() == "" {
		return &Error{Message: "'repository.clone_url' or after fields not present in payload"}
	}

	log.Infof("Cloning %s@%s", cloneUrl, commit)

	err := gitClone(cloneUrl.String(), commit.String())
	if err != nil {
		return err
	}

	log.Info("Clone succeeded")

	err = runJenkinsFile()
	if err != nil {
		return err
	}

	return nil
}

func gitClone(cloneUrl string, commit string) *Error {
	_ = os.RemoveAll(WORKDIR)

	err := os.Mkdir(WORKDIR, os.ModePerm)
	if err != nil {
		return &Error{Message: fmt.Sprintf("Failed to created dir: %s", WORKDIR)}
	}

	cmd := exec.Command("git", "clone", cloneUrl, WORKDIR)
	cmd.Dir = WORKDIR
	err = cmd.Run()
	if err != nil {
		return &Error{Message: fmt.Sprintf("Failed to clone repository '%s' into %s: %s", cloneUrl, WORKDIR, err.Error())}
	}

	cmd = exec.Command("git", "checkout", commit)
	cmd.Dir = WORKDIR
	err = cmd.Run()
	if err != nil {
		return &Error{Message: fmt.Sprintf("Failed to checkout commit '%s': %s", commit, err.Error())}
	}

	return nil
}

func runJenkinsFile() *Error {
	// TODO: Investigate on how to add custom plugins and check if
	// they have to be extracted into the /tmp/plugins folder

	_ = os.RemoveAll(JENKINSHOMEDIR)
	err := os.Mkdir(JENKINSHOMEDIR, os.ModePerm)
	if err != nil {
		return &Error{Message: fmt.Sprintf("Failed to created dir: %s", JENKINSHOMEDIR)}
	}

	// listFolder(WORKDIR)

	cmd := exec.Command("jenkinsfile-runner",
		"--jenkins-war",
		"/app/jenkins",
		"--plugins",
		"/usr/share/jenkins/ref/plugins",
		"--file",
		fmt.Sprintf("%s/Jenkinsfile", WORKDIR),
		"--runWorkspace",
		TMPDIR,
		"--jenkinsHome",
		JENKINSHOMEDIR,
	)
	cmd.Dir = WORKDIR

	out, err := cmd.CombinedOutput()
	if err != nil {
		return &Error{Message: fmt.Sprintf("Failed to execute command '%s': %s", cmd.String(), string(out))}
	} else {
		log.Infof("%s", string(out))
	}
	return nil
}

func main() {
	lambda.Start(handler)
}

/*
func listFolder(folder string) {
	cmd := exec.Command("ls", "-lah", folder)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("Content Error: %s", err.Error())
	} else {
		log.Infof("Content: %s", string(out))
	}

}
*/
