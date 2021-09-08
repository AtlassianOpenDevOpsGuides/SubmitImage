package main

import (
	"log"
	"os"
	"submit-image/opendevopslambda"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	sess := session.Must(session.NewSession())

	d := opendevopslambda.Dependency{
		DepS3: s3.New(sess),
		DepDynamoDB: dynamodb.New(sess),
	}

	lambda.Start(d.Handler)
}
