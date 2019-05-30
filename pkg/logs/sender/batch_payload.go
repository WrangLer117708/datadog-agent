// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

const (
	batchTimeout   = 5 * time.Second
	maxBatchSize   = 20
	maxContentSize = 1000000
)

type BatchPayload struct {
	formatter     Formatter
	messageBuffer []*message.Message
	contentSize   int
	flushTimer    *time.Timer
	isReady       <-chan time.Time
}

func NewBatchPayload(formatter Formatter) *BatchPayload {
	flushTimer := time.NewTimer(batchTimeout)
	return &BatchPayload{
		formatter:     formatter,
		messageBuffer: make([]*message.Message, 0, maxBatchSize),
		flushTimer:    flushTimer,
		isReady:       flushTimer.C,
	}
}

func (p *BatchPayload) AddMessage(message *message.Message) bool {
	contentSize := len(message.Content)
	if len(p.messageBuffer) < cap(p.messageBuffer) && p.contentSize+contentSize <= maxContentSize {
		p.messageBuffer = append(p.messageBuffer, message)
		p.contentSize += contentSize
		return true
	}
	return false
}

func (p *BatchPayload) Clear() {
	p.messageBuffer = p.messageBuffer[:0]
	if !p.flushTimer.Stop() {
		<-p.flushTimer.C
	}
	p.flushTimer.Reset(batchTimeout)
}

func (p *BatchPayload) GetContent() []byte {
	return p.formatter.Format(p.messageBuffer)
}

func (p *BatchPayload) GetMessages() []*message.Message {
	return p.messageBuffer
}

func (p *BatchPayload) IsFull() bool {
	return len(p.messageBuffer) == cap(p.messageBuffer)
}

func (p *BatchPayload) IsEmpty() bool {
	return len(p.messageBuffer) == 0
}

func (p *BatchPayload) IsReady() <-chan time.Time {
	return p.isReady
}
