package opendevopslambda

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"testing"
)

type mockedPutOjbect struct {
	s3iface.S3API
	Response s3.PutObjectOutput
}

type mockedPutItem struct {
	dynamodbiface.DynamoDBAPI
	Response dynamodb.PutItemOutput
}

func (d mockedPutOjbect) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return &d.Response, nil
}

func (d mockedPutItem) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return &d.Response, nil
}

func TestHandler(t *testing.T) {
	t.Run("Successful Request", func(t *testing.T) {
		mpo := mockedPutOjbect {
			Response: s3.PutObjectOutput{},
		}

		mpi := mockedPutItem {
			Response: dynamodb.PutItemOutput{},
		}

		d := Dependency{
			DepS3: mpo,
			DepDynamoDB: mpi,
		}

		ctx := context.Background()
		lc := new(lambdacontext.LambdaContext)
		lc.InvokedFunctionArn = "arn:aws:lambda:region:123456789000:function:functionName"
		ctx = lambdacontext.NewContext(ctx, lc)

		qsp := map[string]string{}
		qsp["url"] = "https://i.ytimg.com/vi/iVZYAhzxG4Y/maxresdefault.jpg"

		request := events.APIGatewayProxyRequest{
			QueryStringParameters: qsp,
		}

		_, err := d.Handler(ctx, request)
		if err != nil {
			t.Fatal(fmt.Sprintf("TestHandler failed with %s", err.Error()))
		}
	})
}
