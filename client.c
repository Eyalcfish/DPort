#include <stdio.h>
#include "dport.h"

int main(int argc, char** argv) {
    DConnection* conn = connect_dconnection("example_port");

    printf("DEBUG: Flag address: %p | Data address: %p\n", 
       (void*)((char*)conn->shm_ptr - 1), conn->shm_ptr);

    DMessage msg;

    char buffer[] = "asd"; // This is WRITABLE
    msg.data = buffer;
    msg.size = sizeof(buffer);
    for (int i = 0 ;i < 1000; i++) {
        ((char*)msg.data)[2] = '0'+i;
        printf("Sending message: %s\n", (char*)msg.data);
        write_to_dconnection(conn, &msg);
    }

    close_dconnection(conn);
    return 0;
}