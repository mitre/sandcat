package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/mitre/gocat/execute/native/util"
)

type openFileWrapper func(string) (*os.File, error)
type uploadDataWrapper func(context.Context, string, string, io.ReadSeeker) error

type funcWrapperStruct struct {
	openFileFn openFileWrapper
	uploadDataFn uploadDataWrapper
}

const argErrMsg = "Expected format: [file to upload] [bucket name] [object key] [timeout]"

var funcWrappers *funcWrapperStruct

func init() {
	util.NativeMethods["s3upload"] = UploadToS3Bucket
	funcWrappers = &funcWrapperStruct{
		openFileFn: openFile,
		uploadDataFn: uploadDataToBucket,
	}
}

// Wrapper for opening file
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// Wrapper for uploading data to S3 bucket
func uploadDataToBucket(ctx context.Context, bucket string, key string, fileReadSeeker io.ReadSeeker) error {
	svc := s3.New(session.Must(session.NewSession()))
  	_, err := svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
 		Bucket: aws.String(bucket),
 		Key: aws.String(key),
 		Body: fileReadSeeker,
 	})
 	return err
}

// Uploads specified file to s3 bucket.
// Expects args to be of the format: [file to upload] [bucket name] [object key] [timeout]
// Reference: https://pkg.go.dev/github.com/aws/aws-sdk-go#hdr-Complete_SDK_Example
func UploadToS3Bucket(uploadArgs []string) util.NativeCmdResult {
	var errMsg string

	// Process args
	if len(uploadArgs) != 4 {
		return util.GenerateErrorResultFromString(argErrMsg)
	}
	fileToUpload := uploadArgs[0]
	bucket := uploadArgs[1]
	key := uploadArgs[2]
  	timeout, err := time.ParseDuration(uploadArgs[3])
  	if err != nil {
  		return util.GenerateErrorResult(err)
  	}

	// Read in file data to upload
  	fileReadSeeker, err := funcWrappers.openFileFn(fileToUpload)
  	if err != nil {
  		return util.GenerateErrorResult(err)
  	}

	// Set up context
  	ctx := context.Background()
  	var cancelFn func()
  	if timeout > 0 {
  		ctx, cancelFn = context.WithTimeout(ctx, timeout)
  	}
  	defer cancelFn()

	// Upload to S3
	err = funcWrappers.uploadDataFn(ctx, bucket, key, fileReadSeeker)
	if err != nil {
 		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
 			errMsg = fmt.Sprintf("Upload canceled due to timeout: %v\n", err)
 		} else {
 			errMsg = fmt.Sprintf("Failed to upload object: %v\n", err)
 		}
 		return util.GenerateErrorResultFromString(errMsg)
 	}
 	return util.NativeCmdResult{
		Stdout: []byte(fmt.Sprintf("Successfully uploaded file %s to %s/%s\n", fileToUpload, bucket, key)),
		Stderr: nil,
		Err: nil,
	}
}
