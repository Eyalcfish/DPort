#include <stdio.h>
#include "dport.h"

int main() {
    DConnection* conn = connect_dconnection("example_port");
    printf("DEBUG: Flag address: %p | Data address: %p\n", 
       (void*)((char*)conn->shm_ptr - 1), conn->shm_ptr);
    DMessage msg;
    msg.size = 13;
    msg.data = "Hello, World!";
    write_to_dconnection(conn, &msg);
    close_dconnection(conn);
    return 0;
}