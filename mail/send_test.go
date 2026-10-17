package mail

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// fakeSMTP accepts one session on a local port, answers every command with
// success and returns what the DATA section contained.
func fakeSMTP(t *testing.T) (port string, data <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	out := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		reply := func(s string) { conn.Write([]byte(s + "\r\n")) }
		reply("220 fake")
		var body strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "DATA"):
				reply("354 go ahead")
				for {
					l, _ := r.ReadString('\n')
					if l == ".\r\n" {
						break
					}
					body.WriteString(l)
				}
				out <- body.String()
				reply("250 ok")
			case strings.HasPrefix(cmd, "QUIT"):
				reply("221 bye")
				return
			default:
				reply("250 ok")
			}
		}
	}()
	_, port, _ = net.SplitHostPort(ln.Addr().String())
	return port, out
}

func TestSend_PlainSMTP(t *testing.T) {
	port, data := fakeSMTP(t)
	s := &SMTPSender{Host: "127.0.0.1", Port: port, From: "blog@example.com"}
	if err := s.Send("reader@example.com", "Your code", "123456"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := <-data; !strings.Contains(got, "To: reader@example.com\r\n") || !strings.HasSuffix(got, "123456\r\n") {
		t.Fatalf("unexpected message:\n%s", got)
	}
}

func TestSend_RejectsHeaderInjection(t *testing.T) {
	s := &SMTPSender{Host: "127.0.0.1", Port: "1", From: "blog@example.com"}
	if err := s.Send("a@example.com\r\nBcc: x@example.com", "s", "b"); err == nil {
		t.Fatal("expected a header injection error before dialling")
	}
}

func TestAuth(t *testing.T) {
	if (&SMTPSender{}).auth() != nil {
		t.Error("no user should mean no auth")
	}
	if (&SMTPSender{User: "u", Password: "p", Host: "h"}).auth() == nil {
		t.Error("a user should enable PLAIN auth")
	}
}
