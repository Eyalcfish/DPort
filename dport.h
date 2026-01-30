#ifndef DPORT_H
#define DPORT_H

#include <stdio.h>

// --- 1. EXPORT MACROS & WINDOWS HEADERS ---
#ifdef _WIN32
    #include <windows.h>
    #include <intrin.h> // For _mm_pause on MSVC
    #define DPORT_EXPORT __declspec(dllexport)
#else
    #define DPORT_EXPORT
#endif

// --- 2. PREVENT NAME MANGLING ---
#ifdef __cplusplus
extern "C" {
#endif

#define EVENT_BASED_CONNECTION 0
#define HYBRID_CONNECTION 1
#define SPINNING_CONNECTION 2

typedef struct DConnection {
    size_t shm_size;
    char identifier;
    char* port_name;
    void* shm_ptr;
    char connection_type;
    #ifdef _WIN32
    void* hMapFile; // Using void* is safer for generic headers, but HANDLE works if windows.h is included
    void* hEvent;
    #elif __linux__
    int* futex_flag;
    #endif
} DConnection;

typedef struct DMessage {
    size_t size;
    void* data;
} DMessage;

// --- 3. EXPORTED FUNCTIONS ---

/* Create a shared memory connection */
DPORT_EXPORT DConnection* create_dconnection(const char* port_name, size_t shm_size, char connection_type);

/* Connect to an existing shared memory connection */
DPORT_EXPORT DConnection* connect_dconnection(const char* port_name);

/* Close the shared memory connection and free resources */
DPORT_EXPORT void close_dconnection(DConnection* conn);

/* Write data to the shared memory connection */
DPORT_EXPORT void write_to_dconnection(DConnection* conn, DMessage* msg);

/* Read data (Original C Version - Returns Struct by Value) */
DPORT_EXPORT DMessage wait_for_new_message_from_dconnection(DConnection* conn);

// --- 4. PYTHON/CFFI HELPERS ---

/* Safe wrapper for Python: Takes a pointer instead of returning a struct by value */
DPORT_EXPORT void wait_for_new_message_safe(DConnection* conn, DMessage* out_msg);

/* Helper to free memory allocated by this DLL */
DPORT_EXPORT void free_dport_memory(void* ptr);

#ifdef __cplusplus
}
#endif

#endif