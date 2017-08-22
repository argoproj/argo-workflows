import socket
import traceback
import sys

patched = False

def az_patch():
    # make sure we only do once
    global patched
    if patched:
        return
    else:
        patched = True

    orig_socket_init = socket.socket.__init__

    def new_socket_init_with_keep_alive(self, *args, **kwargs):
        ret = orig_socket_init(self, *args, **kwargs)

        # we only do set keep alive for AF_INET + SOCK_STREAM
        is_af_inet = True
        is_sock_stream = True
        if args:
            if len(args) > 0:
                if args[0] != socket.AF_INET:
                    is_af_inet = False
            if len(args) > 1:
                if args[1] != socket.SOCK_STREAM:
                    is_sock_stream = False
        if kwargs:
            if kwargs.get('family', socket.AF_INET) != socket.AF_INET:
                is_af_inet = False
            if kwargs.get('type', socket.SOCK_STREAM) != socket.SOCK_STREAM:
                is_sock_stream = False
        if is_af_inet and is_sock_stream:
            try:
                self.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
            except Exception as e:
                sys.stderr.write('cannot set socket.SO_KEEPALIVE to 1. {}'.format(traceback.format_exc(e)))
                sys.stderr.flush()
                pass
        else:
            # sys.stderr.write('NOT set socket.SO_KEEPALIVE to 1')
            # sys.stderr.flush()
            pass

        return ret

    socket.socket.__init__ = new_socket_init_with_keep_alive
