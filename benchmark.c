#include <stdio.h>
#include "dport.h"
#include <windows.h>

long long get_nanos() {
    static LARGE_INTEGER frequency;
    static int initialized = 0;
    if (!initialized) {
        QueryPerformanceFrequency(&frequency);
        initialized = 1;
    }
    LARGE_INTEGER counter;
    QueryPerformanceCounter(&counter);
    // Convert to nanoseconds: (counter * 1,000,000,000) / frequency
    return (long long)((counter.QuadPart * 1000000000LL) / frequency.QuadPart);
}

void gemini_benchmarking(DConnection* conn, DMessage msg) {
    for(int i = 0; i < 10000; i++) { // Warm-up
        write_to_dconnection(conn, &msg);
        wait_for_new_message_from_dconnection(conn);
    }
    // Pseudo-code for Benchmarking
    long long start = get_nanos();
    for(int i = 0; i < 10000000; i++) {
        write_to_dconnection(conn, &msg);
        wait_for_new_message_from_dconnection(conn); // Wait for Ack
    }
    long long end = get_nanos();
    printf("Average Latency: %lld ns\n", (end - start) / 10000000);
}

int main(int argc, char** argv) {
    DConnection* conn = create_dconnection("example_port", 1024);
    // DConnection* conn = connect_dconnection("example_port");

    printf("DEBUG: Flag address: %p | Data address: %p\n", 
       (void*)((char*)conn->shm_ptr - 1), conn->shm_ptr);

    DMessage msg;
    msg.size = strlen(argv[1]) + 1;
    msg.data = argv[1];

    gemini_benchmarking(conn, msg);

    close_dconnection(conn);
    return 0;
}