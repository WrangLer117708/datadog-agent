// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package sender

import (
	"bytes"

	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

type Formatter interface {
	Format(messages []*message.Message) []byte
}

var (
	LineFormatter  Formatter = &lineFormatter{}
	ArrayFormatter Formatter = &arrayFormatter{}
)

type lineFormatter struct{}

func (f *lineFormatter) Format(messages []*message.Message) []byte {
	buffer := bytes.NewBuffer(nil)
	for _, message := range messages {
		buffer.Write(message.Content)
		buffer.WriteByte('\n')
	}
	return buffer.Bytes()
}

type arrayFormatter struct{}

func (f *arrayFormatter) Format(messages []*message.Message) []byte {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte('[')
	for i, message := range messages {
		if i > 0 {
			buffer.WriteByte(',')
		}
		buffer.Write(message.Content)
	}
	buffer.WriteByte(']')
	return buffer.Bytes()
}
