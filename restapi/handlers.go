package restapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gitlab.com/devskiller-tasks/messaging-app-golang/smsproxy"
)

func sendSmsHandler(smsProxy smsproxy.SmsProxy) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// HINT: you can use `handleError()` function when handling any error
		// 1. read SendSmsRequest from request. If error occurs, return HTTP Status 400
		// 2. try sending an SMS using `smsProxy.Send(...)`
		// if `smsProxy.Send(...)` returns error which is of type *smsproxy.ValidationError -> return HTTP Status 400
		// if it's a different error -> return HTTP Status 500
		// 3. if everything went OK, return HTTP Status 202 and serialize `SendingResult` from `smsproxy/api.go`, sending it as Response Body

		// log.Println("Received request to send SMS")

		var sendSmsRequest SendSmsRequest
		if err := json.NewDecoder(request.Body).Decode(&sendSmsRequest); err != nil {
			log.Println(errors.Wrapf(err, "cannot decode request body").Error())
			handleError(writer, http.StatusBadRequest, err)
			return
		}

		// log.Printf("Sending SMS to %s with content: %s", sendSmsRequest.PhoneNumber, sendSmsRequest.Content)

		sendingResult, err := smsProxy.Send(smsproxy.SendMessage{
			PhoneNumber: smsproxy.PhoneNumber(sendSmsRequest.PhoneNumber),
			Message:     smsproxy.Message(sendSmsRequest.Content),
		})
		if err != nil {
			if validationError, ok := err.(*smsproxy.ValidationError); ok {
				log.Println(errors.Wrapf(validationError, "validation error").Error())
				handleError(writer, http.StatusBadRequest, validationError)
				return
			}
			log.Println(errors.Wrapf(err, "cannot send SMS").Error())
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		// log.Printf("SMS sent successfully, message ID: %s", sendingResult.ID)

		responseBody, err := json.Marshal(sendingResult)
		if err != nil {
			log.Println(errors.Wrapf(err, "cannot serialize response").Error())
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		writer.WriteHeader(http.StatusAccepted)
		if _, err = writer.Write(responseBody); err != nil {
			log.Println(errors.Wrapf(err, "cannot write http response").Error())
		}
	}
}

func getSmsStatusHandler(smsProxy smsproxy.SmsProxy) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		messageID, err := getMessageID(request.URL.RequestURI())
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}
		result, err := smsProxy.GetStatus(messageID.String())
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		responseBody, err := json.Marshal(SmsStatusResponse{Status: result})
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		if _, err = writer.Write(responseBody); err != nil {
			log.Println(errors.Wrapf(err, "cannot write http response").Error())
		}
	}
}

func getMessageID(uri string) (uuid.UUID, error) {
	uriParts := strings.Split(uri, "/")
	parse, err := uuid.Parse(uriParts[2])
	return parse, err
}

func handleError(writer http.ResponseWriter, status int, err error) {
	response := HttpErrorResponse{Error: err.Error()}
	jsonBody, err := json.Marshal(response)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("Error serializing response"))
		log.Println(errors.Wrapf(err, "error serializing json response").Error())
	}
	writer.WriteHeader(status)
	_, err = writer.Write(jsonBody)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Println(errors.Wrapf(err, "error writing HTTP response").Error())
	}
}
