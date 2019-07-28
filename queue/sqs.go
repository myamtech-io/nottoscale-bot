package queue

import (
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bwmarrin/discordgo"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	protos "github.com/webmakersteve/myamtech-bot/proto"
	"sync"
	"time"
)

var (
	queue      string
	regionName string
	timeout    int64
)

func init() {
	regionName = "us-west-2"
	queue = "https://sqs.us-west-2.amazonaws.com/575393002463/homenot_to_scale_20190721061215856500000001"
	timeout = 1000
}

type channelMessage struct {
	Payload *protos.DiscordMessage
}

type Queue struct {
	channel        chan channelMessage
	waitGroup      *sync.WaitGroup
	sqsClient      *sqs.SQS
	isShuttingDown bool
}

func ListenForMessages(s *discordgo.Session) (*Queue, error) {
	// sc := make(chan os.Signal, 1)
	protoChannel := make(chan channelMessage)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(regionName)},
	)

	if err != nil {
		log.Error(err)
		return nil, err
	}

	var wg sync.WaitGroup

	queue := &Queue{
		channel:        protoChannel,
		waitGroup:      &wg,
		sqsClient:      sqs.New(sess, aws.NewConfig().WithLogLevel(aws.LogOff)),
		isShuttingDown: false,
	}

	wg.Add(1)
	go queue.loopMessages()
	wg.Add(1)
	go queue.loopSendMessages(s)

	return queue, err
}

func (q *Queue) getNextMessages() ([]*protos.DiscordMessage, error) {
	var svc = q.sqsClient
	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: &queue,
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MaxNumberOfMessages: aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: aws.Int64(10),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"queue": queue,
		}).Error(err, "Unable to receive message from queue.")
		return nil, err
	}

	log.WithFields(log.Fields{
		"count": len(result.Messages),
		"queue": queue,
	}).Info("Received messages from SQS queue")

	var deserializedProtos = make([]*protos.DiscordMessage, len(result.Messages))
	if len(result.Messages) > 0 {
		for i, message := range result.Messages {
			// Delete the message before we pass it off. We just try our best here and want to be an at
			// most once system
			svc.DeleteMessage(&sqs.DeleteMessageInput{
				ReceiptHandle: message.ReceiptHandle,
				QueueUrl:      &queue,
			})
			if message.Body != nil {
				deserializedMessage := &protos.DiscordMessage{}
				decoded, err := base64.StdEncoding.DecodeString(*message.Body)
				if err != nil {
					continue
				}

				err = proto.Unmarshal(decoded, deserializedMessage)
				if err == nil {
					deserializedProtos[i] = deserializedMessage
				} else {
					deserializedProtos[i] = nil
				}
			}
		}
	}
	return deserializedProtos, nil
}

func (q *Queue) loopMessages() {
	defer q.waitGroup.Done()

	for {
		if q.isShuttingDown {
			break
		}
		log.Info("Getting next messages")
		messages, err := q.getNextMessages()
		if err != nil {
			log.Warn("Delaying next message fetch since we errored")
			time.Sleep(10 * time.Second)
			continue
		}

		if len(messages) == 0 {
			log.Info("Delaying next message fetch since we got nothing back")
			time.Sleep(5 * time.Second)
		}

		for _, message := range messages {
			// Message can be nil if it could not be deserialized
			if message != nil {
				q.channel <- channelMessage{
					Payload: message,
				}
			}
		}
	}

	log.Debug("Read routine exiting")
}

func (q *Queue) loopSendMessages(s *discordgo.Session) {
	defer q.waitGroup.Done()

	for {
		if q.isShuttingDown {
			break
		}

		msg := <-q.channel

		// Nil messages can be used to poke the goroutine
		if msg.Payload == nil {
			continue
		}

		if msg.Payload.Payload == nil || msg.Payload.Channel == nil {
			continue
		}

		_, err := s.ChannelMessageSend(*msg.Payload.Channel, *msg.Payload.Payload)
		if err != nil {
			log.Error(err)
		}
	}

	log.Debug("Send routine exiting")
}

func (q *Queue) Close() {
	q.isShuttingDown = true
	log.Info("Waiting for queue routine to finish...")
	q.channel <- channelMessage{
		Payload: nil,
	}
	q.waitGroup.Wait()
}
