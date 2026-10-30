package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	maxCredentialsBytes = 4 << 10
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// decodeCredentials reads JSON credentials from the request body limited to maxCredentialsBytes.
func decodeCredentials(rspWriter http.ResponseWriter, req *http.Request) (credentialsRequest, error) {
	var creds credentialsRequest

	if err := json.NewDecoder(
		http.MaxBytesReader(rspWriter, req.Body, maxCredentialsBytes),
	).Decode(&creds); err != nil {
		return credentialsRequest{}, fmt.Errorf("json decode: %w", err)
	}

	return creds, nil
}

func writeToken(rspWriter http.ResponseWriter, status int, token string) error {
	rspWriter.Header().Set("Content-Type", "application/json")
	rspWriter.WriteHeader(status)

	if err := json.NewEncoder(rspWriter).Encode(tokenResponse{Token: token}); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}

	return nil
}
