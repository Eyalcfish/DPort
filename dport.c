#include "dport.h"
#include <stdlib.h> // for malloc, free

#ifdef _WIN32
    // windows.h and intrin.h are already included via dport.h
#elif __linux__
    #include <immintrin.h>   // for _mm_pause()
    #include <unistd.h>      // for syscall, close, ftruncate
    #include <sys/syscall.h> // for SYS_futex
    #include <linux/futex.h> // for FUTEX_WAIT, FUTEX_WAKE
    #include <errno.h>       // for errno, EWOULDBLOCK
    #include <string.h>      // for memcpy, strdup
    #include <stdint.h>      // for int32_t
    #include <sys/mman.h>    // for mmap, shm_open
    #include <fcntl.h>       // for O_CREAT, O_RDWR
#endif

typedef struct __attribute__((packed)) DConnectionHeader {
    size_t shm_size;
    char connection_type;
    unsigned char ready_flag_server;
    unsigned char ready_flag_client;
    #ifdef _WIN32
    #elif __linux__
    int futex_flag;
    #endif
} DConnectionHeader;

// --- IMPLEMENTATIONS ---

DPORT_EXPORT DConnection* create_dconnection(const char* port_name, size_t shm_size, char connection_type) {
    DConnection* conn = (DConnection*)malloc(sizeof(DConnection));

    conn->port_name = strdup(port_name);
    conn->shm_size = shm_size;
    conn->connection_type = connection_type;
    shm_size += sizeof(DConnectionHeader);

    #ifdef _WIN32
    HANDLE hMapFile = CreateFileMappingA(
        INVALID_HANDLE_VALUE,
        NULL,
        PAGE_READWRITE,
        0,
        (DWORD)shm_size,
        port_name
    );
    
    if (hMapFile == NULL) {
        printf("Could not create a connnection (Prob outta memory): %lu\n", GetLastError());
    }
    
    conn->shm_ptr = (void*) MapViewOfFile(hMapFile, FILE_MAP_ALL_ACCESS, 0, 0, 0); // Map entire file
    
    if (conn->shm_ptr == NULL) {
        printf("Could not map view of file (%lu).\n", GetLastError());
        CloseHandle(hMapFile);
    }

    conn->hMapFile = hMapFile;

    char event_name[256];
    snprintf(event_name, sizeof(event_name), "%s_event", port_name);

    conn->hEvent = CreateEventA(NULL, FALSE, FALSE, event_name);
    #elif __linux__

    int shm_fd = shm_open(port_name, O_CREAT | O_RDWR, 0666);
    ftruncate(shm_fd, shm_size);
    conn->shm_ptr = mmap(0, shm_size, PROT_READ | PROT_WRITE, MAP_SHARED, shm_fd, 0);

    #endif

    #ifdef __linux__
    *((DConnectionHeader*)conn->shm_ptr) = (DConnectionHeader){.shm_size = conn->shm_size, .connection_type = conn->connection_type, .ready_flag_client = 1, .ready_flag_server = 1, .futex_flag = 0}; 
    conn->futex_flag = &((DConnectionHeader*)conn->shm_ptr)->futex_flag;
    #else
    *((DConnectionHeader*)conn->shm_ptr) = (DConnectionHeader){.shm_size = conn->shm_size, .connection_type = conn->connection_type, .ready_flag_client = 1, .ready_flag_server = 1};
    #endif
    conn->shm_ptr = (char*)conn->shm_ptr + sizeof(DConnectionHeader);

    conn->identifier = 1;
    return conn;
}

