package achievements

import (
	"context"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/log"
	"github.com/pkg/errors"
)

func newProxyProcessor(processors ...messagebroker.Processor) messagebroker.Processor {
	return &proxyProcessor{internalProcessors: processors}
}

func (proxy *proxyProcessor) Process(ctx context.Context, message *messagebroker.Message) error {
	// May be to add async processing in the futrure.
	for _, processor := range proxy.internalProcessors {
		if err := processor.Process(ctx, message); err != nil {
			log.Error(errors.Wrapf(err, "proxyProcessor: failed to process %v message on %T", string(message.Value), processor))
		}
	}

	return nil
}
