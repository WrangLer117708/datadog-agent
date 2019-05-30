// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"context"

	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/DataDog/datadog-agent/pkg/logs/metrics"
)

// Sender contains the internal logic to send a payload to multiple destinations,
// it will forever retry for the main destination unless the error is not retryable
// and only try once for additionnal destinations.
type Sender struct {
	inputChan    chan *message.Message
	outputChan   chan *message.Message
	destinations *client.Destinations
	payload      Payload
	done         chan struct{}
}

// NewSender returns a new Sender.
func NewSender(inputChan, outputChan chan *message.Message, destinations *client.Destinations, payload Payload) *Sender {
	return &Sender{
		inputChan:    inputChan,
		outputChan:   outputChan,
		destinations: destinations,
		payload:      payload,
		done:         make(chan struct{}),
	}
}

// Start starts the Sender
func (s *Sender) Start() {
	go s.run()
}

// Stop stops the Sender,
// this call blocks until inputChan is flushed
func (s *Sender) Stop() {
	close(s.inputChan)
	<-s.done
}

// run lets the Sender send messages.
func (s *Sender) run() {
	defer func() {
		s.done <- struct{}{}
	}()

	for {
		select {
		case message, isOpen := <-s.inputChan:
			if !isOpen {
				s.sendPayload()
				return
			} else {
				ok := s.payload.AddMessage(message)
				if !ok || s.payload.IsFull() {
					s.sendPayload()
				}
				if !ok {
					s.payload.AddMessage(message)
				}
			}
		case <-s.payload.IsReady():
			s.sendPayload()
		}
	}
}

// send sends the current payload to the destinations
// and forward its messages to the next stage.
func (s *Sender) sendPayload() {
	if s.payload.IsEmpty() {
		return
	}

	content := s.payload.GetContent()
	defer s.payload.Clear()

	// this call is blocking until the payload gets sent
	// or the client context got cancelled
	err := s.sendToDestinations(content)
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

	for _, message := range s.payload.GetMessages() {
		s.outputChan <- message
	}
}

// sendToDestinations sends a content to the different destinations,
// returns an error if it failed.
func (s *Sender) sendToDestinations(content []byte) error {
	for {
		err := s.destinations.Main.Send(content)
		if err != nil {
			metrics.DestinationErrors.Add(1)

			switch err.(type) {
			case *client.RetryableError:
				// could not send the content because of a client issue,
				// let's retry
				continue
			}

			return err
		}

		// content sent successfully
		metrics.LogsSent.Add(1)
		break
	}

	for _, destination := range s.destinations.Additionals {
		// send in the background so that the agent does not fall behind
		// because of too many destinations
		destination.SendAsync(content)
	}

	return nil
}
