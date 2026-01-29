#include <stdio.h>
#include "dport.h"

int main() {
    DConnection* conn = create_dconnection("example_port", 1024, EVENT_BASED_CONNECTION);
    
    printf("DEBUG: Flag address: %p | Data address: %p\n", (void*)((char*)conn->shm_ptr - 1), conn->shm_ptr);
    
    for(int i = 0 ;i < 1000; i++) {
        DMessage msg = wait_for_new_message_from_dconnection(conn);
        printf("Size: %zu | Data: %s\n", msg.size, (char*)msg.data);
    }

    close_dconnection(conn);
    return 0;
}