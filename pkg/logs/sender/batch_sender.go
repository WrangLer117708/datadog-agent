// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"context"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

const (
	batchTimeout = 5 * time.Second
)

// BatchSender is responsible for sending a batch of logs to different destinations.
type BatchSender struct {
	inputChan    chan *message.Message
	outputChan   chan *message.Message
	destinations *client.Destinations
	formatter    Formatter
	buffer       *MessageBuffer
	batchTimeout time.Duration
	done         chan struct{}
}

// NewBatchSender returns an new BatchSender.
func NewBatchSender(inputChan, outputChan chan *message.Message, destinations *client.Destinations, formatter Formatter) *BatchSender {
	return &BatchSender{
		inputChan:    inputChan,
		outputChan:   outputChan,
		destinations: destinations,
		formatter:    formatter,
		buffer:       NewMessageBuffer(),
		batchTimeout: batchTimeout,
		done:         make(chan struct{}),
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
		case message, isOpen := <-s.inputChan:
			if !isOpen {
				s.sendBuffer()
				return
			}
			added := s.buffer.AddMessage(message)
			if !added || s.buffer.IsFull() {
				if !flushTimer.Stop() {
					select {
					case <-flushTimer.C:
					default:
					}
				}
				s.sendBuffer()
				flushTimer.Reset(s.batchTimeout)
			}
			if !added {
				s.buffer.AddMessage(message)
			}
		case <-flushTimer.C:
			s.sendBuffer()
			flushTimer.Reset(s.batchTimeout)
		}
	}
}

// sendBuffer keeps trying to send the message to the main destination until it succeeds
// and try to send the message to the additional destinations only once.
func (s *BatchSender) sendBuffer() {
	if s.buffer.IsEmpty() {
		return
	}

	messages := s.buffer.GetMessages()
	defer s.buffer.Clear()

	// this call is blocking until the payload gets sent
	// or the connection context got cancelled
	err := send(s.formatter.Format(messages), s.destinations)
	if err != nil {
		if err == context.Canceled {
			// the context was cancelled, the agent is stopping non-gracefully,
			// drop the message
			return
		}
		log.Warn("Could not send payload: %v", err)
	}

	for _, message := range messages {
		s.outputChan <- message
	}
}
