package main

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

const (
	ip       = "127.0.0.1"
	port     = 8080
	backlog  = 128
	bufSize  = 4096
	maxEvBuf = 128
)

func ListeningForConnections() error {
	// 1) Create listen socket
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	// Nice-to-have for restarts
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return err
	}

	// Must be non-blocking for event-loop style servers
	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}

	sa := &unix.SockaddrInet4{Port: port}
	copy(sa.Addr[:], net.ParseIP(ip).To4())

	if err := unix.Bind(fd, sa); err != nil {
		return err
	}
	if err := unix.Listen(fd, backlog); err != nil {
		return err
	}
	fmt.Printf("Listening on %s:%d (listenFD=%d)\n", ip, port, fd)

	// 2) Create kqueue
	kq, err := unix.Kqueue()
	if err != nil {
		return err
	}
	defer unix.Close(kq)

	// 3) Register listen fd for read events (incoming connections)
	if err := kqAddRead(kq, fd); err != nil {
		return err
	}

	// 4) Event loop: kevent waits and returns ready fds [web:139]
	events := make([]unix.Kevent_t, maxEvBuf)
	buf := make([]byte, bufSize)

	for {
		n, err := unix.Kevent(kq, nil, events, nil) // nil timeout => block
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			ev := events[i]
			readyFD := int(ev.Ident)

			// If it's the listen socket: accept all pending connections
			if readyFD == fd {
				for {
					cfd, _, err := unix.Accept(fd)
					if err != nil {
						if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
							break
						}
						// transient accept error; keep server alive
						break
					}

					_ = unix.SetNonblock(cfd, true)
					fmt.Printf("Accepted clientFD=%d\n", cfd)

					// Register client for read events
					_ = kqAddRead(kq, cfd)
				}
				continue
			}

			// If it's a client socket: read and echo
			if (ev.Flags & unix.EV_EOF) != 0 {
				// peer hung up
				_ = kqDelRead(kq, readyFD)
				_ = unix.Close(readyFD)
				continue
			}

			for {
				nr, rerr := unix.Read(readyFD, buf)
				if nr > 0 {
					// naive echo; ignores partial writes (fine for small payload demos)
					_, _ = unix.Write(readyFD, buf[:nr])
				}

				if rerr != nil {
					if rerr == unix.EAGAIN || rerr == unix.EWOULDBLOCK {
						break
					}
					_ = kqDelRead(kq, readyFD)
					_ = unix.Close(readyFD)
					break
				}

				if nr == 0 {
					// EOF
					_ = kqDelRead(kq, readyFD)
					_ = unix.Close(readyFD)
					break
				}
			}
		}
	}
}

func kqAddRead(kq int, fd int) error {
	// EVFILT_READ = "wake me when fd is readable" (for listen socket: connection pending) [web:131]
	ch := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_ADD | unix.EV_ENABLE,
	}
	_, err := unix.Kevent(kq, []unix.Kevent_t{ch}, nil, nil)
	return err
}

func kqDelRead(kq int, fd int) error {
	ch := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_DELETE,
	}
	_, err := unix.Kevent(kq, []unix.Kevent_t{ch}, nil, nil)
	return err
}
