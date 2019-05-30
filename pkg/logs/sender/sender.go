// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/metrics"
)

// Sender sends logs to different destinations.
type Sender interface {
	Start()
	Stop()
}

// sender contains the internal logic to send a payload to multiple destinations,
// it will forever retry for the main destination unless the error is not retryable
// and only try once for additionnal destinations.
type sender struct {
	destinations *client.Destinations
}

// newSender returns a new sender.
func newSender(destinations *client.Destinations) *sender {
	return &sender{
		destinations: destinations,
	}
}

// send sends a payload to multiple destinations,
// returns an error if it failed.
func (s *sender) send(payload []byte) error {
	for {
		err := s.destinations.Main.Send(payload)
		if err != nil {
			metrics.DestinationErrors.Add(1)

			switch err.(type) {
			case *client.RetryableError:
				// could not send the payload because of a client issue,
				// let's retry
				continue
			}

			return err
		}

		// payload sent successfully
		metrics.LogsSent.Add(1)
		break
	}

	for _, destination := range s.destinations.Additionals {
		// send in the background so that the agent does not fall behind
		// because of too many destinations
		destination.SendAsync(payload)
	}

	return nil
}
