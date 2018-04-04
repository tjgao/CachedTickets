package ws

import (
	"bytes"
	"encoding/gob"
)

type MessageType int

const (
	RegisterRespType MessageType = iota
	TaskRequestType
	TaskResultType
	LeaveReqType
	LeaveRespType
)

const (
	RetrieveDataSuccessfully int = iota
	FailedToAccessURL
	FailedToReadFromResponse
)

// Slave expects messages like this and then it can parse body field according to the specified id
type Message struct {
	ID      MessageType
	TransID int64
	Body    []byte
}

// RegisterResp: When slave connects it should expect this as the first message from master
type RegisterResp struct {
	Code        int
	Description string
}

// Task: server will ask slave to do some task
type Task struct {
	TargetURL string
}

// TaskResult: when task is done, slave replies to master
type TaskResult struct {
	Result      []byte
	Code        int
	Description string
}

// LeaveReq: the slave wants to exit
type LeaveReq struct {
}

func Encode(msg *Message) ([]byte, error) {
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	if err := e.Encode(*msg); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func EncodeTaskResult(msg *TaskResult) ([]byte, error) {
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	if err := e.Encode(*msg); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func DecodeTaskResult(buf []byte, msg *TaskResult) error {
	var newBuf = bytes.NewBuffer(buf)
	e := gob.NewDecoder(newBuf)
	return e.Decode(*msg)
}

func Decode(buf []byte, msg *Message) error {
	var newBuf = bytes.NewBuffer(buf)
	e := gob.NewDecoder(newBuf)
	return e.Decode(*msg)
}

func DecodeRegisterResp(buf []byte, msg *RegisterResp) error {
	var newBuf = bytes.NewBuffer(buf)
	e := gob.NewDecoder(newBuf)
	return e.Decode(*msg)
}

func EncodeTask(task *Task) ([]byte, error) {
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	if err := e.Encode(*task); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func DecodeTask(buf []byte, msg *Task) error {
	var newBuf = bytes.NewBuffer(buf)
	e := gob.NewDecoder(newBuf)
	return e.Decode(*msg)
}
