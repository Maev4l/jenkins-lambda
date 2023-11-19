package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	log "github.com/sirupsen/logrus"
)

var secret string = os.Getenv("GITHUB_WEBHOOK_SECRET")
var topicArn string = os.Getenv("TOPIC_ARN")

type Response struct {
	Message string `json:"message"`
}

func makeResponse(status int, body interface{}) (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	if body != nil {
		b, _ := json.Marshal(body)
		resp.Body = string(b)
	}

	return &resp, nil
}

func hashPayload(secret string, playloadBody []byte) string {
	hm := hmac.New(sha256.New, []byte(secret))
	hm.Write(playloadBody)
	sum := hm.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func isValidPayload(secret, headerHash string, payload []byte) bool {
	hash := hashPayload(secret, payload)
	return hmac.Equal(
		[]byte(hash),
		[]byte(headerHash),
	)
}

func publishEvent(event string) (*string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := sns.NewFromConfig(cfg)

	input := &sns.PublishInput{
		Message:  &event,
		TopicArn: &topicArn,
	}

	output, err := client.Publish(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	return output.MessageId, nil
}

func handler(event events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	log.Info("Processing event ...")

	var message string
	hash, ok := event.Headers["X-Hub-Signature-256"]
	if !ok {
		message = "Missing signature header"
		log.Error(message)
		return makeResponse(401, Response{Message: message})
	}

	if hash == "" {
		message = "Missing signature"
		log.Error(message)
		return makeResponse(401, Response{Message: message})
	}

	signature_parts := strings.SplitN(hash, "=", 2)
	if len(signature_parts) != 2 {
		log.Errorf("Invalid signature header: '%s' does not contain two parts (hash type and hash)", hash)
		return makeResponse(401, Response{Message: "Invalid signature header"})
	}

	// Ensure secret is a sha256 hash
	signatureType := signature_parts[0]
	signatureHash := signature_parts[1]
	if signatureType != "sha256" {
		log.Errorf("Signature should be a 'sha1' hash not '%s'", signatureType)
		return makeResponse(401, Response{Message: "Invalid signature algorithm"})
	}

	if !isValidPayload(secret, signatureHash, []byte(event.Body)) {
		message = "Invalid Github signature."
		return makeResponse(401, Response{Message: message})
	}

	messageId, err := publishEvent(event.Body)
	if err != nil {
		message = fmt.Sprintf("Failed to publish event: %s", err.Error())
		log.Errorf(message)
		return makeResponse(500, Response{Message: message})
	}

	log.Infof("Event submitted. Message id: %s.", *messageId)
	return makeResponse(200, Response{Message: "Event submitted."})

}

func main() {
	lambda.Start(handler)
}
