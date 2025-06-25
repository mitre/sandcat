#ifndef SANDCAT_DLL_MAIN
#define SANDCAT_DLL_MAIN

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

// https://learn.microsoft.com/en-us/windows/win32/dlls/dllmain
typedef struct {
    HINSTANCE hinstDLL;
    DWORD fdwReason;
    LPVOID lpReserved;
} DllMainThreadParams;

void VoidFunc();

DWORD __stdcall DllMainThreadFunc(LPVOID lpThreadParameter);

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved);

#endif
