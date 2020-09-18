package client

const DefaultUIPort = "8080" // Port number for exchanging messages with the user interface

type ClientMessage struct {
	Contents string `json:"contents"`
}
