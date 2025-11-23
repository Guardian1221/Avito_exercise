package api

import (
    "encoding/json"
    "net/http"
)

type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

func writeErrorJSON(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(ErrorResponse{
        Error: ErrorDetail{Code: code, Message: message},
    })
}
