# SMS Proxy Golang

# Background
Your company, the best online shop in your country, wants to implement a new feature to its platform. 
Your task is to write a small service to deliver messages to the company's clients.

After doing an honest research about your options, you have decided to use an external SMS Provider service called FastSMSing.
Unfortunately, FastSMSing requires payment for each SMS sent :(... but there is a way to save some money using their batch API, as the cost for using a batch request is the same as for using a single-message function.
The company decided that they can live with a short delay of message delivery as the costs savings will be huge!
Your task is to write a cost-efficient, stable and reliable integration with FastSMSing service.

# Introduction

Your task is to provide a REST service which allows to:

* `POST` a new SMS message, which queues its sending.
* `GET` a current Status of a given message by its `MessageID`.

Additional reason for writing this proxy SMS service is to provide a larger range of SMS statuses.

FastSMSing only operates using three statuses:

* `CONFIRMED` - used when FastSMSing receives a given request and schedules the SMS delivery.
* `FAILED` - used when FastSMSing fails to deliver a message after it has already been `Confirmed`.
* `DELIVERED` - used when the delivery is successful.

Your service provides five different statuses that might be returned via REST API:

* `ACCEPTED` - used when your service accepts a message to be sent in the nearest batch of messages.
* `NOT_FOUND` - used for messages that have not been planned to be sent.
* `CONFIRMED` - used for messages with the `CONFIRMED` status, received via FastSMSing updates mechanism.
* `FAILED` - used for messages with the `FAILED` status, received via FastSMSing updates mechanism.
* `DELIVERED` - used for messages with the `DELIVERED` status, received via FastSMSing updates mechanism.

# Problem Statement
  
This is going to be easy! Each function you have to implement contains a few additional tips you should follow.

Follow this order of implementation to make all tests green!

### 1. Saving statuses of all messages

* Implement `inMemoryRepository` in `smsproxy/repository.go`. Make all operations resistant to race conditions when using goroutines.
    * `save(...)` - save a given MessageID with the `ACCEPTED` status. If the given MessageID already exists, return an error.
    * `get(...)` - return the status of a given message, by its MessageID. If not found, return the `NotFound` status.
    * `update()` - set a new status for a given message. If the message is not in the `ACCEPTED` state, return an error. If the current status is `FAILED` or `DELIVERED`, do not update it and return an error. Those are the final statuses and they cannot be overwritten.
    
### 2. Sending messages in batches

* Complete the implementation of `simpleBatchingClient` in `smsproxy/batching_sender.go`
    * After receiving a new message, save it in the `repository` with the `ACCEPTED` status. If saving fails, return an error.

    * If the number of messages in the current batch is `>=` than the batch size specified in the `simpleBatchingClient.config`, send a batch of messages via `simpleBatchingClient.client.Send()`.
    
  
    * Sometimes FastSMSing API is not stable. A good-enough solution for now is to use retries. Try sending each batch of messages at the `maxAttempts` number of times specified by the `simpleBatchingClient.config.maxAttempts`.

* You have to gather success/failure statistics of FastSMSing Client for monitoring purposes (how stable it is, etc.). You can use `simpleBatchingClient.sendStatistics(...)` and use all specified information to gather all necessary information. 
    * Whenever a batch is sent successfully, send statistics with `nil` error.
    * Whenever a batch sending failed after using all available retries, send statistics with the last error received from FastSMSing Client.
    
* FastSMSing Client can take a long time to complete the given request. As we do not want users of your `Proxy SMS Service` to wait for a long time (FastSMSing Client finishes processing the request), all attempts to use FastSMSing Client should be asynchronous. Waiting for statistics to be sent should not block the `Proxy SMS Service` client as well.
  
  
  
### 3. Receiving message status updates

* Your service knows about FastSMSing messages statuses thanks to the `FastSmsingClient.Subscribe(...)` method. Those updates come in batches as well.
* Implement the `Start()` method in `smsproxy/status_updater.go`:
    * When started, statusUpdater should continue reading from the statusUpdater.C channel, where the updates will be delivered, and save them using the `repository.update(...)`.
    * `fastssmsing.MessageStatus` should be mapped to the `smsproxy.MessageStatus` using the `mapToInternalStatus` method before updating state using the `repository.update(...)`.
    * When mapping to internal status failed or updating the status using the `repository.update(...)` failed, you should asynchronously send the `statusUpdateError` into the `statusUpdater.Errors` channel.

### 4. Allow HTTP access to your service

Implement the `sendSmsHandler` in the `restapi/handlers.go` to enable sending messages via HTTP POST method:
* HINT: you can take a look at the `getSmsStatusHandler` method for some inspiration.
* HINT: you can use the `handleError()` function when handling any error.
    
Requirements: 
1. Read the SendSmsRequest from the request. When an error occurred, return the HTTP Status 400.
2. Try sending an SMS using the `smsProxy.Send(...)`.
    * If the `smsProxy.Send(...)` returns an error of the following type: *smsproxy.ValidationError,  return the HTTP Status 400.
    * If there is a different error, return the HTTP Status 500.
 3. If everything is OK, return the HTTP Status `202` and serialize the `SendingResult` from `smsproxy/api.go`, sending it as a Response Body.

# Application and tests 
 
## Running app locally

1. `go build -o fastSmsingProxy`
2. `./fastSmsingProxy -port=8080`
 
## Running all tests
 
 ```
     go test ./... -v -count=1 -race
 ```

## Linter used:

`golangci-lint run`

