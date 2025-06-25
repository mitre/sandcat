#include "dllmain.h"

DWORD __stdcall DllMainThreadFunc(LPVOID lpThreadParameter) {
    VoidFunc();
    return 0;
}

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved) {
    switch(fdwReason) {
        case DLL_PROCESS_ATTACH:
            // Initialize once for each new process.
            // Return FALSE to fail DLL load.
            pThreadParams = (DllMainThreadParams*)malloc(sizeof(DllMainThreadParams));
            if (pThreadParams == NULL) {
                return FALSE;
            }
            pThreadParams->hinstDLL = hinstDLL;
            pThreadParams->fdwReason = fdwReason;
            pThreadParams->lpReserved = lpvReserved;

            HANDLE hGoThread = CreateThread(NULL, 0, DllMainThreadFunc, pThreadParams, 0, NULL);
            if (hGoThread == NULL) {
                return FALSE;
            }

            break;

        case DLL_THREAD_ATTACH:
            // Do thread-specific initialization.
            break;

        case DLL_THREAD_DETACH:
            // Do thread-specific cleanup.
            break;

        case DLL_PROCESS_DETACH:

            if (lpvReserved != NULL)
            {
                break; // do not do cleanup if process termination scenario
            }
            // Perform any necessary cleanup.
            break;
    }
    return TRUE;
}
