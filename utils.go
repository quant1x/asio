package asio

import (
	"errors"
	"net"
	"os"
	"reflect"
)

// filer describes an object that has ability to return os.File.
type filer interface {
	// File returns a copy of object's file descriptor.
	File() (*os.File, error)
}

func handle(x interface{}) (socket_t, error) {
	f, ok := x.(filer)
	if !ok {
		return -1, errors.New("not filter")
	}

	// Get a copy of fd.
	file, err := f.File()
	if err != nil {
		return -1, err
	}
	return int(file.Fd()), nil
}

func handle3(conn net.Conn) (socket_t, error) {
	tcpConn := conn.(*net.TCPConn)
	file, err := tcpConn.File()
	if err != nil {
		return -1, err
	}
	return int(file.Fd()), nil
}

func socketFD(conn net.Conn) int {
	//tls := reflect.TypeOf(conn.UnderlyingConn()) == reflect.TypeOf(&tls.Conn{})
	// Extract the file descriptor associated with the connection
	//connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	//if tls {
	//  tcpConn = reflect.Indirect(tcpConn.Elem())
	//}
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}