#ifndef DPORT_H
#define DPORT_H
#include <stdio.h>
#ifdef _WIN32
#include <windows.h>
#endif

typedef struct DConnection {
    size_t shm_size;
    char identifier;
    char* port_name;
    void* shm_ptr;
    char spinning_connection;
    #ifdef _WIN32
    HANDLE hMapFile;
    HANDLE hEvent;
    #endif
} DConnection;

typedef struct __attribute__((packed)) DConnectionHeader {
    size_t shm_size;
    char spinning_connection;
    unsigned char ready_flag_server;
    unsigned char ready_flag_client;
} DConnectionHeader;

typedef struct DMessage {
    size_t size;
    void* data;
} DMessage;

/*Create a shared memory connection at id/port <port> and size <shm_size> at most */
DConnection* create_dconnection(const char* port_name, size_t shm_size, char spinning_connection);

/*Connect to an existing shared memory connection at id/port <port> */
DConnection* connect_dconnection(const char* port_name);

/*Close the shared memory connection and free resources */
void close_dconnection(DConnection* conn);

/*Write data to the shared memory connection */
void write_to_dconnection(DConnection* conn, DMessage* msg);

/*Read data from the shared memory connection */
DMessage wait_for_new_message_from_dconnection(DConnection* conn);

#endif
