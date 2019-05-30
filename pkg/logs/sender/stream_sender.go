// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"context"

	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// StreamSender sends one log at a time to different destinations.
type StreamSender struct {
	inputChan    chan *message.Message
	outputChan   chan *message.Message
	sender       *sender
	destinations *client.Destinations
	done         chan struct{}
}

// NewStreamSender returns an new StreamSender.
func NewStreamSender(inputChan, outputChan chan *message.Message, destinations *client.Destinations) *StreamSender {
	return &StreamSender{
		inputChan:    inputChan,
		outputChan:   outputChan,
		sender:       newSender(destinations),
		destinations: destinations,
		done:         make(chan struct{}),
	}
}

// Start starts the StreamSender
func (s *StreamSender) Start() {
	go s.run()
}

// Stop stops the StreamSender,
// this call blocks until inputChan is flushed
func (s *StreamSender) Stop() {
	close(s.inputChan)
	<-s.done
}

// run lets the StreamSender send messages.
func (s *StreamSender) run() {
	defer func() {
		s.done <- struct{}{}
	}()

	for payload := range s.inputChan {
		s.send(payload)
	}
}

// send keeps trying to send the message to the main destination until it succeeds
// and try to send the message to the additional destinations only once.
func (s *StreamSender) send(message *message.Message) {
	// this call is blocking until the payload gets sent
	// or the connection context got cancelled
	err := s.sender.send(message.Content)
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
	s.outputChan <- message
}
