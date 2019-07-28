package simulationcraft

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

var (
	s3store    *s3.S3
	bucketName string
	regionName string
)

func init() {
	regionName = "us-west-2"
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(regionName)},
	)

	if err != nil {
		log.Fatal()
	}

	s3store = s3.New(sess, aws.NewConfig().WithLogLevel(aws.LogOff))
	bucketName = "myamtech-simulation-sims"
}

// PutFile does things
func PutFile(filePath string) (string, error) {
	key := path.Join("simulations", uuid.New().String()+".html")
	fullPath := "https://" + bucketName + ".s3." + regionName + ".amazonaws.com/" + key

	log.Info("Putting s3 object up at path: " + fullPath)

	// Open the file for use
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	putInput := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buffer),
		ContentType:   aws.String("text/html"),
		ContentLength: aws.Int64(size),
	}

	_, err = s3store.PutObject(putInput)

	if err != nil {
		log.Error("Error while trying to put object to s3: ", err)
		return "", err
	}

	return fullPath, nil
}
