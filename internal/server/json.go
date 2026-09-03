package server

import (
	"errors"
	"io"
	"net/http"

	agentforumv1 "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1"
	"github.com/go-go-golems/agentforum/internal/service"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// writeProtoJSON marshals msg with protojson (camelCase field names, enum
// names as strings, int64 as strings) and writes it with status.
func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) error {
	data, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return nil
}

// decodeProtoJSON reads the request body into msg. Unknown fields are
// discarded (decision R2 in the design doc): newer clients may send fields
// this server does not know yet, and rejecting them would couple deploy
// order to schema order.
func decodeProtoJSON(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return err
	}
	dec := protojson.UnmarshalOptions{DiscardUnknown: true}
	return dec.Unmarshal(body, msg)
}

// writeError emits the standard Error envelope.
func writeError(w http.ResponseWriter, status int, code, message string) {
	_ = writeProtoJSON(w, status, &agentforumv1.Error{
		SchemaVersion: 1,
		Code:          code,
		Message:       message,
	})
}

// writeServiceError maps a service-layer error to the HTTP status table
// (design doc §5.3.1). Unknown errors become 500 with the message elided.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, "unauthenticated", err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, "invalid_argument", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
