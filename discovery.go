/*
Copyright (c) 2013-2016 The btcsuite developers

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package spvwallet

import (
	"encoding/binary"
	"errors"
	"net"
	"github.com/btcsuite/go-socks/socks"
	"encoding/hex"
	"crypto/rand"
	"io"
)

const (
	torSucceeded         = 0x00
	torGeneralError      = 0x01
	torNotAllowed        = 0x02
	torNetUnreachable    = 0x03
	torHostUnreachable   = 0x04
	torConnectionRefused = 0x05
	torTTLExpired        = 0x06
	torCmdNotSupported   = 0x07
	torAddrNotSupported  = 0x08

	protocolVersion = 5

	authNone             = 0
	authGssApi           = 1
	authUsernamePassword = 2
	authUnavailable      = 0xff
)

var (
	// ErrTorInvalidAddressResponse indicates an invalid address was
	// returned by the Tor DNS resolver.
	ErrTorInvalidAddressResponse = errors.New("invalid address response")

	// ErrTorInvalidProxyResponse indicates the Tor proxy returned a
	// response in an unexpected format.
	ErrTorInvalidProxyResponse = errors.New("invalid proxy response")

	// ErrTorUnrecognizedAuthMethod indicates the authentication method
	// provided is not recognized.
	ErrTorUnrecognizedAuthMethod = errors.New("invalid proxy authentication method")

	ErrAuthFailed             = errors.New("authentication failed")
	ErrInvalidProxyResponse   = errors.New("invalid proxy response")
	ErrNoAcceptableAuthMethod = errors.New("no acceptable authentication method")

	torStatusErrors = map[byte]error{
		torSucceeded:         errors.New("tor succeeded"),
		torGeneralError:      errors.New("tor general error"),
		torNotAllowed:        errors.New("tor not allowed"),
		torNetUnreachable:    errors.New("tor network is unreachable"),
		torHostUnreachable:   errors.New("tor host is unreachable"),
		torConnectionRefused: errors.New("tor connection refused"),
		torTTLExpired:        errors.New("tor TTL expired"),
		torCmdNotSupported:   errors.New("tor command not supported"),
		torAddrNotSupported:  errors.New("tor address type not supported"),
	}
)

// torLookupIP uses Tor to resolve DNS via the SOCKS extension they provide for
// resolution over the Tor network. Tor itself doesn't support ipv6 so this
// doesn't either.
func TorResolve(host string, p *socks.Proxy) ([]net.IP, error) {
	conn, err := net.Dial("tcp", p.Addr)
	if err != nil {
		return nil, err
	}

	var user, pass string
	if p.TorIsolation {
		var b [16]byte
		_, err := io.ReadFull(rand.Reader, b[:])
		if err != nil {
			conn.Close()
			return nil, err
		}
		user = hex.EncodeToString(b[0:8])
		pass = hex.EncodeToString(b[8:16])
	} else {
		user = p.Username
		pass = p.Password
	}
	buf := make([]byte, 32+len(host)+len(user)+len(pass))

	// Initial greeting
	buf[0] = protocolVersion
	if user != "" {
		buf = buf[:4]
		buf[1] = 2 // num auth methods
		buf[2] = authNone
		buf[3] = authUsernamePassword
	} else {
		buf = buf[:3]
		buf[1] = 1 // num auth methods
		buf[2] = authNone
	}

	_, err = conn.Write(buf)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Server's auth choice

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		conn.Close()
		return nil, err
	}
	if buf[0] != protocolVersion {
		conn.Close()
		return nil, ErrInvalidProxyResponse
	}
	err = nil
	switch buf[1] {
	default:
		err = ErrInvalidProxyResponse
	case authUnavailable:
		err = ErrNoAcceptableAuthMethod
	case authGssApi:
		err = ErrNoAcceptableAuthMethod
	case authUsernamePassword:
		buf = buf[:3+len(user)+len(pass)]
		buf[0] = 1 // version
		buf[1] = byte(len(user))
		copy(buf[2:], user)
		buf[2+len(user)] = byte(len(pass))
		copy(buf[3+len(user):], pass)
		if _, err = conn.Write(buf); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err = io.ReadFull(conn, buf[:2]); err != nil {
			conn.Close()
			return nil, err
		}
		if buf[0] != 1 { // version
			err = ErrInvalidProxyResponse
		} else if buf[1] != 0 { // 0 = succes, else auth failed
			err = ErrAuthFailed
		}
	case authNone:
	// Do nothing
	}
	if err != nil {
		conn.Close()
		return nil, err
	}

	buf = make([]byte, 7+len(host))
	buf[0] = 5      // protocol version
	buf[1] = '\xF0' // Tor Resolve
	buf[2] = 0      // reserved
	buf[3] = 3      // Tor Resolve
	buf[4] = byte(len(host))
	copy(buf[5:], host)
	buf[5+len(host)] = 0 // Port 0

	_, err = conn.Write(buf)
	if err != nil {
		return nil, err
	}

	buf = make([]byte, 4)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if buf[0] != 5 {
		return nil, ErrTorInvalidProxyResponse
	}
	if buf[1] != 0 {
		if int(buf[1]) > len(torStatusErrors) {
			err = ErrTorInvalidProxyResponse
		} else {
			err := torStatusErrors[buf[1]]
			if err == nil {
				err = ErrTorInvalidProxyResponse
			}
		}
		return nil, err
	}
	if buf[3] != 1 {
		err := torStatusErrors[torGeneralError]
		return nil, err
	}

	buf = make([]byte, 4)
	bytes, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if bytes != 4 {
		return nil, ErrTorInvalidAddressResponse
	}

	r := binary.BigEndian.Uint32(buf)

	addr := make([]net.IP, 1)
	addr[0] = net.IPv4(byte(r>>24), byte(r>>16), byte(r>>8), byte(r))
	return addr, nil
}
