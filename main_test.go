package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var handler http.Handler

func TestMain(m *testing.M) {
	logger = log.New(ioutil.Discard, "http: ", log.LstdFlags)
	handler = initHTTPHandler()
	db = initDB()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

func TestPostMessages(t *testing.T) {
	clearTable()

	body := []byte(`{"sender": "anson","conversation_id": 1234, "message": "I'm a teapot"}`)
	req := httptest.NewRequest("POST", "/messages/", bytes.NewBuffer(body))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	checkResponseCode(t, http.StatusOK, res.Code)
}

func TestPostMessagesWithoutConversationID(t *testing.T) {
	clearTable()

	body := []byte(`{"sender": "anson", "message": "I'm a teapot"}`)
	req := httptest.NewRequest("POST", "/messages/", bytes.NewBuffer(body))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	checkResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestGetConversations(t *testing.T) {
	clearTable()

	if _, err := db.Exec(`
		INSERT INTO messages (sender, conversation_id, message)
		VALUES
		('anson', 1234, 'I''m a teapot'),
		('david', 1234, 'Short and stout'),
		('mrs.potts', 4321, 'Now, Chip, I won''t have you making up such wild stories.')
	`); err != nil {
		t.Errorf("Failed to insert test data")
	}

	req := httptest.NewRequest("GET", "/conversations/1234", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	checkResponseCode(t, http.StatusOK, res.Code)

	var m map[string]interface{}
	json.Unmarshal(res.Body.Bytes(), &m)

	if m["id"] != float64(1234) {
		t.Errorf(
			"Expected the id to remain the same (%v). Got %v",
			1234, m["id"],
		)
	}

	if len(m["messages"].([]interface{})) != 2 {
		t.Errorf(
			"Expected messages to contain %v messages. Got %v",
			2, len(m["messages"].([]interface{})),
		)
	}
}

func TestGetConversationsWithoutID(t *testing.T) {
	clearTable()

	req := httptest.NewRequest("GET", "/conversations/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	checkResponseCode(t, http.StatusNotFound, res.Code)
}

func clearTable() {
	db.Exec("DELETE FROM messages")
	db.Exec("ALTER SEQUENCE messages_id_seq RESTART WITH 1")
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
