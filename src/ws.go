package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type websocketConn struct {
	conn net.Conn
	r    *bufio.Reader
	mu   sync.Mutex
}

func dialWebSocket(rawURL string, insecure bool) (*websocketConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, fmt.Errorf("unsupported websocket scheme %q", u.Scheme)
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	d := net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	conn, err := d.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "wss" {
		serverName := u.Hostname()
		conn = tls.Client(conn, &tls.Config{ServerName: serverName, InsecureSkipVerify: insecure}) //nolint:gosec // optional Kindle CA compatibility knob
		if err := conn.(*tls.Conn).Handshake(); err != nil {
			conn.Close()
			return nil, err
		}
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\nUser-Agent: kindle-dashboard-native\r\n\r\n", path, u.Host, key)
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return nil, err
	}

	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, &http.Request{Method: "GET"})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: %s", resp.Status)
	}
	if accept := resp.Header.Get("Sec-WebSocket-Accept"); accept != websocketAccept(key) {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: bad accept header")
	}
	return &websocketConn{conn: conn, r: r}, nil
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func (c *websocketConn) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *websocketConn) WriteText(payload []byte) error {
	return c.writeFrame(0x1, payload)
}

func (c *websocketConn) writeFrame(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var b bytes.Buffer
	b.WriteByte(0x80 | opcode)
	plen := len(payload)
	switch {
	case plen < 126:
		b.WriteByte(0x80 | byte(plen))
	case plen <= 0xffff:
		b.WriteByte(0x80 | 126)
		var tmp [2]byte
		binary.BigEndian.PutUint16(tmp[:], uint16(plen))
		b.Write(tmp[:])
	default:
		b.WriteByte(0x80 | 127)
		var tmp [8]byte
		binary.BigEndian.PutUint64(tmp[:], uint64(plen))
		b.Write(tmp[:])
	}
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	b.Write(mask[:])
	for i, v := range payload {
		b.WriteByte(v ^ mask[i%4])
	}
	_, err := c.conn.Write(b.Bytes())
	return err
}

func (c *websocketConn) ReadMessage() ([]byte, error) {
	var message []byte
	for {
		fin, opcode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x0, 0x1, 0x2: // continuation/text/binary
			message = append(message, payload...)
			if fin {
				return message, nil
			}
		case 0x8: // close
			_ = c.writeFrame(0x8, nil)
			return nil, io.EOF
		case 0x9: // ping
			_ = c.writeFrame(0xA, payload)
		case 0xA: // pong
			continue
		default:
			continue
		}
	}
}

func (c *websocketConn) readFrame() (bool, byte, []byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(c.r, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin := hdr[0]&0x80 != 0
	opcode := hdr[0] & 0x0f
	masked := hdr[1]&0x80 != 0
	plen := uint64(hdr[1] & 0x7f)
	if plen == 126 {
		var ext [2]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return false, 0, nil, err
		}
		plen = uint64(binary.BigEndian.Uint16(ext[:]))
	} else if plen == 127 {
		var ext [8]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return false, 0, nil, err
		}
		plen = binary.BigEndian.Uint64(ext[:])
	}
	if plen > 16*1024*1024 {
		return false, 0, nil, fmt.Errorf("websocket frame too large: %d", plen)
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.r, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload := make([]byte, int(plen))
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}
