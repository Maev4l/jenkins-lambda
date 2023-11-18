package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
)

func handler(request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	log.Info("so far so good")

	res := events.APIGatewayCustomAuthorizerResponse{}
	return res, nil
}

func main() {
	lambda.Start(handler)
}
