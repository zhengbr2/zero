package zero

import (
		"net"
	"testing"
	"time"
	"log"
)


func TestService(t *testing.T) {
	log.SetFlags(log.Ldate|log.Lmicroseconds)
	host := "127.0.0.1:18787"

	ss, err := NewSocketService(host)
	if err != nil {
		return
	}

	// ss.SetHeartBeat(5*time.Second, 30*time.Second)

	ss.RegMessageHandler(HandleMessage)
	ss.RegConnectHandler(HandleConnect)
	ss.RegDisconnectHandler(HandleDisconnect)
	ss.SetHeartBeat(time.Second * 1, time.Second * 1)

	go NewClientConnect()

	timer := time.NewTimer(time.Second * 7)
	go func() {
		<-timer.C
		ss.Stop("stop service")
		t.Log("service stoped")
	}()

	t.Log("service running on " + host)

	ss.Serv()
}

func HandleMessage(s *Session, msg *Message) {
	log.Println("receive msgID:", msg)
	log.Println("receive data:", string(msg.GetData()))
}

func HandleDisconnect(s *Session, err error) {
	log.Println(s.GetConn().GetName() + " lost.",err)
}

func HandleConnect(s *Session) {
	log.Println(s.GetConn().GetName() + " connected.")
}

func NewClientConnect() {
	host := "127.0.0.1:18787"
	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return
	}


	msg := NewMessage(1, []byte("Hello Zero!"))
	data, err := Encode(msg)
	if err != nil {
		return
	}
	conn.Write(data)
	log.Println("client sending msg 1")

	time.Sleep(time.Second * 3)

	msg = NewMessage(1, []byte("Hello Zero!"))
	data, _= Encode(msg)
	if err != nil {
		return
	}
	conn.Write(data)
	log.Println("client sending msg 2")

}
