package zero

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

// Conn wrap net.Conn
type Conn struct {
	sid        string
	rawConn    net.Conn
	sendCh     chan []byte
	done       chan error
	name       string
	messageCh  chan *Message

	hbTimer    *time.Timer
	hbTicker   *time.Ticker
	hbInterval time.Duration
	hbTimeout  time.Duration
	hbNext <-chan  time.Time
}

// GetName Get conn name
func (c *Conn) GetName() string {
	return c.name
}

// NewConn create new conn
func NewConn(c net.Conn, hbInterval time.Duration, hbTimeout time.Duration) *Conn {
	conn := &Conn{
		rawConn:    c,
		sendCh:     make(chan []byte, 100),
		done:       make(chan error),
		messageCh:  make(chan *Message, 100),
		hbInterval: hbInterval,
		hbTimeout:  hbTimeout,
	}

	conn.name = c.RemoteAddr().String()
	conn.hbTimer = time.NewTimer(conn.hbInterval)
	conn.hbTicker = time.NewTicker(conn.hbInterval)

	if conn.hbInterval == 0 {
		conn.hbTimer.Stop()
		conn.hbTicker.Stop()
	}

	return conn
}

// Close close connection
func (c *Conn) Close() {
	c.hbTimer.Stop()
	c.hbTicker.Stop()
	c.rawConn.Close()
}

// SendMessage send message
func (c *Conn) SendMessage(msg *Message) error {
	pkg, err := Encode(msg)
	if err != nil {
		return err
	}

	c.sendCh <- pkg
	return nil
}

// writeCoroutine write coroutine
func (c *Conn) writeCoroutine(ctx context.Context) {
	hbData := make([]byte, 0)

	for {
		select {
		case <-ctx.Done():
			return

		case pkt := <-c.sendCh:

			if pkt == nil {
				continue
			}

			if _, err := c.rawConn.Write(pkt); err != nil {
				c.done <- err
				log.Println("write failed:", err)
			}
			//c.hbTicker.Stop()
			c.hbNext= time.Tick(c.hbInterval)

		case <-c.hbNext:
			hbMessage := NewMessage(MsgHeartbeat, hbData)
			c.SendMessage(hbMessage)
			log.Println("sending hb to client...")
		}
	}
}

// readCoroutine read coroutine
func (c *Conn) readCoroutine(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return

		default:
			// 设置超时
			if c.hbInterval > 0 {
				err := c.rawConn.SetReadDeadline(time.Now().Add(c.hbTimeout))
				if err != nil {
					c.done <- err
					continue
				}
			}
			// 读取长度
			buf := make([]byte, 4)
			_, err := io.ReadFull(c.rawConn, buf)
			if err != nil {
				c.done <- err
				log.Println("read failed:", err)
				continue
			}

			bufReader := bytes.NewReader(buf)

			var dataSize int32
			err = binary.Read(bufReader, binary.LittleEndian, &dataSize)
			if err != nil {
				c.done <- err
				continue
			}

			// 读取数据
			databuf := make([]byte, dataSize)
			_, err = io.ReadFull(c.rawConn, databuf)
			if err != nil {
				c.done <- err
				continue
			}

			// 解码
			msg, err := Decode(databuf)
			if err != nil {
				c.done <- err
				continue
			}

			// 设置心跳timer
			if c.hbInterval > 0 {
				c.hbTimer.Reset(c.hbInterval)
				log.Println("reset Ticker")
				//c.hbTicker.Stop()
				c.hbNext= time.Tick(c.hbInterval)

			}

			if msg.GetID() == MsgHeartbeat {
				continue
			}

			c.messageCh <- msg
		}
	}
}
