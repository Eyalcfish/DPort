#include <stdio.h>
#include <string.h>
#include <windows.h>
#include "dport.h"

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

    const int ITERATIONS = 1000000;
    const int MSG_SIZE = sizeof(long long); // We'll send an 8-byte counter

    if (strcmp(argv[1], "server") == 0) {
        printf("[SERVER] Initializing DPort (Integrity Mode)...\n");
        // DConnection* conn = create_dconnection("example_port", 1024, EVENT_BASED_CONNECTION);
        DConnection* conn = create_dconnection("example_port", 1024, HYBRID_CONNECTION);
        // DConnection* conn = create_dconnection("example_port", 1024, SPINNING_CONNECTION);

        int corruption_count = 0;

        // 1. LATENCY STAGE (Verification)
        for(int i = 0; i < (ITERATIONS + 10000); i++) {
            DMessage msg = wait_for_new_message_from_dconnection(conn);
            // Integrity Check
            if (*(long long*)msg.data != (long long)i) corruption_count++;
            
            write_to_dconnection(conn, &msg);
            free(msg.data); 
        }

        // 2. THROUGHPUT STAGE (Flood Verification)
        printf("[SERVER] Latency done. Verifying throughput flood...\n");
        for(int i = 0; i < ITERATIONS; i++) {
            DMessage msg = wait_for_new_message_from_dconnection(conn);
            if (*(long long*)msg.data != (long long)i) {
                corruption_count++;
            }
            free(msg.data);
        }
        
        DMessage ack = { .size = 4, .data = "OK" };
        write_to_dconnection(conn, &ack);

        printf("[SERVER] Integrity Results: %d errors found.\n", corruption_count);
        close_dconnection(conn);
        
    } else if (strcmp(argv[1], "client") == 0) {
        printf("[CLIENT] Connecting to DPort...\n");
        DConnection* conn = connect_dconnection("example_port");
        
        long long counter = 0;
        DMessage msg = { .size = MSG_SIZE, .data = (char*)&counter };
        int errors = 0;

        printf("[CLIENT] Starting Integrity + Latency Test...\n");
        
        // Warmup
        for(int i = 0; i < 10000; i++) {
            counter = (long long)i;
            write_to_dconnection(conn, &msg);
            DMessage r = wait_for_new_message_from_dconnection(conn);
            free(r.data);
        }
        
        long long start_lat = get_nanos();
        for(int i = 0; i < ITERATIONS; i++) {
            counter = (long long)(i + 10000); // Sequence continues
            write_to_dconnection(conn, &msg);
            
            DMessage response = wait_for_new_message_from_dconnection(conn);
            if (*(long long*)response.data != counter) errors++;
            free(response.data);
        }
        long long end_lat = get_nanos();

        printf("[CLIENT] Starting Throughput Flood...\n");
        long long start_tp = get_nanos();
        for(int i = 0; i < ITERATIONS; i++) {
            counter = (long long)i; // Reset for flood test
            write_to_dconnection(conn, &msg);
        }
        DMessage final_ack = wait_for_new_message_from_dconnection(conn);
        long long end_tp = get_nanos();
        free(final_ack.data);

        // --- RESULTS ---
        long long lat_total = end_lat - start_lat;
        long long tp_total = end_tp - start_tp;
        
        printf("\n--- INTEGRITY RESULTS ---\n");
        printf("Integrity Errors: %d\n", errors);
        printf("Latency (RTT):    %lld ns\n", lat_total / ITERATIONS);
        printf("Throughput:       %.2f M/s\n", (double)ITERATIONS / ((double)tp_total / 1e9) / 1e6);
        
        close_dconnection(conn);
    }
    return 0;
}