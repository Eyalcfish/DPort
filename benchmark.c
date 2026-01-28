#include <stdio.h>
#include <string.h>
#include <windows.h>
#include "dport.h"

// --- High-Res Timer Implementation ---
long long get_nanos() {
    static LARGE_INTEGER frequency;
    static int initialized = 0;
    if (!initialized) {
        QueryPerformanceFrequency(&frequency);
        initialized = 1;
    }
    LARGE_INTEGER counter;
    QueryPerformanceCounter(&counter);
    return (long long)((counter.QuadPart * 1000000000LL) / frequency.QuadPart);
}

int main(int argc, char** argv) {
    if (argc < 2) {
        printf("Usage: %s <server|client>\n", argv[0]);
        return 1;
    }

    if (strcmp(argv[1], "server") == 0) {
        // --- SERVER MODE ---
        printf("[SERVER] Initializing DPort...\n");
        DConnection* conn = create_dconnection("example_port", 1024, 1);
        
        // WARM-UP: Get the CPU out of power-saving mode
        for(int i = 0; i < 10000; i++) {
            DMessage msg = wait_for_new_message_from_dconnection(conn);
            write_to_dconnection(conn, &msg);
            free(msg.data);
        }

        printf("[SERVER] Ready. Waiting for 1,000,000 pings...\n");
        for(int i = 0; i < 1000000; i++) {
            // Wait for message from client
            DMessage msg = wait_for_new_message_from_dconnection(conn);

            // Immediately send it back (Ping-Pong)
            write_to_dconnection(conn, &msg);
            free(msg.data);
        }
        printf("[SERVER] Benchmark complete.\n");
        close_dconnection(conn);
        
    } else if (strcmp(argv[1], "client") == 0) {
        // --- CLIENT MODE ---
        printf("[CLIENT] Connecting to DPort...\n");
        DConnection* conn = connect_dconnection("example_port");
        
        DMessage msg;
        char* data = "ping";
        msg.size = strlen(data) + 1;
        msg.data = data;
        
        printf("[CLIENT] Starting Benchmark (1,000,000 iterations)...\n");
        
        // WARM-UP: Get the CPU out of power-saving mode
        for(int i = 0; i < 10000; i++) {
            write_to_dconnection(conn, &msg);
            wait_for_new_message_from_dconnection(conn);
        }
        
        long long start = get_nanos();
        for(int i = 0; i < 1000000; i++) {
            write_to_dconnection(conn, &msg);
            DMessage response = wait_for_new_message_from_dconnection(conn);
            free(response.data);
        }
        long long end = get_nanos();

        printf("\n--- RESULTS ---\n");
        printf("Total Time:    %lld ns\n", (end - start));
        printf("Avg Round-Trip: %lld ns\n", (end - start) / 1000000);
        printf("One-Way Latency: ~%lld ns\n", ((end - start) / 1000000) / 2);
        
        close_dconnection(conn);
    }

    return 0;
}