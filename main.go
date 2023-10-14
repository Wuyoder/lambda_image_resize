package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/disintegration/imaging"
)

// GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap -tags lambda.norpc main.go
// zip myFunction.zip bootstrap

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	sess, err := session.NewSession(&aws.Config{
		// Tokyo region is ap-northeast-1
		Region: aws.String("ap-northeast-1")},
	)
	if err != nil {
		// check error, lambda logs also show up in cloudwatch
		fmt.Println("message: error = ", err.Error())
		return err
	}
	svc := s3.New(sess)
	fmt.Println("message: get session")
	// only deal with one file at a time
	object := s3Event.Records[0].S3.Object

	// key is the file name with all prefix
	key := object.Key
	// bucket is the bucket name
	bucket := "intoxicating"
	// check the file key
	fmt.Println("message: key = ", key)
	resp, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Println("message: error = ", err.Error())
		return err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("message: error = ", err.Error())
		return err
	}

	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		fmt.Println("message: error = ", err.Error())
		return err
	}

	resized := imaging.Resize(img, 3000, 3000, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resized, nil)
	if err != nil {
		fmt.Println("message: error = ", err.Error())
		return err
	}
	// get the file name with prefix
	keyPath := strings.Split(key, "/")
	imageName := keyPath[len(keyPath)-1]
	// put the new resized image to the another folder, avoid infinite loop
	newKey := "/podcast/images/" + imageName

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(newKey),
		ContentType: aws.String("image/jpeg"),
		Body:        bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		fmt.Println("message: error = ", err.Error())
		return err
	}

	fmt.Println("message: success")
	return nil
}
