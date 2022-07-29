package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"

	factc "github.com/faas-facts/fact-go-client"

	"tester/bencher"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

var client *factc.FactClient

func init() {
	logrus.SetLevel(logrus.DebugLevel)

	client = &factc.FactClient{}
	// platfrom := "AWS"
	client.Boot(factc.FactClientConfig{
		// Platform:           &platfrom,
		SendOnUpdate:       false,
		IncludeEnvironment: false,
		IOArgs:             map[string]string{},
	})
}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, job bencher.Job) (Response, error) {

	trace := bencher.Handle(client, job, ctx)

	body, err := json.Marshal(trace)
	if err != nil {
		return Response{StatusCode: 404}, err
	}
	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
