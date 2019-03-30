// +build !windows

package asio

/*
#include <fcntl.h>

int fd_valid (int fd)
{
#ifdef _WIN32
  return _get_osfhandle(fd);
#else
  return fcntl (fd, F_GETFD);
#endif
}
*/
import "C"

func FD_VALID(fd int) bool {

	return C.fd_valid(C.int(fd)) != -1;
}
