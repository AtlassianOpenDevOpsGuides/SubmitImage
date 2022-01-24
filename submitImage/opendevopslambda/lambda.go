package opendevopslambda

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
  "os"
	"strings"
  "time"
  "gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
  ld "gopkg.in/launchdarkly/go-server-sdk.v5"
)

type Dependency struct {
	DepS3 s3iface.S3API
	DepDynamoDB dynamodbiface.DynamoDBAPI
}

var bucketRootName = "open-devops-images"

func getLaunchDarklyFlags(username string) (bool, error) {
  client, _ := ld.MakeClient(os.Getenv("LaunchDarklySDKKey"), 5 * time.Second)
  flagKey := "SubmitImageDemoFeature"

  userUuid, uuidErr := uuid.NewRandom()
  if uuidErr != nil {
		return false, uuidErr
	}

  var user lduser.User
  if(username == "") {
    user = lduser.NewAnonymousUser(userUuid.String())
  } else {
    user = lduser.NewUser(username)
  }

  showFeature, _ := client.BoolVariation(flagKey, user, false)

  if showFeature {
    return true, nil
  } else {
    return false, nil
  }
}

func (d *Dependency) processRequest(imageUrl string, region string, aws_account_id string) (string, error) {
	response, err := http.Get(imageUrl)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("response.StatusCode %d != 200\n", response.StatusCode))
	}

  flagVal, flagErr  := getLaunchDarklyFlags("")
  if flagErr != nil {
    return "", flagErr
  }
  fmt.Println("DEMO flagVal for anonymous user: ", flagVal)

  flagVal, flagErr  = getLaunchDarklyFlags("AtlassianTestUser@atlassian.com")
  if flagErr != nil {
    return "", flagErr
  }
  fmt.Println("DEMO flagVal for AtlassianTestUser@atlassian.com: ", flagVal)

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	bucketName := fmt.Sprintf("%s-%s-%s", bucketRootName, region, aws_account_id)

	imageUuid, uuidErr := uuid.NewRandom()
	if uuidErr != nil {
		return "", uuidErr
	}

	s3Input := &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(bucketName),
		Key:    aws.String(imageUuid.String()),
	}

	_, s3err := d.DepS3.PutObject(s3Input)
	if s3err != nil {
		return "", s3err
	}

	dynamoInput := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"Id": {
				S: aws.String(imageUuid.String()),
			},
			"Label": {
				S: aws.String("NOT_CLASSIFIED"),
			},
		},
		TableName: aws.String("ImageLabels"),
	}

	_, dynamoErr := d.DepDynamoDB.PutItem(dynamoInput)
	if dynamoErr != nil {
		return "", dynamoErr
	}

	return imageUuid.String(), nil
}

func isValidExtension(urlVal string) bool {
	validExtensions := []string{"jpeg", "jpg", "bmp", "png", "tiff", "gif", "tif"}

	urlSlice := strings.Split(urlVal, "/")
	fileName := urlSlice[len(urlSlice)-1]
	fileNameSlice := strings.Split(fileName, ".")
	fileExtension := fileNameSlice[len(fileNameSlice)-1]

	for _, ext := range validExtensions {
		if fileExtension == ext {
			return true
		}
	}
	return false
}

func (d *Dependency) Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	lc, _ := lambdacontext.FromContext(ctx)
	region := strings.Split(lc.InvokedFunctionArn, ":")[3]
  aws_account_id := strings.Split(lc.InvokedFunctionArn, ":")[4]

	urlParam, found := request.QueryStringParameters["url"]
	if found {
		urlVal, err := url.QueryUnescape(urlParam)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500,
				Body: `{"ImageId":"error"}`,
				IsBase64Encoded: false,
			}, err
		}

		if !isValidExtension(urlVal) {
			return events.APIGatewayProxyResponse{StatusCode: 500,
				Body: `{"ImageId":"error"}`,
				IsBase64Encoded: false,
			}, errors.New("file extension %s is not valid")
		}

		processString, processErr := d.processRequest(urlVal, region, aws_account_id)
		return events.APIGatewayProxyResponse{StatusCode: 200,
			Body: fmt.Sprintf(`"ImageId":"%s"`, processString),
			IsBase64Encoded: false,
		}, processErr
	}

	return events.APIGatewayProxyResponse{StatusCode: 500,
		Body: `{"ImageId":"error"}`,
		IsBase64Encoded: false,
	}, errors.New("url parameter not found")
}
