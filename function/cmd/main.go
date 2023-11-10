package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
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

	var message string
	valid := gjson.Valid(event.Body)
	if !valid {
		message = "Invalid Github payload"
		log.Error(message)
		return *apiResponse(500, Error{Message: message}), nil
	}

	repository := gjson.Get(event.Body, "repository")
	if repository.Type == gjson.Null {
		message = "'respository' field not present in payload"
		return *apiResponse(500, Error{Message: message}), nil
	}

	cloneUrl := gjson.Get(event.Body, "repository.clone_url")
	commit := gjson.Get(event.Body, "after")

	if cloneUrl.Type == gjson.Null || commit.Type == gjson.Null || cloneUrl.String() == "" || commit.String() == "" {
		message = "'repository.clone_url' or after fields not present in payload"
		return *apiResponse(500, Error{Message: message}), nil
	}

	log.Infof("Cloning %s@%s", cloneUrl, commit)

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "\"Hello from Lambda!\"",
	}
	return response, nil
}

func main() {
	lambda.Start(handler)
}
