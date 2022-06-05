package legacy

import (
	"encoding/json"
	"log"
	"net/http"
)

// Global responses
var responseOK = Response{ResponseCode: 1, ResponseMessage: "OK"}
var response9 = Response{ResponseCode: 9, ResponseMessage: "Missing parameter/s."}

// 200s - Admin Settings responses
var response209 = Response{ResponseCode: 209, ResponseMessage: "Provided old password does not correspond with the hash in database."}

// 900s - Generic responses
var response900 = Response{ResponseCode: 900, ResponseMessage: "Operation failed on Backend side."}

func responseMissingParameterWithLog(parameter string, w http.ResponseWriter) {
	payload, err := json.Marshal(MissingParameterResponse{response9, parameter})
	if err != nil {
		log.Println("Error:", "function responseMissingParameterWithLog", "json.Marshal", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println("Error:", "function responseMissingParameterWithLog", "w.Write", err)
	}
}

func responseWithLog(responseTemplate Response, w http.ResponseWriter) {
	payload, err := json.Marshal(responseTemplate)
	if err != nil {
		log.Println("Error:", "function responseWithLog", "json.Marshal", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println("Error:", "function responseWithLog", "w.Write", err)
	}
}
