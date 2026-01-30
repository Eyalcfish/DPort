# dpyport.py
from cffi import FFI
import sys
import os

ffi = FFI()

# 1. Define C signatures
# Note: We use 'signed char' for connection_type so you can pass ints (0, 1, 2)
# without Python complaining about needing bytes.
ffi.cdef("""
    typedef struct DConnection {
        size_t shm_size;
        char identifier;
        char* port_name;
        void* shm_ptr;
        signed char connection_type;
        
        // Platform agnostic pointers (void* fits HANDLE and int*)
        void* hMapFile; 
        void* hEvent;
        int* futex_flag;
    } DConnection;

    typedef struct DMessage {
        size_t size;
        void* data;
    } DMessage;

    // Standard functions
    DConnection* create_dconnection(const char* port_name, size_t shm_size, signed char connection_type);
    DConnection* connect_dconnection(const char* port_name);
    void close_dconnection(DConnection* conn);
    void write_to_dconnection(DConnection* conn, DMessage* msg);
    
    // CRITICAL UPDATE: 
    // We bind to the '_safe' function we added to the DLL.
    // This takes a pointer (DMessage*) instead of returning DMessage by value.
    void wait_for_new_message_safe(DConnection* conn, DMessage* out_msg);
    
    // Memory cleanup helper
    void free_dport_memory(void* ptr); 
""")

# 2. Robust DLL Loading
here = os.path.dirname(os.path.abspath(__file__))
if os.name == 'nt':
    dll_path = os.path.join(here, "dport.dll")
else:
    # Fallback for Linux
    dll_path = os.path.join(here, "libdport.so")

try:
    lib = ffi.dlopen(dll_path)
except OSError as e:
    print(f"CRITICAL ERROR: Could not load DLL at {dll_path}")
    print(f"Details: {e}")
    sys.exit(1)

# 3. Constants
EVENT_BASED_CONNECTION = 0
HYBRID_CONNECTION = 1
SPINNING_CONNECTION = 2

# 4. Helpers
def free_msg_data(msg_struct):
    """Safely frees the data pointer inside a DMessage struct."""
    if msg_struct.data != ffi.NULL:
        lib.free_dport_memory(msg_struct.data)