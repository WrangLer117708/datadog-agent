// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

type SinglePayload struct {
	singleMessage []*message.Message
	isReady       <-chan time.Time
}

func NewSinglePayload() *SinglePayload {
	return &SinglePayload{
		singleMessage: make([]*message.Message, 1),
		isReady:       make(chan time.Time, 1),
	}
}

func (p *SinglePayload) AddMessage(message *message.Message) bool {
	if p.singleMessage[0] != nil {
		return false
	}
	p.singleMessage[0] = message
	return true
}

func (p *SinglePayload) Clear() {
	p.singleMessage[0] = nil
	select {
	case <-p.isReady:
	default:
	}
}

func (p *SinglePayload) GetContent() []byte {
	message := p.singleMessage[0]
	if message != nil {
		return message.Content
	}
	return nil
}

func (p *SinglePayload) GetMessages() []*message.Message {
	return p.singleMessage
}

func (p *SinglePayload) IsFull() bool {
	return p.singleMessage[0] != nil
}

func (p *SinglePayload) IsEmpty() bool {
	return p.singleMessage[0] == nil
}

func (p *SinglePayload) IsReady() <-chan time.Time {
	return p.isReady
}
