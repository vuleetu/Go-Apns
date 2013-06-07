package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"
    "errors"
)

var WRITE_FAILED = errors.New("Write data to apns failed")

type Logger interface {
    Info(...interface{})
    Debug(...interface{})
    Warn(...interface{})
    Error(...interface{})
}

type Stdlog byte

func (*Stdlog) Info(args ...interface{}) {
    fmt.Println(args...)
}

func (*Stdlog) Debug(args ...interface{}) {
    fmt.Println(args...)
}

func (*Stdlog) Warn(args ...interface{}) {
    fmt.Println(args...)
}

func (*Stdlog) Error(args ...interface{}) {
    fmt.Println(args...)
}

type Notification struct {
	DeviceToken        string
	Identifier         uint32
	ExpireAfterSeconds int

	Payload *Payload
}

// An Apn contain a ErrorChan channle when connected to apple server. When a notification sent wrong, you can get the error infomation from this channel.
type Apn struct {
	server  string
	conf    *tls.Config
	sendChan  chan *sendArg
    log Logger
}

type ApnConn struct {
    conn *tls.Conn
    log Logger
}

func (a *ApnConn) Close() error {
    a.log.Info("Close connection now")
	if a.conn == nil {
		return nil
	}
	conn := a.conn
	a.conn = nil
	return conn.Close()
}

func (a *ApnConn) Send(notification *Notification) error {
	tokenbin, err := hex.DecodeString(notification.DeviceToken)
	if err != nil {
		return fmt.Errorf("convert token to hex error: %s", err)
	}

	payloadbyte, _ := json.Marshal(notification.Payload)
	expiry := time.Now().Add(time.Duration(notification.ExpireAfterSeconds) * time.Second).Unix()

	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(1))
	binary.Write(buffer, binary.BigEndian, uint32(notification.Identifier))
	binary.Write(buffer, binary.BigEndian, uint32(expiry))
	binary.Write(buffer, binary.BigEndian, uint16(len(tokenbin)))
	binary.Write(buffer, binary.BigEndian, tokenbin)
	binary.Write(buffer, binary.BigEndian, uint16(len(payloadbyte)))
	binary.Write(buffer, binary.BigEndian, payloadbyte)
	pushPackage := buffer.Bytes()

	_, err = a.conn.Write(pushPackage)
	if err != nil {
		a.log.Error("write socket error:", err)
        return WRITE_FAILED
	}
	return nil
}


// New Apn with cert_filename and key_filename.
func New(cert_filename, key_filename, server string, timeout time.Duration) (*Apn, error) {
	cert, err := tls.LoadX509KeyPair(cert_filename, key_filename)
	if err != nil {
		return nil, err
	}

	certificate := []tls.Certificate{cert}
	conf := &tls.Config{
		Certificates: certificate,
	}

    stdlog := Stdlog(1)
	ret := &Apn{
		server:    server,
		conf:      conf,
		sendChan:  make(chan *sendArg),
        log: &stdlog,
	}

	go sendLoop(ret)
	return ret, err
}

// Send a notification to iOS
func (a *Apn) Send(notification *Notification) error {
	err := make(chan error)
	arg := &sendArg{
		n:   notification,
		err: err,
	}
	a.sendChan <- arg
	return nil
}

type sendArg struct {
	n   *Notification
	err chan<- error
}

func (a *Apn) NewApnConn() (*ApnConn, error) {
	conn, err := net.Dial("tcp", a.server)
	if err != nil {
		return nil, fmt.Errorf("connect to server error: %d", err)
	}

	var client_conn *tls.Conn = tls.Client(conn, a.conf)
	err = client_conn.Handshake()
	if err != nil {
		return nil, fmt.Errorf("handshake server error: %s", err)
	}

    apn_conn := &ApnConn{client_conn, a.log}
	//go readError(apn_conn)

    return apn_conn, nil
}

func sendLoop(apn *Apn) {
    c := make([]*sendArg, 0, 100)

    recvChan := make(chan *sendArg)

    go sendLoop0(apn, recvChan)

    for {
        if len(c) > 0 {
            select {
            case arg := <- apn.sendChan:
                c = append(c, arg)
            case recvChan <- c[0]:
                c = c[1:]
            }
        } else {
            arg := <- apn.sendChan
            c = append(c, arg)
        }
    }

    apn.log.Error("Fatal, Notification channel closed")
}

func sendLoop0(apn *Apn, recvChan <-chan *sendArg) {
    //maximum pool size
    pool := make(chan byte, 100)
    for {
        apn.log.Info("Waiting for new notification, current pool size is", len(pool))
        arg := <-recvChan
        apn.log.Info("Received notification from channel, device:", arg.n.DeviceToken)
        pool <- 1
        apn.log.Info("Create new goroutine to send notication")
        go sendLoop1(pool, apn, arg)
    }
}

func sendLoop1(pool <-chan byte, apn *Apn, arg *sendArg) {
    defer func() {
        <-pool
    }()

    for {
        apn.log.Info("Connecting to apns")
        apnconn, err := apn.NewApnConn()
        if err != nil {
            apnconn.Close()
            apn.log.Error("Connect to apns failed", err)
            time.Sleep(5*time.Second)
            continue
        }

        apnconn.log.Info("Connected to apns success")
        apnconn.log.Info("Sending data to apns")
        err = apnconn.Send(arg.n)
        if err != WRITE_FAILED {
            apn.log.Info("Sending data to apns success")
            apnconn.Close()
            break
        }
        apnconn.log.Info("Write data to apns failed, reconnecting now")
        apnconn.Close()
    }

    apn.log.Info("Send notification success")
}

func readError(a *ApnConn) {
	p := make([]byte, 6, 6)
    a.log.Info("Reading error data from apns")
    n, err := a.conn.Read(p)
    if err != nil {
        a.log.Warn("Read data from apns failed:", err)
        return
    }

    a.log.Info("Reading error data from apns success", p[:n])
    //there is something wrong, so we need to log it
    a.log.Error(NewNotificationError(p[:n], err))
}