DPORT_EXPORT DConnection* connect_dconnection(const char* port_name) {
    DConnection* conn = (DConnection*)malloc(sizeof(DConnection));
    conn->port_name = strdup(port_name);

    #ifdef _WIN32
    HANDLE hMapFile = OpenFileMappingA(
        FILE_MAP_ALL_ACCESS,
        FALSE,
        port_name
    );

    if (hMapFile == NULL) {
        printf("Could not open file mapping object (%lu).\n", GetLastError());
    }

    conn->shm_ptr = (void*) MapViewOfFile(hMapFile, FILE_MAP_ALL_ACCESS, 0, 0, 0);
    
    if (conn->shm_ptr == NULL) {
        printf("Could not map view of file (%lu).\n", GetLastError());
        CloseHandle(hMapFile);
    }
    
    conn->hMapFile = hMapFile;
    
    char event_name[256];
    snprintf(event_name, sizeof(event_name), "%s_event", port_name);
    
    conn->hEvent = OpenEventA(EVENT_ALL_ACCESS, FALSE, event_name);

    #elif __linux__

    int shm_fd = shm_open(port_name, O_RDWR, 0666);
    // Map header first to read size
    conn->shm_ptr = ((DConnectionHeader*)mmap(0, sizeof(DConnectionHeader), PROT_READ | PROT_WRITE, MAP_SHARED, shm_fd, 0));
    size_t shm_size = ((DConnectionHeader*)conn->shm_ptr)->shm_size + sizeof(DConnectionHeader);
    munmap((void*)((DConnectionHeader*)conn->shm_ptr), sizeof(DConnectionHeader));
    // Remap full size
    conn->shm_ptr = mmap(0, shm_size, PROT_READ | PROT_WRITE, MAP_SHARED, shm_fd, 0);
    conn->futex_flag = &((DConnectionHeader*)conn->shm_ptr)->futex_flag;

    #endif
    
    conn->shm_size = ((DConnectionHeader*)conn->shm_ptr)->shm_size;
    conn->connection_type = ((DConnectionHeader*)conn->shm_ptr)->connection_type;
    conn->shm_ptr = (char*)conn->shm_ptr + sizeof(DConnectionHeader);

    conn->identifier = 0;
    return conn;
}

DPORT_EXPORT void close_dconnection(DConnection* conn) {
    #ifdef _WIN32
    UnmapViewOfFile((char*)conn->shm_ptr - sizeof(DConnectionHeader));
    CloseHandle(conn->hMapFile);
    CloseHandle(conn->hEvent); // Don't forget to close the event handle
    #elif __linux__
    munmap((void*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)), conn->shm_size + sizeof(DConnectionHeader));
    if (conn->identifier == 1) {
        shm_unlink(conn->port_name);
    }
    #endif
    free(conn->port_name);
    free(conn);
}

DPORT_EXPORT void write_to_dconnection(DConnection* conn, DMessage* msg) {
    if (msg->size > conn->shm_size) {
        printf("Message size exceeds shared memory size\n");
        return;
    }

    DConnectionHeader* conn_header = ((DConnectionHeader*)((char*)conn->shm_ptr - sizeof(DConnectionHeader)));
    unsigned char* flag_ptr = (conn->identifier == 0) ? &conn_header->ready_flag_server : &conn_header->ready_flag_client;
    
    while (!*flag_ptr) { 
        _mm_pause();
    }

    __sync_synchronize();
    
    *((size_t*)conn->shm_ptr) = msg->size;
    memcpy((char*)conn->shm_ptr+sizeof(size_t), msg->data, msg->size);
    
    __sync_synchronize();
    __sync_lock_test_and_set(flag_ptr, 0);
    
    if (!conn->connection_type) {
        #ifdef _WIN32
        SetEvent(conn->hEvent);
        #elif __linux__
        __sync_fetch_and_add(conn->futex_flag, 1);
        syscall(SYS_futex, conn->futex_flag, FUTEX_WAKE, 1, NULL, NULL, 0);
        #endif
    }
}

// --- ORIGINAL FUNCTION (Returns Struct - Safe for C, Crash for Python) ---
DPORT_EXPORT DMessage wait_for_new_message_from_dconnection(DConnection* conn) {
    
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
            WaitForSingleObject(conn->hEvent, 1);
        }
        #elif __linux__
        int expected = *conn->futex_flag;
        while (*flag_ptr) {
            int ret = syscall(SYS_futex, conn->futex_flag, FUTEX_WAIT, expected, NULL, NULL, 0);
            if (ret == -1 && errno == EWOULDBLOCK) {
                expected = *conn->futex_flag;
            }
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

// --- NEW HELPER: Safe Wrapper for Python ---
DPORT_EXPORT void wait_for_new_message_safe(DConnection* conn, DMessage* out_msg) {
    // Call the original function
    DMessage temp = wait_for_new_message_from_dconnection(conn);
    // Copy result to the pointer provided by Python
    *out_msg = temp;
}

// --- NEW HELPER: Memory Cleanup ---
DPORT_EXPORT void free_dport_memory(void* ptr) {
    if (ptr) free(ptr);
}