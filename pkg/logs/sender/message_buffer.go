// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

const (
	maxBatchSize   = 20
	maxContentSize = 1000000
)

type MessageBuffer struct {
	messageBuffer []*message.Message
	contentSize   int
}

func NewMessageBuffer() *MessageBuffer {
	return &MessageBuffer{
		messageBuffer: make([]*message.Message, 0, maxBatchSize),
	}
}

func (p *MessageBuffer) AddMessage(message *message.Message) bool {
	contentSize := len(message.Content)
	if len(p.messageBuffer) < cap(p.messageBuffer) && p.contentSize+contentSize <= maxContentSize {
		p.messageBuffer = append(p.messageBuffer, message)
		p.contentSize += contentSize
		return true
	}
	return false
}

func (p *MessageBuffer) Clear() {
	p.messageBuffer = p.messageBuffer[:0]
}

func (p *MessageBuffer) GetMessages() []*message.Message {
	return p.messageBuffer
}

func (p *MessageBuffer) IsFull() bool {
	return len(p.messageBuffer) == cap(p.messageBuffer)
}

func (p *MessageBuffer) IsEmpty() bool {
	return len(p.messageBuffer) == 0
}
