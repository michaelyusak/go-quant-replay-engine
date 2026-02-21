package common

import (
	"time"

	"github.com/gorilla/websocket"
)

func CloseConn(conn *websocket.Conn) error {
	err := conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
	)
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	return conn.Close()
}
