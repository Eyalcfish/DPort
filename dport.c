#include "dport.h"
#ifdef _WIN32
#include <windows.h>
#endif

// typedef struct DConnection {
//     size_t shm_size;
//     char* port_name;
//     void* shm_ptr;
// } DConnection;

// typedef struct DMessage {
//     size_t size;
//     void* data;
// } DMessage;

/*Create a shared memory connection at id/port <port> and size <shm_size> */
DConnection* create_dconnection(const char* port_name, size_t shm_size, char connection_type) {
    DConnection* conn = (DConnection*)malloc(sizeof(DConnection));

    conn->port_name = strdup(port_name);
    conn->shm_size = shm_size;
    conn->connection_type = connection_type;
    shm_size += sizeof(DConnectionHeader);

    #ifdef _WIN32
    
    HANDLE hMapFile = CreateFileMapping(
        INVALID_HANDLE_VALUE,
        NULL,
        PAGE_READWRITE,
        0,
        (DWORD)shm_size,
        port_name
    );
    
    if (hMapFile == NULL) {
        printf(TEXT("Could not create a connnection(Prob outta memory)\n"), GetLastError());
    }
    
    conn->shm_ptr = (void*) MapViewOfFile(hMapFile, FILE_MAP_ALL_ACCESS, 0, 0, shm_size);
    
    if (conn->shm_ptr == NULL) {
        printf(TEXT("Could not map view of file (%d).\n"), GetLastError());
        
        CloseHandle(hMapFile);
    }

    conn->hMapFile = hMapFile;

    char event_name[256];
    snprintf(event_name, sizeof(event_name), "%s_event", port_name);

    conn->hEvent = CreateEvent(NULL, FALSE, FALSE, event_name);
    #endif

    *((DConnectionHeader*)conn->shm_ptr) = (DConnectionHeader){.shm_size = conn->shm_size, .connection_type = conn->connection_type, .ready_flag_client = 1, .ready_flag_server = 1};
    conn->shm_ptr += sizeof(DConnectionHeader);

    conn->identifier = 1;

    return conn;
}

/*Connect to an existing shared memory connection at id/port <port> */
DConnection* connect_dconnection(const char* port_name) {
    DConnection* conn = (DConnection*)malloc(sizeof(DConnection));
    conn->port_name = strdup(port_name);

    #ifdef _WIN32
    HANDLE hMapFile = OpenFileMapping(
        FILE_MAP_ALL_ACCESS,
        FALSE,
        port_name
    );

    if (hMapFile == NULL) {
        printf(TEXT("Could not open file mapping object (%d).\n"), GetLastError());
    }

    conn->shm_ptr = (void*) MapViewOfFile(hMapFile, FILE_MAP_ALL_ACCESS, 0, 0, 0);
    
    if (conn->shm_ptr == NULL) {
        printf(TEXT("Could not map view of file (%d).\n"), GetLastError());
        CloseHandle(hMapFile);
    }
    
    conn->hMapFile = hMapFile;
    
    char event_name[256];
    snprintf(event_name, sizeof(event_name), "%s_event", port_name);
    
    conn->hEvent = OpenEvent(EVENT_ALL_ACCESS, FALSE, event_name);
    
    #endif
    
    conn->shm_size = ((DConnectionHeader*)conn->shm_ptr)->shm_size;
    conn->connection_type = ((DConnectionHeader*)conn->shm_ptr)->connection_type;
    conn->shm_ptr += sizeof(DConnectionHeader);

    conn->identifier = 0;

    return conn;
}

/*Close the shared memory connection and free resources */
void close_dconnection(DConnection* conn) {
    #ifdef _WIN32
    UnmapViewOfFile((char*)conn->shm_ptr - sizeof(DConnectionHeader));
    CloseHandle(conn->hMapFile);
    #endif
}

/*Write data to the shared memory connection */
void write_to_dconnection(DConnection* conn, DMessage* msg) {
    if (msg->size > conn->shm_size) {
        printf("Message size exceeds shared memory size\n");
        return;
    }

    DConnectionHeader* conn_header = ((DConnectionHeader*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)));
    unsigned char* flag_ptr = (conn->identifier == 0) ? &conn_header->ready_flag_server : &conn_header->ready_flag_client;
    
    
    while (!*flag_ptr || *flag_ptr == 2) { // FOR NOW UNTILL SEPERATE THREAD QUEING IS IMPLEMENTED
        _mm_pause();
    }
    
    *((size_t*)conn->shm_ptr) = msg->size;
    memcpy((char*)conn->shm_ptr+sizeof(size_t), msg->data, msg->size);
    
    __sync_synchronize();
    __sync_lock_test_and_set(flag_ptr, 0);
    
    if (!conn->connection_type) {
        SetEvent(conn->hEvent);
    }
    
    #ifdef _WIN32
    #endif
}

/*Read data from the shared memory connection */
DMessage wait_for_new_message_from_dconnection(DConnection* conn) {
    
    DConnectionHeader* conn_header = ((DConnectionHeader*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)));
    unsigned char* flag_ptr = (conn->identifier == 0) ? &conn_header->ready_flag_client : &conn_header->ready_flag_server;
    
    if (conn->connection_type == SPINNING_CONNECTION) {
        while (*flag_ptr) {
            _mm_pause();
        }
    } else {
        if (conn->connection_type == HYBRID_CONNECTION) {
            for(int i = 0 ; i < 100000 && *flag_ptr; i++) {
                _mm_pause();
            }
        }
        #ifdef _WIN32
        while (*flag_ptr) {
            // Use a timeout (e.g. 10ms) to "self-heal" if a signal is missed
            WaitForSingleObject(conn->hEvent, 1);
        }
        #endif
    }

    DMessage msg;
    msg.size = *(size_t*)conn->shm_ptr;

    msg.data = malloc(msg.size);
    memcpy(msg.data, (char*)conn->shm_ptr+sizeof(size_t), msg.size);
    __sync_synchronize();

    __sync_lock_test_and_set(flag_ptr, 1);
 
    return msg;
}

void* get_pointer_for_new_message_dconnection(DConnection* conn, size_t msg_size) {
    DConnectionHeader* conn_header = ((DConnectionHeader*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)));
    unsigned char* flag_ptr = (conn->identifier == 0) ? &conn_header->ready_flag_server : &conn_header->ready_flag_client;

    while (!*flag_ptr || *flag_ptr == 2) { // FOR NOW UNTILL SEPERATE THREAD QUEING IS IMPLEMENTED
        _mm_pause();
    }

    __sync_lock_test_and_set(flag_ptr, 2); // lock writing for all other writers

    *((size_t*)conn->shm_ptr) = msg_size;


    return conn->shm_ptr+sizeof(size_t);
}

void publish_new_message_dconnection(DConnection* conn) {
    DConnectionHeader* conn_header = ((DConnectionHeader*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)));
    unsigned char* flag_ptr = (conn->identifier == 0) ? &conn_header->ready_flag_server : &conn_header->ready_flag_client;

    __sync_synchronize();
    __sync_lock_test_and_set(flag_ptr, 0);
    
    if (conn->connection_type != SPINNING_CONNECTION) {
        SetEvent(conn->hEvent);
    }
}