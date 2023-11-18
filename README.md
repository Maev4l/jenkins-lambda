# Jenkins Lambda

An AWS Lambda based project to run Jenkins pipelines triggered bu Github webhooks.
This project relies on the serverless framework (https://www.serverless.com/framework/docs) for AWS Lambda functions deployment and CloudFormation for AWS assets provisioning.

**Warning** This project cannot be deployed and executed as-is. You have to adapt it according to your environment and your needs.

## Architecture

TODO

This project consists mainly of 2 lambda functions:

- authorizer: Validates the incoming request from GitHub, based on the webhook secret
- handler: The main processor. It runs the Jenkinsfile included in Github repository

The Github webhook endpoint is exposed by an API Gateway endpoint, and thanks to API Gateway integration feature, the event is pushed into an SNS topic, and consumed by the handler Lambda function. In addition, a Route 53 record points to API Gateway endpoint, hence providing a user-friendly URL.

It is an asynchroneous processing to workaround the API Gateway 30s timeout.

## Prerequisites

### 1. Docker image registry

Create the ECR repository to host the handler Docker images.

```shell
aws ecr create-repository --repository-name jenkins-lambda/handler --tags Key=application,Value=jenkins-lambda
aws ecr put-lifecycle-policy --repository-name jenkins-lambda/handler --lifecycle-policy-text "file://functions/handler/ecr-lifecycle-policy.json"
```

### 2. AWS IAM Role for AWS API Gateway access logging

Create an IAM role so AWS API Gateway can log into Cloudwatch.

```yaml
---
AWSTemplateFormatVersion: "2010-09-09"
Resources:
  ApiGatewayLoggingRole:
    Type: "AWS::IAM::Role"
    Properties:
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal:
              Service:
                - "apigateway.amazonaws.com"
            Action: "sts:AssumeRole"
      Path: "/"
      ManagedPolicyArns:
        - !Sub "arn:${AWS::Partition}:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
```

Then set the above role ARN in API Gateway console: API Gateway > Settings > Logging

## Resources

- https://programmerblock.com/a-step-by-step-guide-to-connect-sns-with-api-gateway/#How_to_connect_API_Gateway_with_SNS_%E2%80%93_Step_by_Step_Process
- https://github.com/carlossg/jenkinsfile-runner-lambda
- https://github.com/jenkinsci/jenkinsfile-runner
- https://www.alexdebrie.com/posts/api-gateway-access-logs/
- https://gist.github.com/villasv/4f5b62a772abe2c06525356f80299048
- https://github.com/awslabs/aws-apigateway-lambda-authorizer-blueprints
- https://gist.github.com/carceneaux/7a5ef7439a7dc514b8da61fe929df5ca (API Gateway custom authorizer)
