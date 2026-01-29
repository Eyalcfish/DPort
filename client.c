#include <stdio.h>
#include "dport.h"

int main(int argc, char** argv) {
    DConnection* conn = connect_dconnection("example_port");

    DMessage msg;
    char buffer[] = "asd";
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