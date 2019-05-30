// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"context"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

const (
	batchTimeout   = 5 * time.Second
	maxBatchSize   = 20
	maxContentSize = 1000000
)

// BatchSender is responsible for sending a batch of logs to different destinations.
type BatchSender struct {
	inputChan     chan *message.Message
	outputChan    chan *message.Message
	sender        *sender
	done          chan struct{}
	batchTimeout  time.Duration
	messageBuffer *MessageBuffer
}

// NewBatchSender returns an new BatchSender.
func NewBatchSender(inputChan, outputChan chan *message.Message, destinations *client.Destinations) *BatchSender {
	return &BatchSender{
		inputChan:     inputChan,
		outputChan:    outputChan,
		sender:        newSender(destinations),
		done:          make(chan struct{}),
		batchTimeout:  batchTimeout,
		messageBuffer: NewMessageBuffer(maxBatchSize, maxContentSize),
	}
}

// Start starts the BatchSender
func (s *BatchSender) Start() {
	go s.run()
}

// Stop stops the BatchSender,
// this call blocks until inputChan is flushed
func (s *BatchSender) Stop() {
	close(s.inputChan)
	<-s.done
}

// run lets the BatchSender send messages.
func (s *BatchSender) run() {
	flushTimer := time.NewTimer(s.batchTimeout)
	defer func() {
		flushTimer.Stop()
		s.done <- struct{}{}
	}()

	for {
		select {
		case payload, isOpen := <-s.inputChan:
			if !isOpen {
				// inputChan has been closed, no more payload are expected,
				// flush the remaining messages.
				s.sendBuffer()
				return
			}
			ok := s.messageBuffer.TryAddMessage(payload)
			if !ok || s.messageBuffer.IsFull() {
				// message buffer is full, either reaching maxBatchCount of maxRequestSize
				// send request now.
				if !flushTimer.Stop() {
					select {
					case <-flushTimer.C:
					default:
					}
				}
				s.sendBuffer()
				flushTimer.Reset(s.batchTimeout)
			}
			if !ok {
				// it's possible we didn't append last try because maxRequestSize is reached
				// append it again after the sendbuffer is flushed
				s.messageBuffer.TryAddMessage(payload)
			}
		case <-flushTimer.C:
			// the timout expired, the content is ready to be sent
			s.sendBuffer()
			flushTimer.Reset(s.batchTimeout)
		}
	}
}

// sendBuffer keeps trying to send the message to the main destination until it succeeds
// and try to send the message to the additional destinations only once.
func (s *BatchSender) sendBuffer() {
	if s.messageBuffer.IsEmpty() {
		return
	}

	batchedContent := s.messageBuffer.GetPayload()
	defer s.messageBuffer.Clear()

	// this call is blocking until the payload gets sent
	// or the connection context got cancelled
	err := s.sender.send(batchedContent)
	switch err {
	case context.Canceled:
		// the context was cancelled, the agent is stopping non-gracefully,
		// drop the message
		return
	default:
		// the sender could not sent the payload,
		// this can happen when the payload can not be frammed properly,
		// or the client is configured with a wrong url or a wrong API key.
		break
	}

	for _, message := range s.messageBuffer.GetMessages() {
		s.outputChan <- message
	}
}
