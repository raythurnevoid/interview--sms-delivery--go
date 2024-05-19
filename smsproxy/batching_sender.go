package smsproxy

import (
	"sync"

	"gitlab.com/devskiller-tasks/messaging-app-golang/fastsmsing"
)

type batchingClient interface {
	send(message SendMessage, ID MessageID) error
}

func newBatchingClient(
	repository repository,
	client fastsmsing.FastSmsingClient,
	config smsProxyConfig,
	statistics ClientStatistics,
) batchingClient {
	return &simpleBatchingClient{
		repository:     repository,
		client:         client,
		messagesToSend: make([]fastsmsing.Message, 0),
		config:         config,
		statistics:     statistics,
		lock:           sync.RWMutex{},
	}
}

type simpleBatchingClient struct {
	config         smsProxyConfig
	repository     repository
	client         fastsmsing.FastSmsingClient
	statistics     ClientStatistics
	messagesToSend []fastsmsing.Message
	lock           sync.RWMutex
}

func (b *simpleBatchingClient) send(message SendMessage, ID MessageID) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if err := b.repository.save(ID); err != nil {
		return err
	}

	b.messagesToSend = append(b.messagesToSend, fastsmsing.Message{
		PhoneNumber: message.PhoneNumber,
		Message:     message.Message,
		MessageID:   ID,
	})

	if len(b.messagesToSend) >= b.config.minimumInBatch {
		// go func() {
		// b.lock.Lock()
		// defer b.lock.Unlock()

		var err error
		for i := 1; i <= calculateMaxAttempts(b.config.maxAttempts); i++ {
			err = b.client.Send(b.messagesToSend)

			if lastAttemptFailed(i, b.config.maxAttempts, err) {
				sendStatistics(b.messagesToSend, err, i, b.config.maxAttempts, b.statistics)
				break
			} else if err == nil {
				b.messagesToSend = make([]fastsmsing.Message, 0)
				sendStatistics(b.messagesToSend, err, i, b.config.maxAttempts, b.statistics)
				break
			}

		}

		if err == nil {
			for _, message := range b.messagesToSend {
				b.repository.update(message.MessageID, Delivered)
			}
		} else {
			for _, message := range b.messagesToSend {
				b.repository.update(message.MessageID, Failed)
			}
		}

		// }()
	}

	return nil
}

func calculateMaxAttempts(configMaxAttempts int) int {
	if configMaxAttempts < 1 {
		return 1
	}
	return configMaxAttempts
}

func lastAttemptFailed(currentAttempt int, maxAttempts int, currentAttemptError error) bool {
	return currentAttempt == maxAttempts && currentAttemptError != nil
}

func sendStatistics(messages []fastsmsing.Message, lastErr error, currentAttempt int, maxAttempts int, statistics ClientStatistics) {
	statistics.Send(clientResult{
		messagesBatch:  messages,
		err:            lastErr,
		currentAttempt: currentAttempt,
		maxAttempts:    maxAttempts,
	})
}
